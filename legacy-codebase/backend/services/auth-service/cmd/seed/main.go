package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/baaaki/mydreamcampus/auth-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Ping database
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	fmt.Println("✅ Connected to database")

	queries := db.New(pool)

	// Delete existing admin if exists
	_, err = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", cfg.Admin.Email)
	if err != nil {
		log.Printf("Warning: failed to delete existing admin: %v", err)
	} else {
		fmt.Println("🗑️  Deleted existing admin user (if any)")
	}

	// Hash password
	passwordHash, err := utils.HashPassword(cfg.Admin.InitialPassword)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Fixed admin UUID
	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create admin user
	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		ID:                  utils.UUIDToPgtype(adminID),
		Email:               cfg.Admin.Email,
		PasswordHash:        passwordHash,
		Role:                "admin",
		Department:          nil,
		IsActive:            utils.BoolPtr(true),
		TokenVersion:        utils.Int32Ptr(1),
		ForcePasswordChange: utils.BoolPtr(false), // Don't force password change for seeded admin
	})
	if err != nil {
		log.Fatalf("failed to create admin user: %v", err)
	}

	fmt.Println("✅ Admin user created successfully!")
	fmt.Println("")
	fmt.Println("📧 Email:", cfg.Admin.Email)
	fmt.Println("🔑 Password:", cfg.Admin.InitialPassword)
	fmt.Println("")
	fmt.Println("You can now login with these credentials.")

	os.Exit(0)
}
