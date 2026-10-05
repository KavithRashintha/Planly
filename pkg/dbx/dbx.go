package dbx

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds Postgres connection options.
type Config struct {
	Host     string `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port     int    `envconfig:"POSTGRES_PORT" default:"5432"`
	User     string `envconfig:"POSTGRES_USER" default:"postgres"`
	Password string `envconfig:"POSTGRES_PASSWORD" default:"postgres"`
	Database string `envconfig:"POSTGRES_DB" required:"true"`
	SSLMode  string `envconfig:"POSTGRES_SSLMODE" default:"disable"`
	MaxConns int32  `envconfig:"POSTGRES_MAX_CONNS" default:"25"`
	MinConns int32  `envconfig:"POSTGRES_MIN_CONNS" default:"5"`
}

// Connect initializes a pgx connection pool with retries.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	maxConns := cfg.MaxConns
	if maxConns <= 0 {
		maxConns = 25
	}
	minConns := cfg.MinConns
	if minConns < 0 {
		minConns = 5
	}

	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	var pool *pgxpool.Pool
	const maxAttempts = 10
	retryDelay := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			pingErr := pool.Ping(pingCtx)
			pingCancel()

			if pingErr == nil {
				slog.Info("connected to database successfully",
					"host", cfg.Host,
					"database", cfg.Database,
					"attempt", attempt,
				)
				return pool, nil
			}
			pool.Close()
			err = pingErr
		}

		slog.Warn("database connection attempt failed, retrying...",
			"database", cfg.Database,
			"attempt", attempt,
			"error", err,
			"retry_in", retryDelay.String(),
		)

		time.Sleep(retryDelay)
		if retryDelay < 5*time.Second {
			retryDelay *= 2
		}
	}

	return nil, fmt.Errorf("failed to connect to database %s after %d attempts: %w", cfg.Database, maxAttempts, err)
}
