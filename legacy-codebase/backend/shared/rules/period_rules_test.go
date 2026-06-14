package rules

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/stretchr/testify/assert"
)

func TestIsWithinPeriod(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		now         time.Time
		wantAllowed bool
		wantReason  string
	}{
		{"before start", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), false, "not started"},
		{"on start", start.Add(time.Second), true, "within"},
		{"middle", time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), true, "within"},
		{"after end", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), false, "ended"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock.Set(tt.now)
			defer clock.Reset()
			res := IsWithinPeriod(start, end)
			assert.Equal(t, tt.wantAllowed, res.Allowed)
			assert.Contains(t, res.Reason, tt.wantReason)
			assert.Equal(t, end, res.EffectiveDeadline)
		})
	}
}
