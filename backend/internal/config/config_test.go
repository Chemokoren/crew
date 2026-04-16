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
		t.Error("Load() should fail in production when JAMBOPAY_API_KEY is missing")
	}
}
