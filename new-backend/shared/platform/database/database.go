package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultMaxConns applies when the caller passes no limit. At 10 each, all
// services with a database fit under Postgres' default 100 connections.
const defaultMaxConns = 10

func NewPostgresPool(connString string, maxConns int) (*pgxpool.Pool, error) {
	config, err := poolConfig(connString, maxConns)
	if err != nil {
		return nil, err
	}

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

func poolConfig(connString string, maxConns int) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	if maxConns <= 0 {
		maxConns = defaultMaxConns
	}
	config.MaxConns = int32(maxConns)
	config.MinConns = min(2, config.MaxConns)

	// MaxConnLifetime: Maximum lifetime of a connection
	// Helps prevent connection leaks and stale connections
	config.MaxConnLifetime = time.Hour

	// MaxConnIdleTime: Maximum time a connection can be idle
	// Idle connections are closed to free up resources
	config.MaxConnIdleTime = 5 * time.Minute

	// HealthCheckPeriod: How often to check connection health
	config.HealthCheckPeriod = time.Minute

	return config, nil
}
