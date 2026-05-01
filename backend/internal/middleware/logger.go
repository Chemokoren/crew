package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

// RequestID adds a unique request ID to each request context and response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Logger logs each HTTP request with structured fields.
// For 4xx/5xx responses, captures and logs the error message from the response body.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Wrap the response writer to capture the body for error responses
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		}

		if query != "" {
			attrs = append(attrs, slog.String("query", query))
		}

		if requestID, exists := c.Get(RequestIDKey); exists {
			attrs = append(attrs, slog.String("request_id", requestID.(string)))
		}

		// For error responses, extract the error message from the response body
		if status >= 400 {
			if errMsg := extractErrorMessage(blw.body.Bytes()); errMsg != "" {
				attrs = append(attrs, slog.String("error", errMsg))
			}
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("gin_errors", c.Errors.String()))
		}

		msg := "HTTP Request"
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		logger.LogAttrs(c.Request.Context(), level, msg, attrs...)
	}
}

// bodyLogWriter wraps gin.ResponseWriter to capture response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b) // capture a copy
	return w.ResponseWriter.Write(b)
}

// extractErrorMessage tries to pull "error.message" from a JSON error response body.
func extractErrorMessage(body []byte) string {
	if len(body) == 0 || body[0] != '{' {
		return ""
	}
	var resp struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
		Message string `json:"message"` // fallback: some endpoints use top-level message
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return ""
	}
	if resp.Error.Message != "" {
		if resp.Error.Code != "" {
			return resp.Error.Code + ": " + resp.Error.Message
		}
		return resp.Error.Message
	}
	return resp.Message
}

// Recovery recovers from panics and returns a 500 error.
// Logs the full stack trace for debugging.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace
				stack := string(debug.Stack())

				requestID, _ := c.Get(RequestIDKey)
				logger.Error("panic recovered",
					slog.Any("error", err),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.Any("request_id", requestID),
					slog.String("stack", stack),
				)

				c.AbortWithStatusJSON(500, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An unexpected error occurred",
					},
				})
			}
		}()
		c.Next()
	}
}
