// Package clocksync keeps the time machine's offset identical in every
// service. The state lives in one Redis key; a pub/sub message tells the
// services to re-read it, and a periodic re-read covers messages missed
// during a reconnect and keys removed without a message (the nightly
// reset flushes Redis). Without Redis a service keeps the real clock.
package clocksync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

const (
	// StateKey holds the JSON-encoded State.
	StateKey = "clock:state"
	// Channel carries a signal after every change to StateKey.
	Channel = "clock:changed"

	resyncInterval = 10 * time.Second
	readTimeout    = 3 * time.Second
)

// Backend is the Redis surface clocksync needs; platform/redis implements it.
type Backend interface {
	GetClockState(ctx context.Context, key string) (string, error)
	StoreClockState(ctx context.Context, key, channel, value string, ttl time.Duration) error
	DeleteClockState(ctx context.Context, key, channel string) error
	WatchClockChanges(ctx context.Context, channel string) <-chan struct{}
}

// State is the cluster-wide simulation.
type State struct {
	OffsetSeconds int64 `json:"offset_seconds"`
	// Until is the real instant the simulation ends; nil never expires.
	Until *time.Time `json:"until,omitempty"`
	SetAt time.Time  `json:"set_at"`
	SetBy string     `json:"set_by"`
}

// Offset returns the offset as a duration.
func (s State) Offset() time.Duration {
	return time.Duration(s.OffsetSeconds) * time.Second
}

// Start applies the stored state before returning, so the first request
// already sees the shared clock, then follows changes until ctx ends.
func Start(ctx context.Context, b Backend) {
	s := &syncer{backend: b, interval: resyncInterval}
	s.resync(ctx)
	changes := b.WatchClockChanges(ctx, Channel)
	go s.run(ctx, changes)
}

// Publish stores the state for every service and applies it here at once,
// so the caller's own response already reflects it.
func Publish(ctx context.Context, b Backend, state State) error {
	var ttl time.Duration
	if state.Until != nil {
		// The key's TTL is a real-clock duration, like Until itself.
		ttl = time.Until(*state.Until)
		if ttl <= 0 {
			return Clear(ctx, b)
		}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal clock state: %w", err)
	}
	if err := b.StoreClockState(ctx, StateKey, Channel, string(raw), ttl); err != nil {
		return fmt.Errorf("store clock state: %w", err)
	}
	apply(&state)
	return nil
}

// Clear returns every service to the real clock.
func Clear(ctx context.Context, b Backend) error {
	if err := b.DeleteClockState(ctx, StateKey, Channel); err != nil {
		return fmt.Errorf("delete clock state: %w", err)
	}
	clock.Reset()
	return nil
}

type syncer struct {
	backend  Backend
	interval time.Duration
	failing  bool
}

func (s *syncer) run(ctx context.Context, changes <-chan struct{}) {
	// Real time on purpose: a ticker is infrastructure, not a business rule.
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-changes:
			if !ok {
				// The periodic re-read keeps the clock in step on its own.
				changes = nil
				continue
			}
			s.resync(ctx)
		case <-ticker.C:
			s.resync(ctx)
		}
	}
}

// resync reads the key and applies it. A read error keeps the clock as it
// is: flipping to real time on every Redis hiccup would make simulated
// deadlines flap. The error is logged once per outage, not every tick.
func (s *syncer) resync(ctx context.Context) {
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	raw, err := s.backend.GetClockState(readCtx, StateKey)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		if !s.failing {
			logger.Warn("clock state unreadable, keeping current clock", zap.Error(err))
			s.failing = true
		}
		return
	}
	if s.failing {
		logger.Info("clock state readable again")
		s.failing = false
	}

	if raw == "" {
		apply(nil)
		return
	}
	var state State
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		// Only this package writes the key; a value it cannot parse is not
		// a simulation anyone asked for.
		logger.Error("invalid clock state, using real time", zap.Error(err))
		apply(nil)
		return
	}
	apply(&state)
}

func apply(state *State) {
	if state == nil {
		clock.Reset()
		return
	}
	var until time.Time
	if state.Until != nil {
		until = *state.Until
	}
	clock.SetOffset(state.Offset(), until)
}
