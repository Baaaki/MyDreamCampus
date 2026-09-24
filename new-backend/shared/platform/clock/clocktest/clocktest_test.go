package clocktest

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/stretchr/testify/assert"
)

func TestFreeze_PinsExactInstant(t *testing.T) {
	at := time.Date(2026, 5, 2, 3, 0, 0, 0, time.UTC)

	t.Run("frozen", func(t *testing.T) {
		clock.SetOffset(time.Hour, time.Time{})
		Freeze(t, at)
		assert.Equal(t, at, clock.Now(), "Freeze must clear the offset")
		time.Sleep(2 * time.Millisecond)
		assert.Equal(t, at, clock.Now())
	})

	assert.WithinDuration(t, time.Now(), clock.Now(), time.Second,
		"cleanup must restore the real clock")
	assert.False(t, clock.State().Active)
}
