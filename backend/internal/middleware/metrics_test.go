package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/middleware"
)

func TestMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add RequestID middleware so X-Request-Id can be tested
	router.Use(middleware.RequestID())
	router.Use(middleware.MetricsMiddleware())

	router.GET("/test-endpoint", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		c.Status(http.StatusCreated)
	})

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/test-endpoint", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify status
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	// Verify headers injected by MetricsMiddleware
	if w.Header().Get("X-Response-Time") == "" {
		t.Errorf("expected X-Response-Time header to be set")
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Errorf("expected X-Request-Id header to be set")
	}
}

func TestMetricsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.Use(middleware.MetricsMiddleware())
	router.GET("/test-endpoint", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/metrics", middleware.MetricsHandler())

	// Hit the endpoint to generate some metrics
	req1 := httptest.NewRequest(http.MethodGet, "/test-endpoint", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Fetch the metrics
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	wMetrics := httptest.NewRecorder()
	router.ServeHTTP(wMetrics, reqMetrics)

	if wMetrics.Code != http.StatusOK {
		t.Fatalf("expected metrics endpoint to return 200, got %d", wMetrics.Code)
	}

	body := wMetrics.Body.String()
	
	// Check for Prometheus specific metric strings
	expectedMetrics := []string{
		`http_requests_total`,
		`http_request_duration_seconds`,
		`http_active_requests`,
		`method="GET"`,
		`path="/test-endpoint"`,
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metrics body to contain %q", metric)
		}
	}
}
