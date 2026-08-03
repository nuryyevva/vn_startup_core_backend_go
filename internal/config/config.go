// Package config provides the single source of truth for all runtime
// configuration. No other package in this project should call os.Getenv —
// every configurable value must be added here and threaded through via
// explicit dependency injection.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is the root configuration object for the service. It is built once
// in cmd/api/main.go via Load() and passed down explicitly to every
// component that needs it.
type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	NATS     NATSConfig
	JWT      JWTConfig
	Dialog   DialogConfig
	Dev      DevConfig
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	Port int    `env:"SERVER_PORT" envDefault:"8080"`
	Env  string `env:"APP_ENV" envDefault:"development"` // development|staging|production
}

// PostgresConfig controls the connection to the primary Postgres database.
type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	DBName   string `env:"POSTGRES_DB" envDefault:"vn_platform"`
	SSLMode  string `env:"POSTGRES_SSLMODE" envDefault:"disable"`
	MaxConns int    `env:"POSTGRES_MAX_CONNS" envDefault:"20"`
}

// DSN builds a libpq-style connection string from the config.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode,
	)
}

// RedisConfig controls the connection to Redis.
type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

// NATSConfig controls the connection to NATS / JetStream.
type NATSConfig struct {
	URL string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
}

// JWTConfig controls token signing and lifetimes.
type JWTConfig struct {
	Secret          string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
}

// DialogConfig controls free-dialog session defaults.
type DialogConfig struct {
	SessionCheckInterval time.Duration `env:"DIALOG_SESSION_CHECK_INTERVAL" envDefault:"10s"`
	DefaultMessageCost   int           `env:"DIALOG_DEFAULT_MESSAGE_COST" envDefault:"20"`
}

// DevConfig controls development-only conveniences that must never be
// enabled in production.
type DevConfig struct {
	EnableDevEndpoints bool `env:"ENABLE_DEV_ENDPOINTS" envDefault:"false"` // включает /dev/wallet/grant
}

// Load parses environment variables into a Config, applying defaults and
// validating required fields. It should be called exactly once, at process
// startup in cmd/api/main.go.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parse environment: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	switch c.Server.Env {
	case "development", "staging", "production":
	default:
		return fmt.Errorf("APP_ENV must be one of development|staging|production, got %q", c.Server.Env)
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT must be a valid TCP port, got %d", c.Server.Port)
	}

	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters long")
	}

	if c.Postgres.MaxConns <= 0 {
		return fmt.Errorf("POSTGRES_MAX_CONNS must be positive, got %d", c.Postgres.MaxConns)
	}

	if c.Dialog.DefaultMessageCost < 0 {
		return fmt.Errorf("DIALOG_DEFAULT_MESSAGE_COST must not be negative, got %d", c.Dialog.DefaultMessageCost)
	}

	return nil
}
