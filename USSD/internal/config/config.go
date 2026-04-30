// Package config provides configuration management for the USSD gateway.
// All settings are loaded from environment variables with sensible defaults.
// Required values trigger startup failure if missing.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all USSD gateway configuration.
type Config struct {
	// Server
	Port        int    // HTTP port (default: 8090)
	Environment string // development | staging | production

	// Backend API
	BackendBaseURL   string // AMY MIS backend URL (required)
	BackendAPIKey    string // API key for backend communication (required in production)
	BackendTimeoutMs int    // Backend HTTP call timeout in ms (default: 1500)

	// Redis (session store)
	RedisURL          string // Redis connection string (required)
	RedisPoolSize     int    // Redis connection pool size (default: 100)
	RedisMinIdleConns int    // Redis minimum idle connections (default: 20)

	// Session Management
	SessionTTLSeconds int // USSD session TTL in seconds (default: 180)
	SessionPrefix     string // Redis key prefix for sessions (default: "ussd:session:")

	// Telco Gateway (Strategy Pattern)
	// Providers: "africastalking" | "generic" (extensible — add new adapters)
	PrimaryGateway   string // Primary webhook adapter (default: africastalking)
	FallbackGateway  string // Fallback adapter if primary unavailable (default: generic)
	ATAPIKey         string // Africa's Talking API key
	ATUsername       string // Africa's Talking username
	ATShortCode      string // Africa's Talking shortcode
	ATBaseURL        string // Africa's Talking base URL

	// Rate Limiting
	RateLimitPerMSISDN int // Max requests per MSISDN per minute (default: 30)
	RateLimitGlobalRPM int // Global requests per minute (default: 50000)

	// Circuit Breaker
	CBMaxFailures    int // Max failures before circuit opens (default: 5)
	CBTimeoutSeconds int // Circuit breaker timeout in seconds (default: 30)

	// Observability
	MetricsEnabled bool   // Enable Prometheus metrics (default: true)
	TracingEnabled bool   // Enable distributed tracing (default: true)
	LogLevel       string // Log level: debug | info | warn | error (default: info)

	// Idempotency
	IdempotencyTTLSeconds int // TTL for idempotency keys in seconds (default: 300)

	// Security
	InputMaxLength       int  // Maximum USSD input length (default: 160)
	SanitizeInput        bool // Enable input sanitization (default: true)
	PINMaskingEnabled    bool // Mask PIN inputs in logs (default: true)

	// Localization
	DefaultLanguage      string // Default language code (default: "en")
	SupportedLanguages   string // Comma-separated supported languages (default: "en,sw")

	// CORS (for USSD simulator)
	CORSAllowedOrigins string // Comma-separated allowed origins (default: *)
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnvInt("PORT", 8090),
		Environment: getEnv("ENVIRONMENT", "development"),

		BackendBaseURL:   os.Getenv("BACKEND_BASE_URL"),
		BackendAPIKey:    os.Getenv("BACKEND_API_KEY"),
		BackendTimeoutMs: getEnvInt("BACKEND_TIMEOUT_MS", 1500),

		RedisURL:          os.Getenv("REDIS_URL"),
		RedisPoolSize:     getEnvInt("REDIS_POOL_SIZE", 100),
		RedisMinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 20),

		SessionTTLSeconds: getEnvInt("SESSION_TTL_SECONDS", 180),
		SessionPrefix:     getEnv("SESSION_PREFIX", "ussd:session:"),

		PrimaryGateway:  getEnv("PRIMARY_GATEWAY", "africastalking"),
		FallbackGateway: getEnv("FALLBACK_GATEWAY", "generic"),
		ATAPIKey:       os.Getenv("AT_API_KEY"),
		ATUsername:      getEnv("AT_USERNAME", "sandbox"),
		ATShortCode:     os.Getenv("AT_SHORTCODE"),
		ATBaseURL:       getEnv("AT_BASE_URL", "https://api.africastalking.com/version1"),

		RateLimitPerMSISDN: getEnvInt("RATE_LIMIT_PER_MSISDN", 30),
		RateLimitGlobalRPM: getEnvInt("RATE_LIMIT_GLOBAL_RPM", 50000),

		CBMaxFailures:    getEnvInt("CB_MAX_FAILURES", 5),
		CBTimeoutSeconds: getEnvInt("CB_TIMEOUT_SECONDS", 30),

		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
		TracingEnabled: getEnvBool("TRACING_ENABLED", true),
		LogLevel:       getEnv("LOG_LEVEL", "info"),

		IdempotencyTTLSeconds: getEnvInt("IDEMPOTENCY_TTL_SECONDS", 300),

		InputMaxLength:    getEnvInt("INPUT_MAX_LENGTH", 160),
		SanitizeInput:     getEnvBool("SANITIZE_INPUT", true),
		PINMaskingEnabled: getEnvBool("PIN_MASKING_ENABLED", true),

		DefaultLanguage:    getEnv("DEFAULT_LANGUAGE", "en"),
		SupportedLanguages: getEnv("SUPPORTED_LANGUAGES", "en,sw"),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required config values are present.
func (c *Config) validate() error {
	if c.RedisURL == "" {
		return fmt.Errorf("config: REDIS_URL is required")
	}
	if c.BackendBaseURL == "" {
		return fmt.Errorf("config: BACKEND_BASE_URL is required")
	}
	if c.Environment == "production" {
		if c.BackendAPIKey == "" {
			return fmt.Errorf("config: BACKEND_API_KEY is required in production")
		}
		if c.ATAPIKey == "" {
			return fmt.Errorf("config: AT_API_KEY is required in production")
		}
	}
	return nil
}

// SessionTTL returns the session TTL as a time.Duration.
func (c *Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionTTLSeconds) * time.Second
}

// BackendTimeout returns the backend timeout as a time.Duration.
func (c *Config) BackendTimeout() time.Duration {
	return time.Duration(c.BackendTimeoutMs) * time.Millisecond
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// --- Helpers ---

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
