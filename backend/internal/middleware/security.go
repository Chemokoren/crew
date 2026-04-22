package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// rateLimitScript atomically increments and sets expiry only on key creation (fixed window).
var rateLimitScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)

// RateLimit returns Gin middleware that rate-limits by client IP using Redis.
// Uses a Lua script to ensure atomic fixed-window behavior.
func RateLimit(redisClient *redis.Client, rate int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rate_limit:" + c.ClientIP()
		ctx := c.Request.Context()

		windowSec := int(window.Seconds())
		count, err := rateLimitScript.Run(ctx, redisClient, []string{key}, windowSec).Int64()

		if err != nil {
			// Fail open if Redis is down
			c.Next()
			return
		}

		if count > int64(rate) {
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

// Timeout returns middleware that sets a request processing deadline using context.WithTimeout.
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a deadline header for downstream services
		c.Header("X-Timeout-Ms", time.Now().Add(duration).Format(time.RFC3339Nano))

		// Create a timeout context
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		// Replace the request's context with the timeout context
		c.Request = c.Request.WithContext(ctx)

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
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}
