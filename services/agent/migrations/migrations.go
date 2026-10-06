package migrations

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 000001_init_agent.up.sql
var UpSQL string

// Run executes the initial schema migration script.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, UpSQL); err != nil {
		return fmt.Errorf("failed to run agent migrations: %w", err)
	}
	return nil
}
