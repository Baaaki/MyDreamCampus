// Package pgtest gives an integration test a real, throwaway PostgreSQL
// database shaped like a service's production one. It is used only from
// files behind the "integration" build tag; `make test-integration` starts
// the server it needs.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvURL names the variable holding a superuser URL of the test server.
const EnvURL = "TEST_DATABASE_URL"

// New creates a database of its own for the test, prepares it the way
// infrastructure/postgres/init-databases.sh prepares the real one (the two
// extensions and the service's schema), applies every goose migration in
// migrationsDir and drops the database when the test ends.
//
// A missing EnvURL fails the test rather than skipping it: a tagged run
// that silently tested nothing would read as green.
func New(t testing.TB, schema, migrationsDir string) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv(EnvURL)
	if adminURL == "" {
		t.Fatalf("%s is not set; run `make test-integration` from the repository root", EnvURL)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to the test server: %v", err)
	}
	defer func() { _ = admin.Close(context.Background()) }()

	name := "it_" + randomSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database %s: %v", name, err)
	}

	cfg, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parse %s: %v", EnvURL, err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open database %s: %v", name, err)
	}

	t.Cleanup(func() {
		pool.Close()
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		conn, err := pgx.Connect(cctx, adminURL)
		if err != nil {
			t.Errorf("drop database %s: %v", name, err)
			return
		}
		defer func() { _ = conn.Close(context.Background()) }()
		if _, err := conn.Exec(cctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database %s: %v", name, err)
		}
	})

	setup := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS %q;`, schema)
	if _, err := pool.Exec(ctx, setup); err != nil {
		t.Fatalf("prepare database %s: %v", name, err)
	}
	if err := migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("migrate database %s: %v", name, err)
	}
	return pool
}

// migrate applies the Up section of each goose migration in file order.
// None of the migrations needs goose's per-statement handling (no
// NO TRANSACTION, no CONCURRENTLY), so each Up section runs as one
// simple-protocol query, dollar-quoted function bodies included.
func migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migrations in %s", dir)
	}
	sort.Strings(files)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		up, err := upSection(string(raw))
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if strings.TrimSpace(up) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, up); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

func upSection(sql string) (string, error) {
	var b strings.Builder
	inUp := false
	for _, line := range strings.Split(sql, "\n") {
		directive := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(directive, "-- +goose Up"):
			inUp = true
			continue
		case strings.HasPrefix(directive, "-- +goose Down"):
			return b.String(), nil
		case strings.HasPrefix(directive, "-- +goose NO TRANSACTION"):
			return "", fmt.Errorf("NO TRANSACTION migrations need goose itself")
		case strings.HasPrefix(directive, "-- +goose"):
			continue
		}
		if inUp {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if !inUp {
		return "", fmt.Errorf("no -- +goose Up section")
	}
	return b.String(), nil
}

func randomSuffix(t testing.TB) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("random database name: %v", err)
	}
	return hex.EncodeToString(b[:])
}
