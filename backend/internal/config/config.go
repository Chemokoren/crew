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

	// =============================================
	// INTEGRATION ENABLE/DISABLE & PRIMARY PROVIDER
	// =============================================
	// These flags allow zero-downtime vendor switching.
	// Set *_ENABLED=false to disable a provider without removing credentials.
	// Set *_PRIMARY to choose the active primary provider.
	// Disabled providers are skipped entirely; enabled providers are chained
	// in order: primary first, then remaining as fallbacks.

	// SMS provider configuration
	SMSPrimaryProvider string // Primary SMS provider: "optimize" | "africastalking" (default: optimize)
	SMSOptimizeEnabled bool   // Enable Optimize SMS provider (default: true if credentials present)
	SMSATEnabled       bool   // Enable Africa's Talking SMS provider (default: true if credentials present)

	// Payment/payout provider configuration
	PaymentPrimaryProvider string // Primary payment provider: "jambopay" | "mpesa" (default: jambopay)
	PaymentJamboPayEnabled bool   // Enable JamboPay payment provider (default: true if credentials present)
	PaymentMpesaEnabled    bool   // Enable M-Pesa direct payment provider (default: true if credentials present)

	// Payroll provider configuration
	PayrollPrimaryProvider string // Primary payroll provider: "perpay" (default: perpay)
	PayrollPerpayEnabled   bool   // Enable PerPay payroll provider (default: true if credentials present)

	// Identity/KYC provider configuration
	IdentityPrimaryProvider string // Primary identity provider: "iprs" (default: iprs)
	IdentityIPRSEnabled     bool   // Enable IPRS identity verification (default: true if credentials present)

	// Storage provider configuration
	StoragePrimaryProvider string // Primary storage provider: "minio" | "s3" (default: minio)
	StorageMinIOEnabled    bool   // Enable MinIO storage (default: true if credentials present)

	// Email provider configuration
	EmailPrimaryProvider string // Primary email provider: "gmail" | "sendgrid" (default: gmail)
	EmailGmailEnabled    bool   // Enable Gmail SMTP provider (default: true if credentials present)

	// WhatsApp provider configuration
	WhatsAppPrimaryProvider string // Primary WhatsApp provider: "meta" | "twilio" (default: meta)
	WhatsAppMetaEnabled     bool   // Enable Meta WhatsApp Cloud API (default: true if credentials present)

	// =============================================
	// OTP & 2FA CONFIGURATION
	// =============================================
	// Controls OTP delivery channel and 2FA enforcement.

	OTPDefaultChannel string // Default OTP channel: "email" | "sms" | "whatsapp" (default: email)
	OTPEnabled        bool   // Enable OTP-based password reset (default: true)
	TwoFAEnabled      bool   // Enable 2FA for login (default: true — OTP after password)
	TwoFAEnforced     bool   // Force 2FA for all users (default: false — configurable per user)

	// =============================================
	// SMS — Optimize (default provider)
	// =============================================
	SMSClientID           string // Optimize SMS client ID
	SMSClientSecret       string // Optimize SMS client secret
	SMSTokenURL           string // Optimize SMS OAuth2 token URL
	SMSURL                string // Optimize SMS send endpoint
	SMSSenderID           string // Optimize SMS sender name
	SMSCallbackURL        string // Optimize SMS callback URL
	SMSTokenExpirySec     int    // Optimize SMS token TTL (default: 3600)

	// SMS — Africa's Talking (alternative provider)
	ATAPIKey    string // Africa's Talking API key
	ATUsername  string // Africa's Talking username
	ATShortCode string // Africa's Talking short code
	ATBaseURL   string // Africa's Talking base URL

	// JamboPay (payment/payout)
	JamboPayClientID     string // JamboPay OAuth2 client ID
	JamboPayClientSecret string // JamboPay OAuth2 client secret
	JamboPayBaseURL      string // JamboPay API base URL

	// M-Pesa Direct (alternative payment provider — future)
	MpesaConsumerKey    string // M-Pesa consumer key
	MpesaConsumerSecret string // M-Pesa consumer secret
	MpesaBaseURL        string // M-Pesa API base URL
	MpesaShortCode      string // M-Pesa shortcode
	MpesaPasskey        string // M-Pesa passkey

	// PerPay (payroll)
	PerpayClientID     string // PerPay OAuth2 client ID
	PerpayClientSecret string // PerPay OAuth2 client secret
	PerpayBaseURL      string // PerPay API base URL

	// IPRS (identity verification)
	IPRSClientID          string // IPRS OAuth2 client ID
	IPRSClientSecret      string // IPRS OAuth2 client secret
	IPRSBaseURL           string // IPRS API base URL
	IPRSTokenEndpoint     string // IPRS OAuth2 token endpoint

	// =============================================
	// EMAIL — Gmail SMTP (default provider)
	// =============================================
	EmailHost        string // SMTP host (default: smtp.gmail.com)
	EmailPort        int    // SMTP port (default: 587)
	EmailUseTLS      bool   // Use STARTTLS (default: true)
	EmailHostUser    string // SMTP username / from address
	EmailHostPassword string // SMTP password (app password for Gmail)
	EmailFromAddress string // Default sender address
	EmailFromName    string // Display name (default: AMY MIS)
	EmailSupport     string // Support email address

	// =============================================
	// WHATSAPP — Meta Cloud API (default provider)
	// =============================================
	WhatsAppPhoneNumberID string // WhatsApp Business Phone Number ID
	WhatsAppAccessToken   string // Meta API permanent access token
	WhatsAppAPIVersion    string // API version (default: v18.0)

	// Rate Limiting
	RateLimitRPM int // Requests per minute per IP (default: 100)

	// CORS
	CORSAllowedOrigins string // Comma-separated allowed origins (default: *)

	// Webhook Signature Verification
	WebhookJamboPaySecret string // HMAC-SHA256 secret for JamboPay webhook verification
	WebhookPerpaySecret   string // HMAC-SHA256 secret for PerPay webhook verification

	// =============================================
	// LOAN POLICY — Concurrent Loan Configuration
	// =============================================
	// Controls whether borrowers can hold multiple active loans.
	// See models.LoanConcurrencyPolicy for available options.

	LoanConcurrencyPolicy          string  // SINGLE | PER_CATEGORY | AGGREGATE (default: SINGLE)
	LoanMaxConcurrent              int     // Max concurrent loans under AGGREGATE policy (default: 3)
	LoanAggregateExposureMultiplier float64 // Max total exposure as multiple of tier limit (default: 2.0)
	LoanCategoriesEnabled          string  // Comma-separated enabled categories (default: all)

	// =============================================
	// OPERATIONAL TUNING
	// =============================================

	// Database connection pool
	DBMaxOpenConns    int // Max open DB connections (default: 25)
	DBMaxIdleConns    int // Max idle DB connections (default: 10)
	DBConnMaxLifeMin  int // Max connection lifetime in minutes (default: 5)
	DBConnMaxIdleMin  int // Max idle time in minutes (default: 1)

	// HTTP server settings
	RequestTimeoutSec int   // Global request timeout in seconds (default: 30)
	MaxRequestBodyMB  int   // Maximum request body size in megabytes (default: 10)
	CSVExportMaxRows  int   // Maximum rows for CSV export (default: 10000)
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

		// Integration enable/disable & primary provider selection
		// Defaults: enabled if credentials are present, can be explicitly overridden
		SMSPrimaryProvider:     getEnv("SMS_PRIMARY_PROVIDER", "optimize"),
		SMSOptimizeEnabled:     getEnvBool("SMS_OPTIMIZE_ENABLED", true),
		SMSATEnabled:           getEnvBool("SMS_AT_ENABLED", true),
		PaymentPrimaryProvider: getEnv("PAYMENT_PRIMARY_PROVIDER", "jambopay"),
		PaymentJamboPayEnabled: getEnvBool("PAYMENT_JAMBOPAY_ENABLED", true),
		PaymentMpesaEnabled:    getEnvBool("PAYMENT_MPESA_ENABLED", false),
		PayrollPrimaryProvider: getEnv("PAYROLL_PRIMARY_PROVIDER", "perpay"),
		PayrollPerpayEnabled:   getEnvBool("PAYROLL_PERPAY_ENABLED", true),
		IdentityPrimaryProvider: getEnv("IDENTITY_PRIMARY_PROVIDER", "iprs"),
		IdentityIPRSEnabled:    getEnvBool("IDENTITY_IPRS_ENABLED", true),
		StoragePrimaryProvider: getEnv("STORAGE_PRIMARY_PROVIDER", "minio"),
		StorageMinIOEnabled:    getEnvBool("STORAGE_MINIO_ENABLED", true),

		// SMS — Optimize
		SMSClientID:       os.Getenv("SMS_CLIENT_ID"),
		SMSClientSecret:   os.Getenv("SMS_CLIENT_SECRET"),
		SMSTokenURL:       os.Getenv("SMS_TOKEN_URL"),
		SMSURL:            os.Getenv("SMS_URL"),
		SMSSenderID:       getEnv("SMS_SENDER_ID", "AMY-MIS"),
		SMSCallbackURL:    os.Getenv("SMS_CALLBACK_URL"),
		SMSTokenExpirySec: getEnvInt("SMS_TOKEN_EXPIRATION_TIME", 3600),

		// SMS — Africa's Talking
		ATAPIKey:    os.Getenv("AT_API_KEY"),
		ATUsername:  os.Getenv("AT_USERNAME"),
		ATShortCode: os.Getenv("AT_SHORTCODE"),
		ATBaseURL:   getEnv("AT_BASE_URL", "https://api.africastalking.com/version1"),

		// JamboPay
		JamboPayClientID:     os.Getenv("JAMBOPAY_CLIENT_ID"),
		JamboPayClientSecret: os.Getenv("JAMBOPAY_CLIENT_SECRET"),
		JamboPayBaseURL:      os.Getenv("JAMBOPAY_BASE_URL"),

		// M-Pesa Direct
		MpesaConsumerKey:    os.Getenv("MPESA_CONSUMER_KEY"),
		MpesaConsumerSecret: os.Getenv("MPESA_CONSUMER_SECRET"),
		MpesaBaseURL:        os.Getenv("MPESA_BASE_URL"),
		MpesaShortCode:      os.Getenv("MPESA_SHORTCODE"),
		MpesaPasskey:        os.Getenv("MPESA_PASSKEY"),

		// PerPay
		PerpayClientID:     os.Getenv("PERPAY_CLIENT_ID"),
		PerpayClientSecret: os.Getenv("PERPAY_CLIENT_SECRET"),
		PerpayBaseURL:      os.Getenv("PERPAY_BASE_URL"),

		// IPRS
		IPRSClientID:      os.Getenv("IPRS_CLIENT_ID"),
		IPRSClientSecret:  os.Getenv("IPRS_CLIENT_SECRET"),
		IPRSBaseURL:       os.Getenv("IPRS_BASE_URL"),
		IPRSTokenEndpoint: os.Getenv("IPRS_TOKEN_ENDPOINT"),

		// Email — Gmail SMTP
		EmailPrimaryProvider: getEnv("EMAIL_PRIMARY_PROVIDER", "gmail"),
		EmailGmailEnabled:    getEnvBool("EMAIL_GMAIL_ENABLED", true),
		EmailHost:            getEnv("EMAIL_HOST", "smtp.gmail.com"),
		EmailPort:            getEnvInt("EMAIL_PORT", 587),
		EmailUseTLS:          getEnvBool("EMAIL_USE_TLS", true),
		EmailHostUser:        os.Getenv("EMAIL_HOST_USER"),
		EmailHostPassword:    os.Getenv("EMAIL_HOST_PASSWORD"),
		EmailFromAddress:     getEnv("DEFAULT_FROM_EMAIL", os.Getenv("EMAIL_HOST_USER")),
		EmailFromName:        getEnv("EMAIL_FROM_NAME", "AMY MIS"),
		EmailSupport:         getEnv("EMAIL_SUPPORT", "sales@amy.co.ke"),

		// WhatsApp — Meta Cloud API
		WhatsAppPrimaryProvider: getEnv("WHATSAPP_PRIMARY_PROVIDER", "meta"),
		WhatsAppMetaEnabled:     getEnvBool("WHATSAPP_META_ENABLED", true),
		WhatsAppPhoneNumberID:   os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		WhatsAppAccessToken:     os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		WhatsAppAPIVersion:      getEnv("WHATSAPP_API_VERSION", "v18.0"),

		// OTP & 2FA
		OTPDefaultChannel: getEnv("OTP_DEFAULT_CHANNEL", "email"),
		OTPEnabled:        getEnvBool("OTP_ENABLED", true),
		TwoFAEnabled:      getEnvBool("TWO_FA_ENABLED", true),
		TwoFAEnforced:     getEnvBool("TWO_FA_ENFORCED", false),

		RateLimitRPM:       getEnvInt("RATE_LIMIT_RPM", 100),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),

		// Webhook Secrets
		WebhookJamboPaySecret: os.Getenv("WEBHOOK_JAMBOPAY_SECRET"),
		WebhookPerpaySecret:   os.Getenv("WEBHOOK_PERPAY_SECRET"),

		// Loan policy
		LoanConcurrencyPolicy:          getEnv("LOAN_CONCURRENCY_POLICY", "SINGLE"),
		LoanMaxConcurrent:              getEnvInt("LOAN_MAX_CONCURRENT", 3),
		LoanAggregateExposureMultiplier: getEnvFloat("LOAN_AGGREGATE_EXPOSURE_MULTIPLIER", 2.0),
		LoanCategoriesEnabled:          getEnv("LOAN_CATEGORIES_ENABLED", ""),

		// Operational tuning
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifeMin:  getEnvInt("DB_CONN_MAX_LIFE_MIN", 5),
		DBConnMaxIdleMin:  getEnvInt("DB_CONN_MAX_IDLE_MIN", 1),
		RequestTimeoutSec: getEnvInt("REQUEST_TIMEOUT_SEC", 30),
		MaxRequestBodyMB:  getEnvInt("MAX_REQUEST_BODY_MB", 10),
		CSVExportMaxRows:  getEnvInt("CSV_EXPORT_MAX_ROWS", 10000),
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

	// External APIs are optional in development, required in production
	if c.Environment == "production" {
		if c.JamboPayClientID == "" {
			return fmt.Errorf("config: JAMBOPAY_CLIENT_ID is required in production")
		}
		if c.SMSClientID == "" && c.ATAPIKey == "" {
			return fmt.Errorf("config: at least one SMS provider (SMS_CLIENT_ID or AT_API_KEY) is required in production")
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

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}
