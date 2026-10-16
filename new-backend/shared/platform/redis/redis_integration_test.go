//go:build integration

package redis

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient connects to the throwaway Redis `make test-integration`
// starts. Like pgtest, a missing address fails rather than skips: a tagged
// run that tested nothing must not read as green.
func newTestClient(t *testing.T) *ClientWrapper {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Fatal("TEST_REDIS_ADDR is not set; run `make test-integration` from the repository root")
	}
	require.NoError(t, logger.Init("test"))
	c, err := NewClient(addr, "", 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func uniqueKey(prefix string) string { return prefix + ":" + uuid.NewString() }

// concurrently runs fn n times at once and returns how many calls reported true.
func concurrently(t *testing.T, n int, fn func() (bool, error)) int {
	t.Helper()
	var wg sync.WaitGroup
	var mu sync.Mutex
	won := 0
	start := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := fn()
			assert.NoError(t, err)
			if ok {
				mu.Lock()
				won++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	return won
}

func TestCheckRateLimit_SlidingWindow(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := uniqueKey("rl")

	for want := 2; want >= 0; want-- {
		allowed, remaining, _, err := c.CheckRateLimit(ctx, key, 3, time.Second)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, want, remaining)
	}
	allowed, remaining, retryAfter, err := c.CheckRateLimit(ctx, key, 3, time.Second)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Zero(t, remaining)
	assert.Equal(t, 1, retryAfter, "seconds until the oldest request leaves the window, rounded up")

	time.Sleep(1100 * time.Millisecond)
	allowed, _, _, err = c.CheckRateLimit(ctx, key, 3, time.Second)
	require.NoError(t, err)
	assert.True(t, allowed, "the window slid past the first requests")
}

// The check and the count are one Lua script; if they were two calls, a
// burst would all see room and all get in.
func TestCheckRateLimit_BurstGetsExactlyTheLimit(t *testing.T) {
	c := newTestClient(t)
	key := uniqueKey("rl")

	admitted := concurrently(t, 25, func() (bool, error) {
		allowed, _, _, err := c.CheckRateLimit(context.Background(), key, 5, time.Minute)
		return allowed, err
	})

	assert.Equal(t, 5, admitted)
}

func TestConsumeResetToken_SingleUseUnderARace(t *testing.T) {
	c := newTestClient(t)
	token := uuid.NewString()
	require.NoError(t, c.StoreResetToken(context.Background(), token, "zeynep@uni.edu.tr", time.Minute))

	used := concurrently(t, 10, func() (bool, error) {
		email, err := c.ConsumeResetToken(context.Background(), token)
		return email == "zeynep@uni.edu.tr", err
	})

	assert.Equal(t, 1, used)
	email, err := c.ConsumeResetToken(context.Background(), "unknown")
	require.NoError(t, err)
	assert.Empty(t, email)
}

func TestPeekResetToken_LeavesTokenInPlace(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	token := uuid.NewString()
	require.NoError(t, c.StoreResetToken(ctx, token, "zeynep@uni.edu.tr", time.Minute))

	email, ttl, err := c.PeekResetToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "zeynep@uni.edu.tr", email)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, time.Minute)

	consumed, err := c.ConsumeResetToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "zeynep@uni.edu.tr", consumed, "a peek must not spend the token")

	email, ttl, err = c.PeekResetToken(ctx, token)
	require.NoError(t, err)
	assert.Empty(t, email)
	assert.Zero(t, ttl)
}

func TestRestoreResetToken_DoesNotOverwrite(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	token := uuid.NewString()

	require.NoError(t, c.RestoreResetToken(ctx, token, "zeynep@uni.edu.tr", time.Minute))
	require.NoError(t, c.RestoreResetToken(ctx, token, "baska@uni.edu.tr", time.Hour))

	email, ttl, err := c.PeekResetToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "zeynep@uni.edu.tr", email)
	assert.LessOrEqual(t, ttl, time.Minute)
}

