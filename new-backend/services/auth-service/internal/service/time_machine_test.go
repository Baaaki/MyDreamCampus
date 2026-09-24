package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loginUser struct {
	userStore
	user db.User
}

func (f loginUser) GetUserByEmail(context.Context, string) (db.User, error) { return f.user, nil }
func (f loginUser) GetUserByID(context.Context, uuid.UUID) (db.User, error) { return f.user, nil }

// memorySessions keeps sessions by refresh JTI.
type memorySessions struct {
	sessionStore
	byJTI map[string]db.Session
}

func (f *memorySessions) CreateSession(_ context.Context, p db.CreateSessionParams) (db.Session, error) {
	session := db.Session{UserID: p.UserID, RefreshTokenJti: p.RefreshTokenJti, ExpiresAt: p.ExpiresAt}
	f.byJTI[p.RefreshTokenJti] = session
	return session, nil
}

func (f *memorySessions) GetSessionByJTI(_ context.Context, jti string) (db.Session, error) {
	session, ok := f.byJTI[jti]
	if !ok {
		return db.Session{}, serviceErrors.ErrSessionNotFoundRepo
	}
	return session, nil
}

func (f *memorySessions) RotateSession(ctx context.Context, oldJTI string, p db.CreateSessionParams) (db.Session, error) {
	delete(f.byJTI, oldJTI)
	return f.CreateSession(ctx, p)
}

// A simulated clock must not reach tokens or sessions: validation and
// sessions.sql compare against real time, so a shifted issue time would
// either log everyone out or keep a token alive for the whole offset.
func TestLogin_TimeMachineActive_TokensAndSessionOnRealClock(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	hash, err := utils.HashPassword("Sifre-12345")
	require.NoError(t, err)

	for _, offset := range []time.Duration{365 * 24 * time.Hour, -365 * 24 * time.Hour} {
		t.Run(offset.String(), func(t *testing.T) {
			clock.SetOffset(offset, time.Time{})
			t.Cleanup(clock.Reset)

			user := tokenTestUser()
			user.PasswordHash = hash
			user.IsActive = utils.BoolPtr(true)
			sessions := &memorySessions{byJTI: map[string]db.Session{}}
			s := newTokenTestService()
			s.authRepo = loginUser{user: user}
			s.sessionRepo = sessions
			s.redisClient = newFakeAuthCache()

			resp, refreshToken, err := s.Login(context.Background(),
				dto.LoginRequest{Email: user.Email, Password: "Sifre-12345"}, "test", "127.0.0.1")
			require.NoError(t, err)

			claims, err := utils.ValidateAccessTokenWithSecret(resp.AccessToken, []byte(tokenTestSecret))
			require.NoError(t, err, "a token issued under the time machine must validate")
			assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, time.Minute)

			require.Len(t, sessions.byJTI, 1)
			for _, session := range sessions.byJTI {
				assert.WithinDuration(t, time.Now().Add(24*time.Hour), session.ExpiresAt.Time, time.Minute)
			}

			_, rotated, err := s.RefreshAccessToken(context.Background(), refreshToken)
			require.NoError(t, err, "the refresh token must still be accepted")
			assert.NotEmpty(t, rotated)
		})
	}
}
