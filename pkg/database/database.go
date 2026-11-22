// Package database initializes GORM against PostgreSQL.
// Schema changes are managed by Atlas — never call AutoMigrate in production.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config holds PostgreSQL connection settings loaded from environment.
type Config struct {
	Host            string
	Hostaddr        string // optional IP; skips DNS when system resolver fails (e.g. Neon)
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN builds a PostgreSQL connection string. Password is never logged by callers.
func (c Config) DSN() string {
	ssl := c.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, ssl,
	)
}

// Open connects with GORM and configures the underlying connection pool.
func Open(cfg Config) (*gorm.DB, error) {
	pgCfg, err := pgx.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	// pgx does not honor libpq hostaddr; override DNS when an IP is provided.
	if cfg.Hostaddr != "" {
		ip := cfg.Hostaddr
		pgCfg.LookupFunc = func(_ context.Context, _ string) ([]string, error) {
			return []string{ip}, nil
		}
	}

	sqlDB := stdlib.OpenDB(*pgCfg)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		// Explicit: do not AutoMigrate. Atlas owns schema.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = 30 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(lifetime)

	return db, nil
}

// Ping verifies connectivity using the request/lifecycle context.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the underlying SQL connection pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
