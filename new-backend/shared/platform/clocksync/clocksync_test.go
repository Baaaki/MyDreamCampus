package clocksync

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBackend struct {
	mu        sync.Mutex
	value     string
	ttl       time.Duration
	getErr    error
	published []string
	changes   chan struct{}
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{changes: make(chan struct{}, 1)}
}

func (f *fakeBackend) GetClockState(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if key != StateKey {
		return "", nil
	}
	return f.value, f.getErr
}

func (f *fakeBackend) StoreClockState(_ context.Context, key, channel, value string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if key == StateKey {
		f.value, f.ttl = value, ttl
	}
	f.published = append(f.published, channel)
	return nil
}

func (f *fakeBackend) DeleteClockState(_ context.Context, key, channel string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if key == StateKey {
		f.value, f.ttl = "", 0
	}
	f.published = append(f.published, channel)
	return nil
}

func (f *fakeBackend) WatchClockChanges(context.Context, string) <-chan struct{} {
	return f.changes
}

// set changes the stored value the way another service's Publish would,
// without signalling.
func (f *fakeBackend) set(t *testing.T, state *State) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if state == nil {
		f.value = ""
		return
	}
	raw, err := json.Marshal(state)
	require.NoError(t, err)
	f.value = string(raw)
}

func (f *fakeBackend) setGetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getErr = err
}

func setup(t *testing.T) *fakeBackend {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	clock.Reset()
	t.Cleanup(clock.Reset)
	return newFakeBackend()
}

func offsetIs(want time.Duration) func() bool {
	return func() bool {
		s := clock.State()
		return s.Active && s.Offset == want
	}
}

func TestStart_StoredState_AppliesBeforeReturning(t *testing.T) {
	b := setup(t)
	b.set(t, &State{OffsetSeconds: 3600})

	Start(t.Context(), b)

	assert.True(t, offsetIs(time.Hour)(), "the first request must already see the shared clock")
}

func TestStart_NoStoredState_UsesRealTime(t *testing.T) {
	b := setup(t)
	clock.SetOffset(time.Hour, time.Time{})

	Start(t.Context(), b)

	assert.False(t, clock.State().Active)
}

func TestStart_ChangeSignal_AppliesNewState(t *testing.T) {
	b := setup(t)
	Start(t.Context(), b)

	b.set(t, &State{OffsetSeconds: -7200})
	b.changes <- struct{}{}

	assert.Eventually(t, offsetIs(-2*time.Hour), time.Second, 5*time.Millisecond)
}

func TestStart_ChangeSignalAfterClear_ReturnsToRealTime(t *testing.T) {
	b := setup(t)
	b.set(t, &State{OffsetSeconds: 60})
	Start(t.Context(), b)
	require.True(t, clock.State().Active)

	b.set(t, nil)
	b.changes <- struct{}{}

	assert.Eventually(t, func() bool { return !clock.State().Active }, time.Second, 5*time.Millisecond)
}

func TestStart_StoredUntil_ExpiresWithoutMessage(t *testing.T) {
	b := setup(t)
	until := time.Now().Add(50 * time.Millisecond)
	b.set(t, &State{OffsetSeconds: 60, Until: &until})

	Start(t.Context(), b)
	require.True(t, clock.State().Active)

	assert.Eventually(t, func() bool { return !clock.State().Active }, time.Second, 5*time.Millisecond)
}

func TestSyncer_KeyRemovedWithoutMessage_PeriodicReadResets(t *testing.T) {
	b := setup(t)
	b.set(t, &State{OffsetSeconds: 60})
	s := &syncer{backend: b, interval: 10 * time.Millisecond}
	s.resync(t.Context())
	require.True(t, clock.State().Active)

	go s.run(t.Context(), nil)
	// A Redis flush deletes the key and sends nothing.
	b.set(t, nil)

	assert.Eventually(t, func() bool { return !clock.State().Active }, time.Second, 5*time.Millisecond)
}

func TestSyncer_ReadError_KeepsCurrentClock(t *testing.T) {
	b := setup(t)
	b.set(t, &State{OffsetSeconds: 60})
	s := &syncer{backend: b, interval: time.Hour}
	s.resync(t.Context())

	b.setGetErr(errors.New("connection refused"))
	s.resync(t.Context())

	assert.True(t, offsetIs(time.Minute)(), "a Redis hiccup must not flip simulated deadlines back")
	assert.True(t, s.failing)

	b.setGetErr(nil)
	b.set(t, nil)
	s.resync(t.Context())
	assert.False(t, clock.State().Active)
	assert.False(t, s.failing)
}

func TestSyncer_InvalidState_UsesRealTime(t *testing.T) {
	b := setup(t)
	clock.SetOffset(time.Hour, time.Time{})
	b.value = "{not json"
	s := &syncer{backend: b, interval: time.Hour}

	s.resync(t.Context())

	assert.False(t, clock.State().Active)
}

func TestPublish_NoUntil_StoresWithoutTTLAndAppliesLocally(t *testing.T) {
	b := setup(t)
	setAt := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)

	err := Publish(t.Context(), b, State{OffsetSeconds: 86400, SetAt: setAt, SetBy: "admin-1"})
	require.NoError(t, err)

	assert.Equal(t, time.Duration(0), b.ttl)
	assert.Equal(t, []string{Channel}, b.published)
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(b.value), &stored))
	assert.Equal(t, map[string]any{
		"offset_seconds": float64(86400),
		"set_at":         "2026-09-24T10:00:00Z",
		"set_by":         "admin-1",
	}, stored)
	assert.True(t, offsetIs(24*time.Hour)(), "the caller's own response must reflect the change")
}

func TestPublish_WithUntil_KeyExpiresWithSimulation(t *testing.T) {
	b := setup(t)
	until := time.Now().Add(30 * time.Minute)

	require.NoError(t, Publish(t.Context(), b, State{OffsetSeconds: 60, Until: &until}))

	assert.InDelta(t, 30*time.Minute, b.ttl, float64(time.Second))
	assert.Equal(t, until, clock.State().Until)
}

func TestPublish_UntilPassed_Clears(t *testing.T) {
	b := setup(t)
	b.set(t, &State{OffsetSeconds: 60})
	clock.SetOffset(time.Minute, time.Time{})
	past := time.Now().Add(-time.Second)

	require.NoError(t, Publish(t.Context(), b, State{OffsetSeconds: 60, Until: &past}))

	assert.Empty(t, b.value)
	assert.False(t, clock.State().Active)
}

func TestClear_DeletesAndResetsLocally(t *testing.T) {
	b := setup(t)
	require.NoError(t, Publish(t.Context(), b, State{OffsetSeconds: 60}))

	require.NoError(t, Clear(t.Context(), b))

	assert.Empty(t, b.value)
	assert.Equal(t, []string{Channel, Channel}, b.published)
	assert.False(t, clock.State().Active)
}
