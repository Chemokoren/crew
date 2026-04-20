// AMY MIS — Backend Server
// A Workforce Financial Operating System for Informal Economies
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
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/external/storage"
	"github.com/kibsoft/amy-mis/internal/handler"
	"github.com/kibsoft/amy-mis/internal/middleware"
	pgRepo "github.com/kibsoft/amy-mis/internal/repository/postgres"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func main() {
	// --- 1. Setup structured logging ---
	var logHandler slog.Handler
	logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	slog.Info("starting AMY MIS server...")

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

	// --- 3. Connect to PostgreSQL ---
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --- 4. Run database migrations ---
	if err := database.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		slog.Error("failed to run database migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --- 5. Connect to Redis ---
	redisClient, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --- 6. Connect to MinIO ---
	minioClient, err := storage.NewMinIOClient(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Error("failed to connect to MinIO", slog.String("error", err.Error()))
		os.Exit(1)
	}
	_ = minioClient // Injected into document handlers in a future phase

	// --- 7. Initialize repositories ---
	userRepo := pgRepo.NewUserRepo(db)
	crewRepo := pgRepo.NewCrewRepo(db)
	walletRepo := pgRepo.NewWalletRepo(db)
	assignmentRepo := pgRepo.NewAssignmentRepo(db)
	earningRepo := pgRepo.NewEarningRepo(db)

	// --- 8. Initialize JWT manager ---
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTExpiryMinutes, cfg.JWTRefreshDays)

	// --- 9. Initialize services ---
	authSvc := service.NewAuthService(userRepo, crewRepo, jwtManager, logger)
	crewSvc := service.NewCrewService(crewRepo, logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, logger)
	assignmentSvc := service.NewAssignmentService(assignmentRepo, earningRepo, walletSvc, logger)

	// --- 10. Initialize handlers ---
	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(authSvc)
	crewHandler := handler.NewCrewHandler(crewSvc)
	walletHandler := handler.NewWalletHandler(walletSvc)
	assignmentHandler := handler.NewAssignmentHandler(assignmentSvc)

	// --- 11. Setup Gin router ---
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.SecureHeaders())
	router.Use(middleware.RequestID())
	router.Use(middleware.RateLimit(100, time.Minute)) // 100 req/min per IP
	router.Use(middleware.Timeout(30 * time.Second))
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))

	// --- 12. Register routes ---

	// Health, readiness, and metrics (no auth)
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", middleware.MetricsHandler())

	// API v1 — public endpoints
	v1 := router.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	// API v1 — authenticated endpoints
	secured := v1.Group("")
	secured.Use(middleware.JWTAuth(jwtManager))
	{
		// Current user
		secured.GET("/auth/me", authHandler.Me)

		// Crew members (SACCO admins & system admins)
		crew := secured.Group("/crew")
		crew.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			crew.POST("", crewHandler.Create)
			crew.GET("", crewHandler.List)
			crew.GET("/:id", crewHandler.GetByID)
			crew.PUT("/:id/kyc", crewHandler.UpdateKYC)
			crew.DELETE("/:id", crewHandler.Deactivate)
		}

		// Assignments
		assignments := secured.Group("/assignments")
		assignments.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			assignments.POST("", assignmentHandler.Create)
			assignments.GET("", assignmentHandler.List)
			assignments.GET("/:id", assignmentHandler.GetByID)
			assignments.POST("/:id/complete", assignmentHandler.Complete)
		}

		// Wallets (system admin only for direct credit/debit; crew can view own)
		wallets := secured.Group("/wallets")
		{
			wallets.GET("/:crew_member_id", walletHandler.GetBalance)
			wallets.GET("/:crew_member_id/transactions", walletHandler.ListTransactions)

			walletAdmin := wallets.Group("")
			walletAdmin.Use(middleware.RequireRole(types.RoleSystemAdmin))
			{
				walletAdmin.POST("/credit", walletHandler.Credit)
				walletAdmin.POST("/debit", walletHandler.Debit)
			}
		}
	}

	// --- 8. Start HTTP server ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP server started",
			slog.Int("port", cfg.Port),
			slog.String("env", cfg.Environment),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// --- 9. Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", slog.String("signal", sig.String()))

	// Create shutdown context with 30s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Stop accepting new HTTP requests, drain in-flight
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	slog.Info("HTTP server stopped")

	// 2. Close Redis
	if err := redisClient.Close(); err != nil {
		slog.Error("Redis close error", slog.String("error", err.Error()))
	}
	slog.Info("Redis connection closed")

	// 3. Close database
	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			slog.Error("database close error", slog.String("error", err.Error()))
		}
	}
	slog.Info("database connection closed")

	slog.Info("AMY MIS server shutdown complete")
}
