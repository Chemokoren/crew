package config

import (
	"os"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	os.Setenv("REDIS_URL", "redis://localhost:6379/0")
	os.Setenv("JWT_SECRET", "test-secret-that-is-definitely-at-least-32-characters-long")
	os.Setenv("MINIO_ENDPOINT", "localhost:9000")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"PORT", "ENVIRONMENT", "DATABASE_URL", "REDIS_URL", "JWT_SECRET",
		"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET"} {
		os.Unsetenv(key)
	}
}

func TestLoadValid(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	defer clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want development", cfg.Environment)
	}
	if cfg.MinIOBucket != "amy-mis" {
		t.Errorf("MinIOBucket = %q, want amy-mis", cfg.MinIOBucket)
	}
	if cfg.JWTExpiryMinutes != 15 {
		t.Errorf("JWTExpiryMinutes = %d, want 15", cfg.JWTExpiryMinutes)
	}
	if cfg.RateLimitRPM != 100 {
		t.Errorf("RateLimitRPM = %d, want 100", cfg.RateLimitRPM)
	}
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Unsetenv("DATABASE_URL")
	defer clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when DATABASE_URL is missing")
	}
}

func TestLoadMissingRedisURL(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Unsetenv("REDIS_URL")
	defer clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when REDIS_URL is missing")
	}
}

func TestLoadMissingJWTSecret(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Unsetenv("JWT_SECRET")
	defer clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when JWT_SECRET is missing")
	}
}

func TestLoadShortJWTSecret(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Setenv("JWT_SECRET", "too-short")
	defer clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when JWT_SECRET is less than 32 chars")
	}
}

func TestLoadMissingMinIO(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Unsetenv("MINIO_ENDPOINT")
	defer clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when MINIO_ENDPOINT is missing")
	}
}

func TestLoadCustomPort(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Setenv("PORT", "3000")
	defer clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Port)
	}
}

func TestIsDevelopment(t *testing.T) {
	cfg := &Config{Environment: "development"}
	if !cfg.IsDevelopment() {
		t.Error("IsDevelopment() should return true for development")
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() should return false for development")
	}
}

func TestIsProduction(t *testing.T) {
	cfg := &Config{Environment: "production"}
	if !cfg.IsProduction() {
		t.Error("IsProduction() should return true for production")
	}
	if cfg.IsDevelopment() {
		t.Error("IsDevelopment() should return false for production")
	}
}

func TestProductionRequiresExternalAPIs(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Setenv("ENVIRONMENT", "production")
	defer clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail in production when JAMBOPAY_CLIENT_ID is missing")
	}
}

// --- Integration Enable/Disable Config Tests ---

func TestIntegrationEnableDisableDefaults(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	defer clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// SMS defaults
	if cfg.SMSPrimaryProvider != "optimize" {
		t.Errorf("SMSPrimaryProvider = %q, want optimize", cfg.SMSPrimaryProvider)
	}
	if !cfg.SMSOptimizeEnabled {
		t.Error("SMSOptimizeEnabled should default to true")
	}
	if !cfg.SMSATEnabled {
		t.Error("SMSATEnabled should default to true")
	}

	// Payment defaults
	if cfg.PaymentPrimaryProvider != "jambopay" {
		t.Errorf("PaymentPrimaryProvider = %q, want jambopay", cfg.PaymentPrimaryProvider)
	}
	if !cfg.PaymentJamboPayEnabled {
		t.Error("PaymentJamboPayEnabled should default to true")
	}
	if cfg.PaymentMpesaEnabled {
		t.Error("PaymentMpesaEnabled should default to false")
	}

	// Payroll defaults
	if cfg.PayrollPrimaryProvider != "perpay" {
		t.Errorf("PayrollPrimaryProvider = %q, want perpay", cfg.PayrollPrimaryProvider)
	}
	if !cfg.PayrollPerpayEnabled {
		t.Error("PayrollPerpayEnabled should default to true")
	}

	// Identity defaults
	if cfg.IdentityPrimaryProvider != "iprs" {
		t.Errorf("IdentityPrimaryProvider = %q, want iprs", cfg.IdentityPrimaryProvider)
	}
	if !cfg.IdentityIPRSEnabled {
		t.Error("IdentityIPRSEnabled should default to true")
	}

	// Storage defaults
	if cfg.StoragePrimaryProvider != "minio" {
		t.Errorf("StoragePrimaryProvider = %q, want minio", cfg.StoragePrimaryProvider)
	}
	if !cfg.StorageMinIOEnabled {
		t.Error("StorageMinIOEnabled should default to true")
	}
}

