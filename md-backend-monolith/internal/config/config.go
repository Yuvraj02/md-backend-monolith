package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds monolith HTTP + Postgres configuration.
type Config struct {
	AppEnv   string
	Service  string
	HTTPPort int

	DatabaseHost     string
	DatabaseHostaddr string // optional; set when local DNS cannot resolve Neon
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	CORSAllowedOrigins string
	OwnerStudioSecret  string
	ShutdownTimeout    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		Service:            getEnv("SERVICE_NAME", "md-backend-monolith"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		OwnerStudioSecret:  getEnv("OWNER_STUDIO_SECRET", "marketing-digest-dev"),
		DatabaseSSLMode:    getEnv("DATABASE_SSLMODE", "require"),
	}

	var err error
	// Prefer HTTP_PORT; fall back to PORT (Render / Fly / Railway inject PORT).
	httpPortDefault := 8080
	if p := os.Getenv("PORT"); p != "" {
		if n, perr := strconv.Atoi(p); perr == nil && n > 0 {
			httpPortDefault = n
		}
	}
	cfg.HTTPPort, err = getEnvInt("HTTP_PORT", httpPortDefault)
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseHost = os.Getenv("DATABASE_HOST")
	cfg.DatabaseHostaddr = os.Getenv("DATABASE_HOSTADDR")
	cfg.DatabaseName = os.Getenv("DATABASE_NAME")
	cfg.DatabaseUser = os.Getenv("DATABASE_USER")
	cfg.DatabasePassword = os.Getenv("DATABASE_PASSWORD")

	cfg.DatabasePort, err = getEnvInt("DATABASE_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}
	cfg.DBConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout, err = getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	required := map[string]string{
		"DATABASE_HOST":     c.DatabaseHost,
		"DATABASE_NAME":     c.DatabaseName,
		"DATABASE_USER":     c.DatabaseUser,
		"DATABASE_PASSWORD": c.DatabasePassword,
	}
	for k, v := range required {
		if v == "" {
			return fmt.Errorf("missing required env: %s", k)
		}
	}
	if c.HTTPPort <= 0 {
		return fmt.Errorf("HTTP_PORT must be positive")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}