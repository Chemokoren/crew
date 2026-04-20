package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple in-memory sliding window rate limiter.
// For production multi-instance deployments, replace with Redis-based limiter.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // max requests
	window   time.Duration // per window
}

type visitor struct {
	timestamps []time.Time
}

// NewRateLimiter creates a rate limiter allowing `rate` requests per `window`.
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup goroutine — evict stale visitors every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rl.window)
	for key, v := range rl.visitors {
		alive := v.timestamps[:0]
		for _, ts := range v.timestamps {
			if ts.After(cutoff) {
				alive = append(alive, ts)
			}
		}
		if len(alive) == 0 {
			delete(rl.visitors, key)
		} else {
			v.timestamps = alive
		}
	}
}

// Allow checks if the given key (IP or user) is within rate limits.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{timestamps: []time.Time{now}}
		return true
	}

	// Prune old timestamps
	alive := v.timestamps[:0]
	for _, ts := range v.timestamps {
		if ts.After(cutoff) {
			alive = append(alive, ts)
		}
	}
	v.timestamps = alive

	if len(v.timestamps) >= rl.rate {
		return false
	}

	v.timestamps = append(v.timestamps, now)
	return true
}

// RateLimit returns Gin middleware that rate-limits by client IP.
func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}
		c.Next()
	}
}

// Timeout returns middleware that sets a request processing deadline.
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a deadline header for downstream services
		c.Header("X-Timeout-Ms", time.Now().Add(duration).Format(time.RFC3339Nano))

		// Gin doesn't natively support context timeout without breaking streaming,
		// so we track start time and check after handler returns
		start := time.Now()
		c.Next()
		elapsed := time.Since(start)
		if elapsed > duration {
			c.Header("X-Timeout-Exceeded", elapsed.String())
		}
	}
}

// SecureHeaders adds security headers to every response.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}
