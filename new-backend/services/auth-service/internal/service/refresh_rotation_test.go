package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staleSessions mimics two refreshes that both read the session before
// either rotates it: GetSessionByJTI keeps answering from the first read,
// while RotateSession applies the row-count check of DeleteSessionByJTI.
type staleSessions struct {
	sessionStore
	snapshot map[string]db.Session
	live     map[string]db.Session
}

func (f *staleSessions) GetSessionByJTI(_ context.Context, jti string) (db.Session, error) {
	session, ok := f.snapshot[jti]
	if !ok {
		return db.Session{}, serviceErrors.ErrSessionNotFoundRepo
	}
	return session, nil
}

func (f *staleSessions) RotateSession(_ context.Context, oldJTI string, params db.CreateSessionParams) (db.Session, error) {
	if _, ok := f.live[oldJTI]; !ok {
		return db.Session{}, fmt.Errorf("%w: already rotated", serviceErrors.ErrSessionNotFoundRepo)
	}
	delete(f.live, oldJTI)
	session := db.Session{UserID: params.UserID, RefreshTokenJti: params.RefreshTokenJti}
	f.live[params.RefreshTokenJti] = session
	return session, nil
}

type fixedUser struct {
	userStore
	user db.User
}

func (f fixedUser) GetUserByID(context.Context, uuid.UUID) (db.User, error) { return f.user, nil }

func TestRefreshAccessToken_SameTokenTwice_SecondGetsSessionNotFound(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	s := newTokenTestService()
	user := tokenTestUser()
	s.authRepo = fixedUser{user: user}

	refreshToken, jti, err := s.generateRefreshToken(user)
	require.NoError(t, err)
	session := db.Session{UserID: user.ID, RefreshTokenJti: jti}
	sessions := &staleSessions{
		snapshot: map[string]db.Session{jti: session},
		live:     map[string]db.Session{jti: session},
	}
	s.sessionRepo = sessions

	_, rotated, err := s.RefreshAccessToken(context.Background(), refreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, rotated)

	_, _, err = s.RefreshAccessToken(context.Background(), refreshToken)
	assert.ErrorIs(t, err, serviceErrors.ErrSessionNotFound)
	assert.Len(t, sessions.live, 1, "the losing refresh must not create a second session")
	assert.NotContains(t, sessions.live, jti)
}
