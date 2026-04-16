package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler provides health and readiness check endpoints.
type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Health returns a basic liveness check.
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "amy-mis",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready returns a deep readiness check (DB + Redis connectivity).
// GET /ready
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	// Check PostgreSQL
	sqlDB, err := h.db.DB()
	if err != nil {
		checks["postgres"] = "error: " + err.Error()
		healthy = false
	} else if err := sqlDB.PingContext(ctx); err != nil {
		checks["postgres"] = "error: " + err.Error()
		healthy = false
	} else {
		checks["postgres"] = "ok"
	}

	// Check Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "error: " + err.Error()
		healthy = false
	} else {
		checks["redis"] = "ok"
	}

	status := http.StatusOK
	statusText := "ready"
	if !healthy {
		status = http.StatusServiceUnavailable
		statusText = "not ready"
	}

	c.JSON(status, gin.H{
		"status": statusText,
		"checks": checks,
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
