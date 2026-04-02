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

type SessionRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateSession creates a new session
func (r *SessionRepository) CreateSession(ctx context.Context, params db.CreateSessionParams) (db.Session, error) {
	session, err := r.queries.CreateSession(ctx, params)
	if err != nil {
		return db.Session{}, fmt.Errorf("failed to create session: %w", err)
	}
	return session, nil
}

// GetSessionByJTI retrieves a session by JWT ID
func (r *SessionRepository) GetSessionByJTI(ctx context.Context, jti string) (db.Session, error) {
	session, err := r.queries.GetSessionByJTI(ctx, jti)
	if err == pgx.ErrNoRows {
		return db.Session{}, fmt.Errorf("session not found")
	}
	if err != nil {
		return db.Session{}, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// GetSessionsByUserID retrieves all sessions for a user
func (r *SessionRepository) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]db.Session, error) {
	sessions, err := r.queries.GetSessionsByUserID(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	return sessions, nil
}

// UpdateSessionLastUsed updates the last_used_at timestamp
func (r *SessionRepository) UpdateSessionLastUsed(ctx context.Context, jti string) error {
	err := r.queries.UpdateSessionLastUsed(ctx, jti)
	if err != nil {
		return fmt.Errorf("failed to update session last used: %w", err)
	}
	return nil
}

// DeleteSession deletes a session by JTI
func (r *SessionRepository) DeleteSession(ctx context.Context, jti string) error {
	err := r.queries.DeleteSession(ctx, jti)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteSessionByID deletes a session by ID (for user-initiated deletion)
func (r *SessionRepository) DeleteSessionByID(ctx context.Context, sessionID, userID uuid.UUID) error {
	err := r.queries.DeleteSessionByID(ctx, db.DeleteSessionByIDParams{
		ID:     utils.UUIDToPgtype(sessionID),
		UserID: utils.UUIDToPgtype(userID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteAllUserSessions deletes all sessions for a user
func (r *SessionRepository) DeleteAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.DeleteAllUserSessions(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("failed to delete all user sessions: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (r *SessionRepository) CleanupExpiredSessions(ctx context.Context) error {
	err := r.queries.CleanupExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}
