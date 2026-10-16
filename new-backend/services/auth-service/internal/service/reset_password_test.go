package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	resetEmail    = "ogrenci@uni.edu.tr"
	resetToken    = "reset-token-1"
	resetPassword = "Yeni-Sifre-67890"
)

// resetUser is the account a reset token points at; updateErr stands in for
// a failed password write.
type resetUser struct {
	loginUser
	updateErr error
	updated   int
}

func (f *resetUser) UpdatePassword(context.Context, uuid.UUID, string, bool) error {
	f.updated++
	return f.updateErr
}

func newResetTestService(t *testing.T) (*AuthService, *fakeAuthCache, *resetUser) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	user := tokenTestUser()
	user.Email = resetEmail
	user.IsActive = utils.BoolPtr(true)
	users := &resetUser{loginUser: loginUser{user: user}}
	cache := newFakeAuthCache()
	s := newTokenTestService()
	s.authRepo = users
	s.sessionRepo = &memorySessions{byJTI: map[string]db.Session{"jti-1": {}}}
	s.redisClient = cache
	require.NoError(t, cache.StoreResetToken(context.Background(), resetToken, resetEmail, time.Hour))
	return s, cache, users
}

func TestResetPassword_UnknownToken_ReturnsInvalidResetToken(t *testing.T) {
	s, cache, users := newResetTestService(t)

	err := s.ResetPassword(context.Background(), "not-a-token", resetPassword)

	assert.True(t, sharedErrors.Is(err, serviceErrors.ErrInvalidResetToken))
	assert.Zero(t, users.updated)
	assert.Equal(t, resetEmail, cache.resetTokens[resetToken], "someone else's token is untouched")
}

// Two resets on one link: both peek successfully, only the first to consume
// may change the password.
func TestResetPassword_TokenSpentBetweenPeekAndConsume_ReturnsInvalidResetToken(t *testing.T) {
	s, cache, users := newResetTestService(t)
	cache.beforeConsume = func() { delete(cache.resetTokens, resetToken) }

	err := s.ResetPassword(context.Background(), resetToken, resetPassword)

	assert.True(t, sharedErrors.Is(err, serviceErrors.ErrInvalidResetToken))
	assert.Zero(t, users.updated, "the losing reset must not write a password")
}

func TestResetPassword_UpdateFails_TokenRestored(t *testing.T) {
	s, cache, users := newResetTestService(t)
	users.updateErr = errors.New("connection reset")

	err := s.ResetPassword(context.Background(), resetToken, resetPassword)

	require.Error(t, err)
	assert.Equal(t, 1, users.updated)
	assert.Equal(t, resetEmail, cache.resetTokens[resetToken], "the user must be able to retry the same link")
	ttl := cache.resetTTLs[resetToken]
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, time.Hour, "a restored token must not outlive the original")
}

func TestResetPassword_HappyPath_SpendsTokenAndEndsSessions(t *testing.T) {
	s, cache, users := newResetTestService(t)
	sessions := s.sessionRepo.(*memorySessions)

	err := s.ResetPassword(context.Background(), resetToken, resetPassword)

	require.NoError(t, err)
	assert.Equal(t, 1, users.updated)
	assert.NotContains(t, cache.resetTokens, resetToken, "the link is single-use")
	assert.Empty(t, sessions.byJTI)

	err = s.ResetPassword(context.Background(), resetToken, resetPassword)
	assert.True(t, sharedErrors.Is(err, serviceErrors.ErrInvalidResetToken))
}
