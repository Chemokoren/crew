package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics tracks basic request/response metrics in-memory.
// For production, wire these into Prometheus via prometheus/client_golang.
type Metrics struct {
	TotalRequests    int64
	TotalErrors      int64
	ActiveRequests   int64
	RequestDurations map[string][]time.Duration // path -> durations
}

var globalMetrics = &Metrics{
	RequestDurations: make(map[string][]time.Duration),
}

// MetricsMiddleware tracks request count, duration, and active connections.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		globalMetrics.ActiveRequests++

		c.Next()

		duration := time.Since(start)
		globalMetrics.ActiveRequests--
		globalMetrics.TotalRequests++

		status := c.Writer.Status()
		if status >= 400 {
			globalMetrics.TotalErrors++
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		globalMetrics.RequestDurations[path] = append(
			globalMetrics.RequestDurations[path], duration,
		)

		// Set response headers for observability
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Request-Id", c.GetString(RequestIDKey))
	}
}

// MetricsHandler returns a simple JSON metrics endpoint.
// GET /metrics
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pathStats := make(map[string]interface{})
		for path, durations := range globalMetrics.RequestDurations {
			if len(durations) == 0 {
				continue
			}
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))
			pathStats[path] = gin.H{
				"count":    len(durations),
				"avg_ms":   float64(avg.Microseconds()) / 1000.0,
				"total_ms": float64(total.Microseconds()) / 1000.0,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"total_requests":  globalMetrics.TotalRequests,
			"total_errors":    globalMetrics.TotalErrors,
			"active_requests": globalMetrics.ActiveRequests,
			"paths":           pathStats,
			"uptime":          time.Since(startTime).String(),
		})
	}
}

var startTime = time.Now()

// CORS adds permissive CORS headers for development.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Idempotency-Key")
		c.Header("Access-Control-Expose-Headers", "X-Request-Id, X-Response-Time")
		c.Header("Access-Control-Max-Age", strconv.Itoa(86400))

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
