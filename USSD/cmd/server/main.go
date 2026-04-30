// AMY MIS — USSD Gateway Service
// A telecom-grade USSD gateway for Kenya's informal transport workforce.
// Designed for millions of concurrent sessions with sub-second response times.
//
// Architecture:
//   - Stateless application layer (horizontally scalable)
//   - Redis-backed session store (sub-ms lookup)
//   - FSM-driven menu engine (deterministic state transitions)
//   - Circuit breaker for backend resilience
//   - Per-MSISDN rate limiting
//   - Prometheus observability
//   - Multi-gateway support (Africa's Talking + generic simulator)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/kibsoft/amy-mis-ussd/internal/backend"
	"github.com/kibsoft/amy-mis-ussd/internal/config"
	"github.com/kibsoft/amy-mis-ussd/internal/engine"
	"github.com/kibsoft/amy-mis-ussd/internal/gateway"
	"github.com/kibsoft/amy-mis-ussd/internal/handler"
	"github.com/kibsoft/amy-mis-ussd/internal/i18n"
	"github.com/kibsoft/amy-mis-ussd/internal/metrics"
	"github.com/kibsoft/amy-mis-ussd/internal/middleware"
	"github.com/kibsoft/amy-mis-ussd/internal/session"
)

func main() {
	// --- 1. Structured logging ---
	var logHandler slog.Handler
	logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	slog.Info("starting USSD gateway service...")

	// --- 2. Load configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.IsDevelopment() {
		slog.Info("running in development mode")
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger = slog.New(logHandler)
		slog.SetDefault(logger)
	}

	// --- 3. Connect to Redis ---
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to parse Redis URL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	redisOpts.PoolSize = cfg.RedisPoolSize
	redisOpts.MinIdleConns = cfg.RedisMinIdleConns

	redisClient := redis.NewClient(redisOpts)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to connect to Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("connected to Redis",
		slog.String("addr", redisOpts.Addr),
		slog.Int("pool_size", cfg.RedisPoolSize),
	)

	// --- 4. Initialize session store ---
	sessionStore := session.NewStore(
		redisClient,
		cfg.SessionPrefix,
		cfg.SessionTTL(),
	)
	slog.Info("session store initialized",
		slog.Int("ttl_seconds", cfg.SessionTTLSeconds),
		slog.String("prefix", cfg.SessionPrefix),
	)

	// --- 5. Initialize backend client ---
	backendClient := backend.NewClient(
		cfg.BackendBaseURL,
		cfg.BackendAPIKey,
		cfg.BackendTimeout(),
		logger,
	)
	slog.Info("backend client initialized",
		slog.String("base_url", cfg.BackendBaseURL),
		slog.Int("timeout_ms", cfg.BackendTimeoutMs),
	)

	// --- 6. Initialize i18n translator ---
	translator := i18n.NewTranslator(cfg.DefaultLanguage)
	slog.Info("translator initialized",
		slog.String("default_language", cfg.DefaultLanguage),
		slog.String("supported", cfg.SupportedLanguages),
	)

	// --- 8. Initialize FSM engine ---
	eng := engine.NewEngine(backendClient, sessionStore, translator, logger)
	slog.Info("FSM engine initialized")

	// --- 9. Initialize gateway registry (strategy pattern) ---
	// Register all available gateway adapters. The primary/fallback are
	// selected by config — no code changes needed to switch providers.
	registry := gateway.NewRegistry(logger)
	registry.Register(gateway.NewAfricasTalkingGateway(logger))
	registry.Register(gateway.NewGenericGateway(logger))

	// Set primary and fallback from config
	if err := registry.SetPrimary(cfg.PrimaryGateway); err != nil {
		slog.Error("invalid PRIMARY_GATEWAY", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := registry.SetFallback(cfg.FallbackGateway); err != nil {
		slog.Error("invalid FALLBACK_GATEWAY", slog.String("error", err.Error()))
		os.Exit(1)
	}

	primaryGW := registry.Primary()
	simulatorGW := registry.Get("generic") // Always available for dev/testing

	slog.Info("gateway strategy configured",
		slog.String("primary", primaryGW.Name()),
		slog.String("fallback", cfg.FallbackGateway),
		slog.Any("registered", registry.Names()),
	)

	// --- 10. Initialize handler ---
	ussdHandler := handler.NewUSSDHandler(
		eng,
		sessionStore,
		redisClient,
		logger,
		cfg.DefaultLanguage,
	)

	// --- 11. Setup Gin router ---
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.SecureHeaders())
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.MaxBodySize(4096)) // 4KB — USSD payloads are < 200 bytes
	router.Use(middleware.Logger(logger))

	// --- 12. Register routes ---

	// Health & metrics (no auth, no rate limiting)
	router.GET("/health", metrics.HealthHandler())
	router.GET("/metrics", metrics.MetricsHandler())

	// USSD endpoints — provider-agnostic naming
	ussd := router.Group("/ussd")
	{
		// Production webhook — receives requests from whichever telco is configured
		// as primary (Africa's Talking, generic, or future providers).
		// POST /ussd/webhook
		webhook := ussd.Group("/webhook")
		webhook.Use(middleware.SanitizeInput(cfg.InputMaxLength))
		webhook.Use(middleware.RateLimitPerMSISDN(redisClient, cfg.RateLimitPerMSISDN))
		webhook.Use(middleware.Idempotency(redisClient, time.Duration(cfg.IdempotencyTTLSeconds)*time.Second))
		webhook.Use(metrics.MetricsMiddleware(primaryGW.Name()))
		{
			webhook.POST("", ussdHandler.Handle(primaryGW))
		}

		// Development/testing simulator — always uses generic JSON format.
		// POST /ussd/simulator
		sim := ussd.Group("/simulator")
		sim.Use(metrics.MetricsMiddleware("simulator"))
		{
			sim.POST("", ussdHandler.Handle(simulatorGW))
		}
	}

	// --- 13. Start HTTP server ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		slog.Info("USSD gateway started",
			slog.Int("port", cfg.Port),
			slog.String("env", cfg.Environment),
			slog.String("primary_gateway", cfg.PrimaryGateway),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// --- 14. Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", slog.String("signal", sig.String()))

	// Drain in-flight requests (5s for USSD — sessions are short-lived)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	slog.Info("HTTP server stopped")

	// Close Redis
	if err := redisClient.Close(); err != nil {
		slog.Error("Redis close error", slog.String("error", err.Error()))
	}
	slog.Info("Redis connection closed")

	slog.Info("USSD gateway shutdown complete")
}
