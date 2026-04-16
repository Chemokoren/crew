package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
// All required fields will cause startup failure if missing.
type Config struct {
	// Server
	Port        int    // HTTP port (default: 8080)
	Environment string // development | staging | production

	// Database
	DatabaseURL    string // PostgreSQL connection string (required)
	MigrationsPath string // Path to migration files (default: ./migrations)

	// Redis
	RedisURL string // Redis connection string (required)

	// JWT
	JWTSecret        string // JWT signing secret (required)
	JWTExpiryMinutes int    // Access token TTL in minutes (default: 15)
	JWTRefreshDays   int    // Refresh token TTL in days (default: 7)

	// MinIO (S3-compatible file storage)
	MinIOEndpoint  string // MinIO server endpoint (required)
	MinIOAccessKey string // MinIO access key (required)
	MinIOSecretKey string // MinIO secret key (required)
	MinIOBucket    string // MinIO bucket name (default: amy-mis)
	MinIOUseSSL    bool   // Use SSL for MinIO (default: false)

	// External APIs
	JamboPayAPIKey  string // JamboPay API key
	JamboPayBaseURL string // JamboPay API base URL
	PerpayAPIKey    string // Perpay API key
	PerpayBaseURL   string // Perpay API base URL
	IPRSAPIKey      string // IPRS API key
	IPRSBaseURL     string // IPRS API base URL
	ATAPIKey        string // Africa's Talking API key
	ATUsername      string // Africa's Talking username
	ATShortCode     string // Africa's Talking USSD short code

	// Rate Limiting
	RateLimitRPM int // Requests per minute per IP (default: 100)

	// CORS
	CORSAllowedOrigins string // Comma-separated allowed origins (default: *)
}

// Load reads configuration from environment variables.
// It fails fast on missing required values.
// In development, it auto-loads from .env file if present.
func Load() (*Config, error) {
	// Auto-load .env file (silently ignored if not present)
	_ = godotenv.Load()

	cfg := &Config{
		Port:             getEnvInt("PORT", 8080),
		Environment:      getEnv("ENVIRONMENT", "development"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		MigrationsPath:   getEnv("MIGRATIONS_PATH", "./migrations"),
		RedisURL:         os.Getenv("REDIS_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTExpiryMinutes: getEnvInt("JWT_EXPIRY_MINUTES", 15),
		JWTRefreshDays:   getEnvInt("JWT_REFRESH_DAYS", 7),
		MinIOEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:      getEnv("MINIO_BUCKET", "amy-mis"),
		MinIOUseSSL:      getEnvBool("MINIO_USE_SSL", false),
		JamboPayAPIKey:   os.Getenv("JAMBOPAY_API_KEY"),
		JamboPayBaseURL:  os.Getenv("JAMBOPAY_BASE_URL"),
		PerpayAPIKey:     os.Getenv("PERPAY_API_KEY"),
		PerpayBaseURL:    os.Getenv("PERPAY_BASE_URL"),
		IPRSAPIKey:       os.Getenv("IPRS_API_KEY"),
		IPRSBaseURL:      os.Getenv("IPRS_BASE_URL"),
		ATAPIKey:         os.Getenv("AT_API_KEY"),
		ATUsername:        os.Getenv("AT_USERNAME"),
		ATShortCode:      os.Getenv("AT_SHORTCODE"),
		RateLimitRPM:     getEnvInt("RATE_LIMIT_RPM", 100),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required config values are present.
func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("config: REDIS_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("config: JWT_SECRET must be at least 32 characters (got %d)", len(c.JWTSecret))
	}

	// MinIO is required in all environments
	if c.MinIOEndpoint == "" {
		return fmt.Errorf("config: MINIO_ENDPOINT is required")
	}
	if c.MinIOAccessKey == "" {
		return fmt.Errorf("config: MINIO_ACCESS_KEY is required")
	}
	if c.MinIOSecretKey == "" {
		return fmt.Errorf("config: MINIO_SECRET_KEY is required")
	}

	// External APIs are optional in development
	if c.Environment == "production" {
		if c.JamboPayAPIKey == "" {
			return fmt.Errorf("config: JAMBOPAY_API_KEY is required in production")
		}
		if c.ATAPIKey == "" {
			return fmt.Errorf("config: AT_API_KEY is required in production")
		}
	}

	return nil
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
