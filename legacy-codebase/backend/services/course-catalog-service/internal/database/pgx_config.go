package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPoolWithEnums creates a PostgreSQL connection pool with custom enum types registered
func NewPoolWithEnums(connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = time.Minute

	// Register custom enum types for each connection
	config.AfterConnect = RegisterEnumTypes

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// RegisterEnumTypes registers custom PostgreSQL enum types for pgx.
// This is required for array operations like: WHERE day = ANY($1::day_of_week_enum[])
func RegisterEnumTypes(ctx context.Context, conn *pgx.Conn) error {
	return registerEnum(ctx, conn, "day_of_week_enum")
}

func registerEnum(ctx context.Context, conn *pgx.Conn, enumName string) error {
	// Get enum OID
	var enumOID uint32
	err := conn.QueryRow(ctx, "SELECT oid FROM pg_type WHERE typname = $1", enumName).Scan(&enumOID)
	if err != nil {
		return fmt.Errorf("failed to get enum OID: %w", err)
	}

	// Get array OID (don't assume it's oid+1, query it explicitly)
	var arrayOID uint32
	err = conn.QueryRow(ctx, "SELECT oid FROM pg_type WHERE typname = $1", "_"+enumName).Scan(&arrayOID)
	if err != nil {
		return fmt.Errorf("failed to get array OID: %w", err)
	}

	// Register enum type
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name:  enumName,
		OID:   enumOID,
		Codec: &pgtype.EnumCodec{},
	})

	// Register array type for ANY(...) operations
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name: "_" + enumName,
		OID:  arrayOID,
		Codec: &pgtype.ArrayCodec{
			ElementType: &pgtype.Type{
				OID:   enumOID,
				Codec: &pgtype.EnumCodec{},
			},
		},
	})

	return nil
}