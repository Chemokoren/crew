package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// PoolConfig holds configurable database connection pool settings.
type PoolConfig struct {
	MaxOpenConns    int // Default: 25
	MaxIdleConns    int // Default: 10
	ConnMaxLifeMin  int // Default: 5
	ConnMaxIdleMin  int // Default: 1
}

// Connect establishes a connection to PostgreSQL and configures the pool.
func Connect(databaseURL string, isDev bool, pool PoolConfig) (*gorm.DB, error) {
	// Configure GORM logger
	logLevel := logger.Silent
	if isDev {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: false,
		PrepareStmt:                              true,
	}

	db, err := gorm.Open(postgres.Open(databaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// Apply pool defaults if zero-valued
	if pool.MaxOpenConns <= 0 {
		pool.MaxOpenConns = 25
	}
	if pool.MaxIdleConns <= 0 {
		pool.MaxIdleConns = 10
	}
	if pool.ConnMaxLifeMin <= 0 {
		pool.ConnMaxLifeMin = 5
	}
	if pool.ConnMaxIdleMin <= 0 {
		pool.ConnMaxIdleMin = 1
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(pool.ConnMaxLifeMin) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(pool.ConnMaxIdleMin) * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("connected to PostgreSQL",
		slog.Int("max_open_conns", pool.MaxOpenConns),
		slog.Int("max_idle_conns", pool.MaxIdleConns),
		slog.Int("conn_max_life_min", pool.ConnMaxLifeMin),
		slog.Int("conn_max_idle_min", pool.ConnMaxIdleMin),
	)

	return db, nil
}

// RunMigrations applies all pending migrations from the given path.
func RunMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	slog.Info("database migrations applied successfully")
	return nil
}
