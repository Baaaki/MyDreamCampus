package service

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// passwordUser lets ChangePassword run against loginUser.
type passwordUser struct {
	loginUser
}

func (passwordUser) UpdatePassword(context.Context, uuid.UUID, string, bool) error { return nil }

func (f *memorySessions) GetSessionsByUserID(context.Context, uuid.UUID) ([]db.Session, error) {
	out := make([]db.Session, 0, len(f.byJTI))
	for _, session := range f.byJTI {
		out = append(out, session)
	}
	return out, nil
}

func (f *memorySessions) DeleteAllUserSessions(context.Context, uuid.UUID) error {
	clear(f.byJTI)
	return nil
}

func onlySessionJTI(t *testing.T, sessions *memorySessions) string {
	t.Helper()
	require.Len(t, sessions.byJTI, 1)
	for jti := range sessions.byJTI {
		return jti
	}
	return ""
}

func sessionIDOf(t *testing.T, accessToken string) string {
	t.Helper()
	claims, err := utils.ValidateAccessTokenWithSecret(accessToken, []byte(tokenTestSecret))
	require.NoError(t, err)
	return claims.SessionID
}

// Every way of getting an access token must name the session it belongs to,
// or the sessions page cannot mark the caller's own entry.
func TestAccessToken_LoginRefreshChangePassword_CarriesSessionJTI(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	ctx := context.Background()
	hash, err := utils.HashPassword(ctx, "Sifre-12345")
	require.NoError(t, err)

	user := tokenTestUser()
	user.PasswordHash = hash
	user.IsActive = utils.BoolPtr(true)
	sessions := &memorySessions{byJTI: map[string]db.Session{}}
	s := newTokenTestService()
	s.authRepo = passwordUser{loginUser{user: user}}
	s.sessionRepo = sessions
	s.redisClient = newFakeAuthCache()

	login, refreshToken, err := s.Login(ctx, dto.LoginRequest{Email: user.Email, Password: "Sifre-12345"}, "test", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, onlySessionJTI(t, sessions), sessionIDOf(t, login.AccessToken), "login")

	refreshed, _, err := s.RefreshAccessToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.Equal(t, onlySessionJTI(t, sessions), sessionIDOf(t, refreshed.AccessToken), "refresh")

	changed, _, err := s.ChangePassword(ctx, utils.PgtypeToUUID(user.ID), dto.ChangePasswordRequest{
		OldPassword: "Sifre-12345",
		NewPassword: "Yeni-Sifre-67890",
	})
	require.NoError(t, err)
	assert.Equal(t, onlySessionJTI(t, sessions), sessionIDOf(t, changed.AccessToken), "change password")
}

func TestGetUserSessions_CallerSessionID_MarksOnlyThatSessionCurrent(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	ctx := context.Background()
	hash, err := utils.HashPassword(ctx, "Sifre-12345")
	require.NoError(t, err)

	user := tokenTestUser()
	user.PasswordHash = hash
	user.IsActive = utils.BoolPtr(true)
	sessions := &memorySessions{byJTI: map[string]db.Session{"other-device": {RefreshTokenJti: "other-device"}}}
	s := newTokenTestService()
	s.authRepo = loginUser{user: user}
	s.sessionRepo = sessions
	s.redisClient = newFakeAuthCache()

	login, _, err := s.Login(ctx, dto.LoginRequest{Email: user.Email, Password: "Sifre-12345"}, "test", "127.0.0.1")
	require.NoError(t, err)

	resp, err := s.GetUserSessions(ctx, utils.PgtypeToUUID(user.ID), sessionIDOf(t, login.AccessToken))
	require.NoError(t, err)
	require.Len(t, resp.Sessions, 2)
	current := 0
	for _, session := range resp.Sessions {
		if session.IsCurrent {
			current++
		}
	}
	assert.Equal(t, 1, current)
}
