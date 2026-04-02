package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/auth-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// GetUserByEmail retrieves a user by email
func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err == pgx.ErrNoRows {
		return db.User{}, fmt.Errorf("user not found")
	}
	if err != nil {
		return db.User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *AuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, err := r.queries.GetUserByID(ctx, utils.UUIDToPgtype(id))
	if err == pgx.ErrNoRows {
		return db.User{}, fmt.Errorf("user not found")
	}
	if err != nil {
		return db.User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// CreateUser creates a new user
func (r *AuthRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (db.CreateUserRow, error) {
	user, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return db.CreateUserRow{}, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

// UpdatePassword updates user password and increments token version
func (r *AuthRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, forcePasswordChange bool) error {
	forceChange := forcePasswordChange
	err := r.queries.UpdatePassword(ctx, db.UpdatePasswordParams{
		ID:                  utils.UUIDToPgtype(userID),
		PasswordHash:        passwordHash,
		ForcePasswordChange: &forceChange,
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// UpdateUser updates user information (email, department)
func (r *AuthRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) error {
	err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// IncrementTokenVersion increments token version for logout-all
func (r *AuthRepository) IncrementTokenVersion(ctx context.Context, userID uuid.UUID) (int32, error) {
	version, err := r.queries.IncrementTokenVersion(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return 0, fmt.Errorf("failed to increment token version: %w", err)
	}
	if version == nil {
		return 0, fmt.Errorf("no version returned")
	}
	return *version, nil
}

// IncrementFailedLoginAttempts increments failed login attempts
func (r *AuthRepository) IncrementFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.IncrementFailedLoginAttempts(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("failed to increment failed login attempts: %w", err)
	}
	return nil
}

// ResetFailedLoginAttempts resets failed login attempts to 0
func (r *AuthRepository) ResetFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.ResetFailedLoginAttempts(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("failed to reset failed login attempts: %w", err)
	}
	return nil
}

// LockAccount locks account until specified time
func (r *AuthRepository) LockAccount(ctx context.Context, params db.LockAccountParams) error {
	err := r.queries.LockAccount(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}
	return nil
}

// DeactivateUser soft deletes a user
func (r *AuthRepository) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.DeactivateUser(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}
	return nil
}

// AdminExists checks if an admin user exists
func (r *AuthRepository) AdminExists(ctx context.Context) (bool, error) {
	exists, err := r.queries.AdminExists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check admin existence: %w", err)
	}
	return exists, nil
}

// CheckEmailVersionSync checks if email changed and increments token version
func (r *AuthRepository) CheckEmailVersionSync(ctx context.Context, userID uuid.UUID, newEmail string) (int32, error) {
	version, err := r.queries.CheckEmailVersionSync(ctx, db.CheckEmailVersionSyncParams{
		ID:    utils.UUIDToPgtype(userID),
		Email: newEmail,
	})
	if err == pgx.ErrNoRows {
		// Email didn't change, return 0
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to check email version sync: %w", err)
	}
	if version == nil {
		return 0, nil
	}
	return *version, nil
}
