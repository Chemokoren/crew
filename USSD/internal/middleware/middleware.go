// Package middleware provides HTTP middleware for the USSD gateway.
// Includes rate limiting, input sanitization, idempotency, and observability.
package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// --- Request ID ---

const RequestIDKey = "X-Request-ID"

// RequestID injects a unique request ID into every request context.
// Uses crypto/rand for uniqueness even under extreme concurrency.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDKey)
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = "ussd-" + hex.EncodeToString(b)
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDKey, id)
		c.Next()
	}
}

// --- Rate Limiting (per MSISDN + global) ---

// RateLimitPerMSISDN enforces per-phone-number rate limiting using Redis.
// Uses a Redis pipeline to make INCR+EXPIRE atomic (single round-trip),
// preventing the TOCTOU race where a crash between INCR and EXPIRE would
// leave a key without TTL — permanently blocking the MSISDN.
func RateLimitPerMSISDN(redisClient *redis.Client, maxPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract MSISDN from request body (best-effort)
		msisdn := extractMSISDN(c)
		if msisdn == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("ussd:ratelimit:%s", msisdn)
		ctx := c.Request.Context()

		// Atomic pipeline: INCR + EXPIRE in one round-trip
		pipe := redisClient.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Minute)
		_, err := pipe.Exec(ctx)
		if err != nil {
			// On Redis failure, allow the request (fail open)
			slog.Warn("rate limit redis error", slog.String("error", err.Error()))
			c.Next()
			return
		}

		count := incrCmd.Val()
		if count > int64(maxPerMinute) {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded for this phone number",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractMSISDN attempts to extract the phone number from the request.
func extractMSISDN(c *gin.Context) string {
	// Africa's Talking format (form-encoded)
	if phone := c.PostForm("phoneNumber"); phone != "" {
		return phone
	}
	// Generic format — we can't peek into JSON body without consuming it
	// Rate limiting for JSON gateway happens at the handler level
	return ""
}

// --- Input Sanitization ---

// SanitizeInput validates and sanitizes USSD inputs to prevent injection attacks.
func SanitizeInput(maxLength int) gin.HandlerFunc {
	// Only allow safe characters in USSD input
	safePattern := regexp.MustCompile(`^[0-9a-zA-Z\s\.\*\#\+\-\_]*$`)

	return func(c *gin.Context) {
		// Check text field (AT format)
		text := c.PostForm("text")
		if text != "" {
			if len(text) > maxLength {
				c.JSON(http.StatusBadRequest, gin.H{"error": "input too long"})
				c.Abort()
				return
			}
			if !safePattern.MatchString(text) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid characters in input"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// --- Idempotency ---

// Idempotency detects duplicate requests from telco retries using request hashing.
// When a telco resends a request, we return the cached response instead of
// reprocessing, preventing double-debits and duplicate registrations.
func Idempotency(redisClient *redis.Client, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.PostForm("sessionId")
		text := c.PostForm("text")

		if sessionID == "" {
			c.Next()
			return
		}

		// Generate request hash
		hash := generateRequestHash(sessionID, text)
		key := fmt.Sprintf("ussd:idempotency:%s", hash)
		ctx := c.Request.Context()

		// Check for cached response
		cached, err := redisClient.Get(ctx, key).Result()
		if err == nil && cached != "" {
			// Duplicate request — return cached response
			slog.Info("idempotent replay",
				slog.String("session_id", sessionID),
				slog.String("hash", hash),
			)
			c.Header("Content-Type", "text/plain")
			c.String(http.StatusOK, cached)
			c.Abort()
			return
		}

		// Store the hash for deduplication (will be updated with response after processing)
		c.Set("idempotency_key", key)
		c.Set("idempotency_ttl", ttl)
		c.Next()
	}
}

// CacheResponse stores the response for idempotency replay.
func CacheResponse(c *gin.Context, redisClient *redis.Client, response string) {
	key, exists := c.Get("idempotency_key")
	if !exists {
		return
	}

	ttl, _ := c.Get("idempotency_ttl")
	ttlDuration, ok := ttl.(time.Duration)
	if !ok {
		ttlDuration = 5 * time.Minute
	}

	ctx := c.Request.Context()
	redisClient.Set(ctx, key.(string), response, ttlDuration)
}

// generateRequestHash creates a deterministic hash for request deduplication.
func generateRequestHash(sessionID, text string) string {
	data := sessionID + "|" + text
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8]) // 16 char hex string
}

// --- Recovery ---

// Recovery catches panics and returns a graceful USSD error message.
// Includes stack trace in logs for production debugging.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())
				logger.Error("USSD handler panic",
					slog.Any("error", err),
					slog.String("path", c.Request.URL.Path),
					slog.String("stack", stack),
				)

				c.Header("Content-Type", "text/plain")
				c.String(http.StatusOK, "END Service temporarily unavailable. Please try again later.")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// MaxBodySize limits the request body size to prevent memory exhaustion.
// USSD payloads are tiny (<200 bytes); 4KB is generous.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// --- Logger ---

// Logger provides structured request logging for USSD interactions.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		// Mask sensitive fields for logging
		sessionID := c.PostForm("sessionId")
		msisdn := maskMSISDN(c.PostForm("phoneNumber"))

		logger.Info("ussd request",
			slog.String("session_id", sessionID),
			slog.String("msisdn", msisdn),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", duration),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
	}
}

// maskMSISDN masks a phone number for safe logging: +254XXXXXX123
func maskMSISDN(phone string) string {
	if len(phone) < 6 {
		return "****"
	}
	return phone[:4] + strings.Repeat("X", len(phone)-7) + phone[len(phone)-3:]
}

// --- Secure Headers ---

// SecureHeaders adds security headers to all responses.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}