func TestClaimIdempotencyKey_OneClaimUntilReleased(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := uniqueKey("idem")

	claimed := concurrently(t, 10, func() (bool, error) {
		return c.ClaimIdempotencyKey(context.Background(), key, "in-flight", time.Minute)
	})
	assert.Equal(t, 1, claimed)

	require.NoError(t, c.SaveIdempotencyRecord(ctx, key, `{"status":201}`, time.Minute))
	record, err := c.GetIdempotencyRecord(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, `{"status":201}`, record)

	require.NoError(t, c.ReleaseIdempotencyKey(ctx, key))
	ok, err := c.ClaimIdempotencyKey(ctx, key, "in-flight", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok, "a released key can be retried")
}

func TestLoginFailures_CountRestartsTheWindowAndClears(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := uniqueKey("login")

	for want := int64(1); want <= 3; want++ {
		n, err := c.RecordLoginFailure(ctx, key, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, want, n)
	}
	ttl, err := c.Client().TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.InDelta(t, time.Minute.Seconds(), ttl.Seconds(), 2, "the lockout runs from the latest failure")

	require.NoError(t, c.ClearLoginFailures(ctx, key))
	n, err := c.LoginFailureCount(ctx, key)
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestTokenRevocation(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	jti, user := uuid.NewString(), uuid.NewString()

	require.NoError(t, c.BlacklistAccessToken(ctx, jti, time.Minute))
	revoked, err := c.IsAccessTokenBlacklisted(ctx, jti)
	require.NoError(t, err)
	assert.True(t, revoked)
	revoked, err = c.IsAccessTokenBlacklisted(ctx, uuid.NewString())
	require.NoError(t, err)
	assert.False(t, revoked)

	v, err := c.GetMinTokenVersion(ctx, user)
	require.NoError(t, err)
	assert.Zero(t, v, "no revocation yet: every version is valid")
	require.NoError(t, c.BlacklistAllUserTokens(ctx, user, 3))
	v, err = c.GetMinTokenVersion(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, 3, v)
}

// The demo rewind runs an hour after the first visitor change; later
// changes must not push it back. The keys are fixed names, so the test
// clears them on both ends.
func TestDemoOps_FirstChangeIsKeptAndTheLockIsSeen(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	reset := func() { _ = c.Client().Del(context.Background(), "ops:dirty_since", "ops:write_lock").Err() }
	reset()
	t.Cleanup(reset)

	require.NoError(t, c.MarkChanged(ctx))
	first, err := c.Client().Get(ctx, "ops:dirty_since").Result()
	require.NoError(t, err)
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, c.MarkChanged(ctx))
	again, err := c.Client().Get(ctx, "ops:dirty_since").Result()
	require.NoError(t, err)
	assert.Equal(t, first, again)
	ttl, err := c.Client().TTL(ctx, "ops:dirty_since").Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl, "only demo-ops removes the stamp")

	locked, err := c.IsWriteLocked(ctx)
	require.NoError(t, err)
	assert.False(t, locked)
	require.NoError(t, c.Client().Set(ctx, "ops:write_lock", "1", 0).Err())
	locked, err = c.IsWriteLocked(ctx)
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestClockState_WatchersHearEveryChange(t *testing.T) {
	c := newTestClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key, channel := uniqueKey("clock"), uniqueKey("clock-changed")
	changes := c.WatchClockChanges(ctx, channel)

	// The subscription is confirmed asynchronously; keep announcing until
	// the watcher hears one rather than guessing how long that takes.
	heard := func(announce func() error) bool {
		deadline := time.After(3 * time.Second)
		for {
			require.NoError(t, announce())
			select {
			case <-changes:
				return true
			case <-deadline:
				return false
			case <-time.After(50 * time.Millisecond):
			}
		}
	}

	require.True(t, heard(func() error { return c.StoreClockState(ctx, key, channel, `{"offset":3600}`, 0) }))
	state, err := c.GetClockState(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, `{"offset":3600}`, state)

	require.True(t, heard(func() error { return c.DeleteClockState(ctx, key, channel) }))
	state, err = c.GetClockState(ctx, key)
	require.NoError(t, err)
	assert.Empty(t, state)

	cancel()
	for range changes {
	}
}