func TestIntegrationDisableViaEnv(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Setenv("SMS_OPTIMIZE_ENABLED", "false")
	os.Setenv("PAYMENT_JAMBOPAY_ENABLED", "false")
	os.Setenv("PAYMENT_MPESA_ENABLED", "true")
	os.Setenv("PAYROLL_PERPAY_ENABLED", "false")
	os.Setenv("IDENTITY_IPRS_ENABLED", "false")
	os.Setenv("STORAGE_MINIO_ENABLED", "false")
	defer clearEnv(t)
	defer func() {
		os.Unsetenv("SMS_OPTIMIZE_ENABLED")
		os.Unsetenv("PAYMENT_JAMBOPAY_ENABLED")
		os.Unsetenv("PAYMENT_MPESA_ENABLED")
		os.Unsetenv("PAYROLL_PERPAY_ENABLED")
		os.Unsetenv("IDENTITY_IPRS_ENABLED")
		os.Unsetenv("STORAGE_MINIO_ENABLED")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.SMSOptimizeEnabled {
		t.Error("SMSOptimizeEnabled should be false when set via env")
	}
	if cfg.PaymentJamboPayEnabled {
		t.Error("PaymentJamboPayEnabled should be false when set via env")
	}
	if !cfg.PaymentMpesaEnabled {
		t.Error("PaymentMpesaEnabled should be true when set via env")
	}
	if cfg.PayrollPerpayEnabled {
		t.Error("PayrollPerpayEnabled should be false when set via env")
	}
	if cfg.IdentityIPRSEnabled {
		t.Error("IdentityIPRSEnabled should be false when set via env")
	}
	if cfg.StorageMinIOEnabled {
		t.Error("StorageMinIOEnabled should be false when set via env")
	}
}

func TestPrimaryProviderOverride(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Setenv("SMS_PRIMARY_PROVIDER", "africastalking")
	os.Setenv("PAYMENT_PRIMARY_PROVIDER", "mpesa")
	os.Setenv("PAYROLL_PRIMARY_PROVIDER", "alternative")
	os.Setenv("IDENTITY_PRIMARY_PROVIDER", "alternative-kyc")
	os.Setenv("STORAGE_PRIMARY_PROVIDER", "s3")
	defer clearEnv(t)
	defer func() {
		os.Unsetenv("SMS_PRIMARY_PROVIDER")
		os.Unsetenv("PAYMENT_PRIMARY_PROVIDER")
		os.Unsetenv("PAYROLL_PRIMARY_PROVIDER")
		os.Unsetenv("IDENTITY_PRIMARY_PROVIDER")
		os.Unsetenv("STORAGE_PRIMARY_PROVIDER")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.SMSPrimaryProvider != "africastalking" {
		t.Errorf("SMSPrimaryProvider = %q, want africastalking", cfg.SMSPrimaryProvider)
	}
	if cfg.PaymentPrimaryProvider != "mpesa" {
		t.Errorf("PaymentPrimaryProvider = %q, want mpesa", cfg.PaymentPrimaryProvider)
	}
	if cfg.PayrollPrimaryProvider != "alternative" {
		t.Errorf("PayrollPrimaryProvider = %q, want alternative", cfg.PayrollPrimaryProvider)
	}
	if cfg.IdentityPrimaryProvider != "alternative-kyc" {
		t.Errorf("IdentityPrimaryProvider = %q, want alternative-kyc", cfg.IdentityPrimaryProvider)
	}
	if cfg.StoragePrimaryProvider != "s3" {
		t.Errorf("StoragePrimaryProvider = %q, want s3", cfg.StoragePrimaryProvider)
	}
}

func TestMpesaCredentialsLoaded(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	os.Setenv("MPESA_CONSUMER_KEY", "test-key")
	os.Setenv("MPESA_CONSUMER_SECRET", "test-secret")
	os.Setenv("MPESA_BASE_URL", "https://sandbox.safaricom.co.ke")
	os.Setenv("MPESA_SHORTCODE", "174379")
	os.Setenv("MPESA_PASSKEY", "test-passkey")
	defer clearEnv(t)
	defer func() {
		os.Unsetenv("MPESA_CONSUMER_KEY")
		os.Unsetenv("MPESA_CONSUMER_SECRET")
		os.Unsetenv("MPESA_BASE_URL")
		os.Unsetenv("MPESA_SHORTCODE")
		os.Unsetenv("MPESA_PASSKEY")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.MpesaConsumerKey != "test-key" {
		t.Errorf("MpesaConsumerKey = %q, want test-key", cfg.MpesaConsumerKey)
	}
	if cfg.MpesaConsumerSecret != "test-secret" {
		t.Errorf("MpesaConsumerSecret = %q, want test-secret", cfg.MpesaConsumerSecret)
	}
	if cfg.MpesaBaseURL != "https://sandbox.safaricom.co.ke" {
		t.Errorf("MpesaBaseURL = %q, want sandbox URL", cfg.MpesaBaseURL)
	}
	if cfg.MpesaShortCode != "174379" {
		t.Errorf("MpesaShortCode = %q, want 174379", cfg.MpesaShortCode)
	}
	if cfg.MpesaPasskey != "test-passkey" {
		t.Errorf("MpesaPasskey = %q, want test-passkey", cfg.MpesaPasskey)
	}
}

func TestOperationalTuningDefaults(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	defer clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"DBMaxOpenConns", cfg.DBMaxOpenConns, 25},
		{"DBMaxIdleConns", cfg.DBMaxIdleConns, 10},
		{"DBConnMaxLifeMin", cfg.DBConnMaxLifeMin, 5},
		{"DBConnMaxIdleMin", cfg.DBConnMaxIdleMin, 1},
		{"RequestTimeoutSec", cfg.RequestTimeoutSec, 30},
		{"MaxRequestBodyMB", cfg.MaxRequestBodyMB, 10},
		{"CSVExportMaxRows", cfg.CSVExportMaxRows, 10000},
		{"RateLimitRPM", cfg.RateLimitRPM, 100},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

func TestOperationalTuningOverrides(t *testing.T) {
	clearEnv(t)
	setRequiredEnv(t)
	defer clearEnv(t)

	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "20")
	os.Setenv("DB_CONN_MAX_LIFE_MIN", "10")
	os.Setenv("DB_CONN_MAX_IDLE_MIN", "3")
	os.Setenv("REQUEST_TIMEOUT_SEC", "60")
	os.Setenv("MAX_REQUEST_BODY_MB", "25")
	os.Setenv("CSV_EXPORT_MAX_ROWS", "5000")
	os.Setenv("RATE_LIMIT_RPM", "200")
	defer func() {
		for _, k := range []string{"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS",
			"DB_CONN_MAX_LIFE_MIN", "DB_CONN_MAX_IDLE_MIN",
			"REQUEST_TIMEOUT_SEC", "MAX_REQUEST_BODY_MB",
			"CSV_EXPORT_MAX_ROWS", "RATE_LIMIT_RPM"} {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"DBMaxOpenConns", cfg.DBMaxOpenConns, 50},
		{"DBMaxIdleConns", cfg.DBMaxIdleConns, 20},
		{"DBConnMaxLifeMin", cfg.DBConnMaxLifeMin, 10},
		{"DBConnMaxIdleMin", cfg.DBConnMaxIdleMin, 3},
		{"RequestTimeoutSec", cfg.RequestTimeoutSec, 60},
		{"MaxRequestBodyMB", cfg.MaxRequestBodyMB, 25},
		{"CSVExportMaxRows", cfg.CSVExportMaxRows, 5000},
		{"RateLimitRPM", cfg.RateLimitRPM, 200},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

