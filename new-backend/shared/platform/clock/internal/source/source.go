// Package source holds the base time the service clock shifts. Production
// always reads the wall clock; only clocktest swaps it, so tests can pin an
// exact instant (a boundary like 03:00:00) that an offset over a moving
// clock would overshoot by the microseconds the test itself takes.
package source

import (
	"sync/atomic"
	"time"
)

var override atomic.Pointer[func() time.Time]

// Now returns the base time.
func Now() time.Time {
	if fn := override.Load(); fn != nil {
		return (*fn)()
	}
	return time.Now()
}

// Set replaces the base time.
func Set(fn func() time.Time) {
	override.Store(&fn)
}

// Restore returns to the wall clock.
func Restore() {
	override.Store(nil)
}
