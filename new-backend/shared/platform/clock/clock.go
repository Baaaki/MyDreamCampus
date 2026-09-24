// Package clock is the service clock for business rules: semester
// deadlines, reservation windows, attendance expiry, grading periods.
//
// The time machine shifts it by an offset rather than freezing it, so the
// simulated time keeps moving — a frozen clock never rotated attendance QR
// codes and kept them valid forever. Security and infrastructure timestamps
// (tokens, sessions, rate limits, idempotency, outbox retries, Redis TTLs)
// must use time.Now instead: shifting those would log everyone out or
// replay work the moment the clock moves forward.
package clock

import (
	"sync/atomic"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/clock/internal/source"
)

type setting struct {
	offset time.Duration
	until  time.Time
}

// activeAt reports whether the setting still applies at the real instant now.
func (s *setting) activeAt(now time.Time) bool {
	return s.until.IsZero() || now.Before(s.until)
}

// Read on every request, written a few times a day at most.
var current atomic.Pointer[setting]

// Now returns the real time, shifted by the offset while a simulation is
// active.
func Now() time.Time {
	now := source.Now()
	if s := current.Load(); s != nil && s.activeAt(now) {
		return now.Add(s.offset)
	}
	return now
}

// SetOffset starts a simulation: Now returns the real time plus offset until
// the real clock reaches until. A zero until never expires.
func SetOffset(offset time.Duration, until time.Time) {
	current.Store(&setting{offset: offset, until: until})
}

// Reset ends the simulation.
func Reset() {
	current.Store(nil)
}

// Snapshot describes the clock at one instant.
type Snapshot struct {
	// Active is false when no simulation was set or its until has passed.
	Active bool
	// Offset is zero when inactive.
	Offset time.Duration
	// Now is what clock.Now returned at the moment of the snapshot.
	Now time.Time
	// Until is zero when inactive or when the simulation never expires.
	Until time.Time
}

// State returns a consistent snapshot; reading Now and the offset through
// separate calls could straddle a SetOffset.
func State() Snapshot {
	now := source.Now()
	s := current.Load()
	if s == nil || !s.activeAt(now) {
		return Snapshot{Now: now}
	}
	return Snapshot{
		Active: true,
		Offset: s.offset,
		Now:    now.Add(s.offset),
		Until:  s.until,
	}
}
