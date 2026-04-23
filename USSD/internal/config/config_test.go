package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Set required env vars
	os.Setenv("REDIS_URL", "redis://localhost:6379/1")
	os.Setenv("BACKEND_BASE_URL", "http://localhost:8080")
	defer func() {
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("BACKEND_BASE_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify defaults
	if cfg.Port != 8090 {
		t.Errorf("Port = %d, want 8090", cfg.Port)
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want 'development'", cfg.Environment)
	}
	if cfg.SessionTTLSeconds != 180 {
		t.Errorf("SessionTTLSeconds = %d, want 180", cfg.SessionTTLSeconds)
	}
	if cfg.RedisPoolSize != 100 {
		t.Errorf("RedisPoolSize = %d, want 100", cfg.RedisPoolSize)
	}
	if cfg.RateLimitPerMSISDN != 30 {
		t.Errorf("RateLimitPerMSISDN = %d, want 30", cfg.RateLimitPerMSISDN)
	}
	if cfg.CBMaxFailures != 5 {
		t.Errorf("CBMaxFailures = %d, want 5", cfg.CBMaxFailures)
	}
	if cfg.DefaultLanguage != "en" {
		t.Errorf("DefaultLanguage = %q, want 'en'", cfg.DefaultLanguage)
	}
	if cfg.InputMaxLength != 160 {
		t.Errorf("InputMaxLength = %d, want 160", cfg.InputMaxLength)
	}
	if cfg.BackendTimeoutMs != 1500 {
		t.Errorf("BackendTimeoutMs = %d, want 1500", cfg.BackendTimeoutMs)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Clear env
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("BACKEND_BASE_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when REDIS_URL is missing")
	}
}

func TestLoad_ProductionValidation(t *testing.T) {
	os.Setenv("REDIS_URL", "redis://localhost:6379/1")
	os.Setenv("BACKEND_BASE_URL", "http://localhost:8080")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("BACKEND_API_KEY", "")
	defer func() {
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("BACKEND_BASE_URL")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("BACKEND_API_KEY")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail in production without BACKEND_API_KEY")
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := &Config{Environment: "development"}
	if !cfg.IsDevelopment() {
		t.Error("IsDevelopment() should return true")
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() should return false")
	}
}

func TestConfig_SessionTTL(t *testing.T) {
	cfg := &Config{SessionTTLSeconds: 120}
	ttl := cfg.SessionTTL()
	if ttl.Seconds() != 120 {
		t.Errorf("SessionTTL() = %v, want 2m0s", ttl)
	}
}

func TestConfig_BackendTimeout(t *testing.T) {
	cfg := &Config{BackendTimeoutMs: 1500}
	timeout := cfg.BackendTimeout()
	if timeout.Milliseconds() != 1500 {
		t.Errorf("BackendTimeout() = %v, want 1.5s", timeout)
	}
}
