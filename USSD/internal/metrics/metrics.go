// Package metrics provides Prometheus instrumentation for the USSD gateway.
// Tracks session lifecycle, latency distribution, telco-specific metrics,
// and drop-off analysis — critical for observability at scale.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// --- Request metrics ---

	// USSDRequestsTotal tracks total USSD requests by gateway and status.
	USSDRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "requests_total",
			Help:      "Total USSD requests by gateway and status",
		},
		[]string{"gateway", "status"},
	)

	// USSDRequestDuration tracks request processing latency.
	// Buckets optimized for USSD: 10ms to 3s (telco timeout threshold).
	USSDRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ussd",
			Name:      "request_duration_seconds",
			Help:      "USSD request processing duration in seconds",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 3.0},
		},
		[]string{"gateway"},
	)

	// --- Session metrics ---

	// USSDSessionsActive tracks currently active sessions (gauge).
	USSDSessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "ussd",
			Name:      "sessions_active",
			Help:      "Number of currently active USSD sessions",
		},
	)

	// USSDSessionsCreated tracks total sessions created.
	USSDSessionsCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "sessions_created_total",
			Help:      "Total USSD sessions created",
		},
	)

	// USSDSessionsCompleted tracks sessions that completed successfully (user reached END).
	USSDSessionsCompleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "sessions_completed_total",
			Help:      "Total USSD sessions completed successfully",
		},
	)

	// USSDSessionsExpired tracks sessions that timed out.
	USSDSessionsExpired = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "sessions_expired_total",
			Help:      "Total USSD sessions that expired/timed out",
		},
	)

	// --- Menu navigation metrics ---

	// USSDMenuStepTotal tracks navigation to each menu step.
	USSDMenuStepTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "menu_step_total",
			Help:      "Total navigations to each menu step",
		},
		[]string{"state"},
	)

	// USSDMenuDropOffTotal tracks where users abandon sessions.
	USSDMenuDropOffTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "menu_dropoff_total",
			Help:      "Total drop-offs per menu state",
		},
		[]string{"state"},
	)

	// --- Backend integration metrics ---

	// USSDBackendCallsTotal tracks backend API calls.
	USSDBackendCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "backend_calls_total",
			Help:      "Total backend API calls by endpoint and status",
		},
		[]string{"endpoint", "status"},
	)

	// USSDBackendLatency tracks backend API call latency.
	USSDBackendLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ussd",
			Name:      "backend_latency_seconds",
			Help:      "Backend API call latency in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 1.5, 2.0},
		},
		[]string{"endpoint"},
	)

	// --- Circuit breaker metrics ---

	// USSDCircuitBreakerState tracks the current circuit breaker state.
	USSDCircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ussd",
			Name:      "circuit_breaker_state",
			Help:      "Current circuit breaker state (0=closed, 1=open, 2=half_open)",
		},
		[]string{"service"},
	)

	// --- Rate limiting metrics ---

	// USSDRateLimitedTotal tracks rate-limited requests.
	USSDRateLimitedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "rate_limited_total",
			Help:      "Total rate-limited USSD requests",
		},
		[]string{"type"}, // "per_msisdn" or "global"
	)

	// --- Idempotency metrics ---

	// USSDIdempotentReplaysTotal tracks idempotent replay responses.
	USSDIdempotentReplaysTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "idempotent_replays_total",
			Help:      "Total idempotent replay responses (duplicate telco requests)",
		},
	)

	// --- Error metrics ---

	// USSDErrorsTotal tracks errors by type.
	USSDErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ussd",
			Name:      "errors_total",
			Help:      "Total USSD errors by type",
		},
		[]string{"type"}, // "parse_error", "backend_error", "session_error", "internal_error"
	)
)

// MetricsMiddleware records request-level metrics.
func MetricsMiddleware(gateway string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())

		USSDRequestsTotal.WithLabelValues(gateway, status).Inc()
		USSDRequestDuration.WithLabelValues(gateway).Observe(duration.Seconds())
	}
}

// MetricsHandler returns the Prometheus metrics endpoint handler.
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordMenuStep records a menu state navigation event.
func RecordMenuStep(state string) {
	USSDMenuStepTotal.WithLabelValues(state).Inc()
}

// RecordDropOff records a session drop-off at a specific state.
func RecordDropOff(state string) {
	USSDMenuDropOffTotal.WithLabelValues(state).Inc()
}

// RecordBackendCall records a backend API call metric.
func RecordBackendCall(endpoint, status string, duration time.Duration) {
	USSDBackendCallsTotal.WithLabelValues(endpoint, status).Inc()
	USSDBackendLatency.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// HealthHandler returns a simple health check endpoint.
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "ussd-gateway",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	}
}
