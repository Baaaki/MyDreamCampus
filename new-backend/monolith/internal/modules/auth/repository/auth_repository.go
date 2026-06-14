package repository

import (
	"context"
	"errors"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, fmt.Errorf("%w: user with email %s not found", serviceErrors.ErrUserNotFoundRepo, email)
		}
		return db.User{}, fmt.Errorf("%w: failed to get user: %v", sharedErrors.ErrQueryFailed, err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *AuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, err := r.queries.GetUserByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, fmt.Errorf("%w: user with id %s not found", serviceErrors.ErrUserNotFoundRepo, id)
		}
		return db.User{}, fmt.Errorf("%w: failed to get user: %v", sharedErrors.ErrQueryFailed, err)
	}
	return user, nil
}

// CreateUser creates a new user
func (r *AuthRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (db.CreateUserRow, error) {
	user, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		// Check for duplicate email constraint violation
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) {
			if pgxErr.Code == "23505" { // unique_violation
				return db.CreateUserRow{}, fmt.Errorf("%w: email already exists", serviceErrors.ErrUserExistsRepo)
			}
		}
		return db.CreateUserRow{}, fmt.Errorf("%w: failed to create user: %v", sharedErrors.ErrQueryFailed, err)
	}
	return user, nil
}

// CreateOutboxEvent creates a new outbox event
func (r *AuthRepository) CreateOutboxEvent(ctx context.Context, params db.CreateOutboxEventParams) (db.CreateOutboxEventRow, error) {
	event, err := r.queries.CreateOutboxEvent(ctx, params)
	if err != nil {
		return db.CreateOutboxEventRow{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return event, nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: user not found for password update", serviceErrors.ErrUserNotFoundRepo)
		}
		return fmt.Errorf("%w: failed to update password: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// UpdateUser updates user information (email, department)
func (r *AuthRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) error {
	err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: user not found for update", serviceErrors.ErrUserNotFoundRepo)
		}
		return fmt.Errorf("%w: failed to update user: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// IncrementTokenVersion increments token version for logout-all
func (r *AuthRepository) IncrementTokenVersion(ctx context.Context, userID uuid.UUID) (int32, error) {
	version, err := r.queries.IncrementTokenVersion(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%w: user not found for token version increment", serviceErrors.ErrUserNotFoundRepo)
		}
		return 0, fmt.Errorf("%w: failed to increment token version: %v", sharedErrors.ErrQueryFailed, err)
	}
	if version == nil {
		return 0, fmt.Errorf("%w: no version returned", sharedErrors.ErrQueryFailed)
	}
	return *version, nil
}

// IncrementFailedLoginAttempts increments failed login attempts
func (r *AuthRepository) IncrementFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.IncrementFailedLoginAttempts(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("%w: failed to increment failed login attempts: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// ResetFailedLoginAttempts resets failed login attempts to 0
func (r *AuthRepository) ResetFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.ResetFailedLoginAttempts(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("%w: failed to reset failed login attempts: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// LockAccount locks account until specified time
func (r *AuthRepository) LockAccount(ctx context.Context, params db.LockAccountParams) error {
	err := r.queries.LockAccount(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: failed to lock account: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// DeactivateUser soft deletes a user
func (r *AuthRepository) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.DeactivateUser(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: user not found for deactivation", serviceErrors.ErrUserNotFoundRepo)
		}
		return fmt.Errorf("%w: failed to deactivate user: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// AdminExists checks if an admin user exists
func (r *AuthRepository) AdminExists(ctx context.Context) (bool, error) {
	exists, err := r.queries.AdminExists(ctx)
	if err != nil {
		return false, fmt.Errorf("%w: failed to check admin existence: %v", sharedErrors.ErrQueryFailed, err)
	}
	return exists, nil
}

// CheckEmailVersionSync checks if email changed and increments token version
func (r *AuthRepository) CheckEmailVersionSync(ctx context.Context, userID uuid.UUID, newEmail string) (int32, error) {
	version, err := r.queries.CheckEmailVersionSync(ctx, db.CheckEmailVersionSyncParams{
		ID:    utils.UUIDToPgtype(userID),
		Email: newEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Email didn't change, return 0
			return 0, nil
		}
		return 0, fmt.Errorf("%w: failed to check email version sync: %v", sharedErrors.ErrQueryFailed, err)
	}
	if version == nil {
		return 0, nil
	}
	return *version, nil
}
