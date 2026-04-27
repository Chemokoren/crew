// AMY MIS — Sandbox Financial Service
// A test-mode financial service that mirrors JamboPay's API surface.
// In production, just switch JAMBOPAY_BASE_URL to the real JamboPay endpoint.
//
// Usage:
//   go run cmd/sandbox/main.go
//   # or
//   ./test_sandbox.sh
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

	"github.com/joho/godotenv"
	"github.com/kibsoft/amy-mis/internal/external/sandbox"
)

func main() {
	// --- 1. Structured logging ---
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("starting AMY MIS sandbox financial service...")

	// --- 2. Load .env (optional — for port config) ---
	_ = godotenv.Load()

	port := os.Getenv("SANDBOX_PORT")
	if port == "" {
		port = "8091"
	}

	// --- 3. Initialize sandbox server ---
	server := sandbox.NewServer(logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      server.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// --- 4. Start HTTP server ---
	go func() {
		slog.Info("sandbox financial service started",
			slog.String("port", port),
			slog.String("jambopay_compat", fmt.Sprintf("http://localhost:%s", port)),
			slog.String("admin_panel", fmt.Sprintf("http://localhost:%s/sandbox/admin/stats", port)),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("sandbox server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// --- 5. Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("sandbox shutdown error", slog.String("error", err.Error()))
	}
	slog.Info("sandbox financial service stopped")
}
