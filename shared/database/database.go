package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool options
	// MaxConns: Maximum number of connections in the pool
	// Recommended: 4 * number of CPU cores for OLTP workloads
	config.MaxConns = 25

	// MinConns: Minimum number of connections to keep alive
	config.MinConns = 5

	// MaxConnLifetime: Maximum lifetime of a connection
	// Helps prevent connection leaks and stale connections
	config.MaxConnLifetime = time.Hour

	// MaxConnIdleTime: Maximum time a connection can be idle
	// Idle connections are closed to free up resources
	config.MaxConnIdleTime = 5 * time.Minute

	// HealthCheckPeriod: How often to check connection health
	config.HealthCheckPeriod = time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	//Connection test
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}