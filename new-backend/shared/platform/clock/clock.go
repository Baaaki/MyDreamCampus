package clock

import (
	"sync"
	"time"
)

// Mode represents the clock operating mode.
type Mode string

const (
	ModeReal      Mode = "real"
	ModeSimulated Mode = "simulated"
)

// systemClock is the package-level singleton.
// Read-heavy, write-rare pattern — uses RWMutex.
var systemClock = &clockState{
	mode: ModeReal,
}

type clockState struct {
	mu        sync.RWMutex
	mode      Mode
	fixedTime time.Time
}

// Now returns the current time based on the active mode.
// In Real mode, it returns time.Now().
// In Simulated mode, it returns the frozen simulated time.
func Now() time.Time {
	systemClock.mu.RLock()
	defer systemClock.mu.RUnlock()

	if systemClock.mode == ModeSimulated {
		return systemClock.fixedTime
	}
	return time.Now()
}

// Set switches to Simulated mode and freezes time at the given value.
func Set(t time.Time) {
	systemClock.mu.Lock()
	defer systemClock.mu.Unlock()

	systemClock.mode = ModeSimulated
	systemClock.fixedTime = t
}

// Reset switches back to Real mode (time.Now()).
func Reset() {
	systemClock.mu.Lock()
	defer systemClock.mu.Unlock()

	systemClock.mode = ModeReal
	systemClock.fixedTime = time.Time{}
}

// GetMode returns the current clock mode ("real" or "simulated").
func GetMode() Mode {
	systemClock.mu.RLock()
	defer systemClock.mu.RUnlock()

	return systemClock.mode
}

// SimulatedTime returns the simulated time if in Simulated mode, or nil if in Real mode.
func SimulatedTime() *time.Time {
	systemClock.mu.RLock()
	defer systemClock.mu.RUnlock()

	if systemClock.mode == ModeSimulated {
		t := systemClock.fixedTime
		return &t
	}
	return nil
}
