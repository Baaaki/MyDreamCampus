// Package clocktest pins the service clock in tests.
package clocktest

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/clock/internal/source"
)

// Freeze makes clock.Now return exactly at until the test ends. It clears
// any simulation offset; calling it again moves the frozen instant.
func Freeze(tb testing.TB, at time.Time) {
	tb.Helper()
	source.Set(func() time.Time { return at })
	clock.Reset()
	tb.Cleanup(func() {
		source.Restore()
		clock.Reset()
	})
}
