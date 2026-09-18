package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// fakeAuthCache keeps login failures in a map; the other authCache methods
// are not exercised by the lockout path.
type fakeAuthCache struct {
	failures   map[string]int64
	windows    map[string]time.Duration
	countErr   error
	minVersion map[string]int
}

func newFakeAuthCache() *fakeAuthCache {
	return &fakeAuthCache{
		failures:   map[string]int64{},
		windows:    map[string]time.Duration{},
		minVersion: map[string]int{},
	}
}

func (f *fakeAuthCache) BlacklistAccessToken(context.Context, string, time.Duration) error {
	return nil
}

func (f *fakeAuthCache) BlacklistAllUserTokens(_ context.Context, userID string, v int) error {
	f.minVersion[userID] = v
	return nil
}

func (f *fakeAuthCache) LoginFailureCount(_ context.Context, key string) (int64, error) {
	return f.failures[key], f.countErr
}

func (f *fakeAuthCache) RecordLoginFailure(_ context.Context, key string, window time.Duration) (int64, error) {
	f.failures[key]++
	f.windows[key] = window
	return f.failures[key], nil
}

func (f *fakeAuthCache) ClearLoginFailures(_ context.Context, key string) error {
	delete(f.failures, key)
	return nil
}

func (f *fakeAuthCache) StoreResetToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (f *fakeAuthCache) ConsumeResetToken(context.Context, string) (string, error) {
	return "", nil
}

func newLockoutService(cache *fakeAuthCache, lockMinutes int) *AuthService {
	cfg := &config.Config{}
	cfg.Timeout.AccountLockDurationMinutes = lockMinutes
	return &AuthService{config: cfg, redisClient: cache}
}

func TestLoginLockout_FifthFailure_LocksThatAddressOnly(t *testing.T) {
	cache := newFakeAuthCache()
	s := newLockoutService(cache, 30)
	ctx := context.Background()
	attacker := loginFailureKey("admin@uni.edu.tr", "203.0.113.9")
	owner := loginFailureKey("admin@uni.edu.tr", "198.51.100.4")

	for range maxLoginFailures {
		assert.False(t, s.loginLockedOut(ctx, attacker, zap.NewNop()))
		s.recordLoginFailure(ctx, attacker, zap.NewNop())
	}

	assert.True(t, s.loginLockedOut(ctx, attacker, zap.NewNop()), "attacker's address must be locked out")
	assert.False(t, s.loginLockedOut(ctx, owner, zap.NewNop()), "the owner on another address must still be able to log in")
}

func TestLoginLockout_WindowComesFromConfig(t *testing.T) {
	cache := newFakeAuthCache()
	s := newLockoutService(cache, 7)
	key := loginFailureKey("a@uni.edu.tr", "203.0.113.9")

	s.recordLoginFailure(context.Background(), key, zap.NewNop())

	assert.Equal(t, 7*time.Minute, cache.windows[key])
}

func TestLoginLockout_UnsetDuration_DefaultsTo30Minutes(t *testing.T) {
	s := newLockoutService(newFakeAuthCache(), 0)
	assert.Equal(t, 30*time.Minute, s.lockDuration())
}

func TestLoginLockout_RedisError_FailsOpen(t *testing.T) {
	cache := newFakeAuthCache()
	cache.countErr = errors.New("redis down")
	s := newLockoutService(cache, 30)

	assert.False(t, s.loginLockedOut(context.Background(), "any", zap.NewNop()))
}

func TestLoginFailureKey_NormalisesEmail(t *testing.T) {
	assert.Equal(t,
		loginFailureKey("Admin@Uni.edu.tr ", "203.0.113.9"),
		loginFailureKey("admin@uni.edu.tr", "203.0.113.9"))
	assert.NotContains(t, loginFailureKey("admin@uni.edu.tr", "203.0.113.9"), "admin@")
}

func TestRevokeOutstandingAccessTokens_RaisesMinimumVersion(t *testing.T) {
	cache := newFakeAuthCache()
	s := newLockoutService(cache, 30)

	s.revokeOutstandingAccessTokens(context.Background(), "user-1", 4)

	assert.Equal(t, 4, cache.minVersion["user-1"])
}
