package middleware_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func TestTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Apply 50ms timeout
	router.Use(middleware.Timeout(50 * time.Millisecond))
	
	router.GET("/fast", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	
	router.GET("/slow", func(c *gin.Context) {
		select {
		case <-time.After(100 * time.Millisecond):
			// Didn't time out properly?
			c.Status(http.StatusOK)
		case <-c.Request.Context().Done():
			// Timed out properly!
			c.Status(http.StatusGatewayTimeout)
		}
	})

	// Test fast route
	reqFast := httptest.NewRequest(http.MethodGet, "/fast", nil)
	wFast := httptest.NewRecorder()
	router.ServeHTTP(wFast, reqFast)

	if wFast.Code != http.StatusOK {
		t.Errorf("expected fast route to succeed with 200, got %d", wFast.Code)
	}

	// Test slow route
	reqSlow := httptest.NewRequest(http.MethodGet, "/slow", nil)
	wSlow := httptest.NewRecorder()
	router.ServeHTTP(wSlow, reqSlow)

	if wSlow.Code != http.StatusGatewayTimeout {
		t.Errorf("expected slow route to timeout with 504, got %d", wSlow.Code)
	}

	// Ensure headers are set
	if wSlow.Header().Get("X-Timeout-Ms") == "" {
		t.Errorf("expected X-Timeout-Ms header to be set")
	}
	if wSlow.Header().Get("X-Timeout-Exceeded") == "" {
		t.Errorf("expected X-Timeout-Exceeded header to be set for slow route")
	}
}

func TestSecureHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.Use(middleware.SecureHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Cache-Control":          "no-store, no-cache, must-revalidate",
		"Pragma":                 "no-cache",
	}

	for key, expected := range expectedHeaders {
		if got := w.Header().Get(key); got != expected {
			t.Errorf("expected header %s=%s, got %s", key, expected, got)
		}
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Only run this test if Redis is available on localhost:6379
	opts := &redis.Options{Addr: "localhost:6379"}
	client := redis.NewClient(opts)
	ctx := context.Background()
	
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("skipping rate limit test: redis not running on localhost:6379")
	}
	defer client.Close()

	// Clear out any old keys from previous tests
	client.FlushDB(ctx)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Allow 2 requests per minute
	router.Use(middleware.RateLimit(client, 2, time.Minute))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Request 1: Should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected request 1 to succeed, got %d", w1.Code)
	}

	// Request 2: Should succeed
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected request 2 to succeed, got %d", w2.Code)
	}

	// Request 3: Should be rate limited (429)
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("expected request 3 to be rate limited (429), got %d", w3.Code)
	}
}

func TestMaxBodySizeMiddleware_UnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 1 KB limit
	router.Use(middleware.MaxBodySize(1024))
	router.POST("/test", func(c *gin.Context) {
		body := make([]byte, 512)
		_, err := c.Request.Body.Read(body)
		if err != nil && err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "too large"})
			return
		}
		c.Status(http.StatusOK)
	})

	// 512 bytes — should succeed
	payload := make([]byte, 512)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for body under limit, got %d", w.Code)
	}
}

func TestMaxBodySizeMiddleware_OverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 1 KB limit
	router.Use(middleware.MaxBodySize(1024))
	router.POST("/test", func(c *gin.Context) {
		body := make([]byte, 2048)
		_, err := c.Request.Body.Read(body)
		if err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "too large"})
			return
		}
		c.Status(http.StatusOK)
	})

	// 2 KB — should fail
	payload := make([]byte, 2048)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for body over limit, got %d", w.Code)
	}
}

