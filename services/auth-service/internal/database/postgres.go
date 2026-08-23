package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(context context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 15
	config.MinConns = 5
	config.MaxConnIdleTime = 15 * time.Minute
	config.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(context, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create PostgreSQL connection pool: %w", err)
	}

	if err := pool.Ping(context); err != nil {
		return nil, fmt.Errorf("Failed to ping PostgreSQL database: %w", err)
	}

	return pool, nil
}
