package clock

import (
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/clock/internal/source"
	"github.com/stretchr/testify/assert"
)

var base = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

// pinBase fixes the real clock the offset is applied to.
func pinBase(t *testing.T, at *time.Time) {
	t.Helper()
	source.Set(func() time.Time { return *at })
	Reset()
	t.Cleanup(func() {
		source.Restore()
		Reset()
	})
}

func TestNow_NoSimulation_ReturnsRealTime(t *testing.T) {
	Reset()
	a := Now()
	time.Sleep(2 * time.Millisecond)
	b := Now()
	assert.True(t, b.After(a), "real clock must advance")
	assert.WithinDuration(t, time.Now(), b, time.Second)
}

func TestSetOffset_ShiftedTime_KeepsMoving(t *testing.T) {
	wall := base
	pinBase(t, &wall)

	SetOffset(365*24*time.Hour, time.Time{})
	assert.Equal(t, base.Add(365*24*time.Hour), Now())

	wall = base.Add(90 * time.Second)
	assert.Equal(t, base.Add(365*24*time.Hour+90*time.Second), Now(),
		"simulated time must advance with the real clock")
}

func TestSetOffset_NegativeOffset_GoesBack(t *testing.T) {
	wall := base
	pinBase(t, &wall)

	SetOffset(-48*time.Hour, time.Time{})
	assert.Equal(t, base.Add(-48*time.Hour), Now())
}

func TestSetOffset_UntilPassed_ReturnsRealTime(t *testing.T) {
	wall := base
	pinBase(t, &wall)

	until := base.Add(30 * time.Minute)
	SetOffset(time.Hour, until)
	assert.Equal(t, base.Add(time.Hour), Now())

	wall = until.Add(-time.Nanosecond)
	assert.Equal(t, wall.Add(time.Hour), Now(), "still active just before until")

	wall = until
	assert.Equal(t, until, Now(), "expires exactly at until")
	assert.False(t, State().Active)
}

func TestReset_EndsSimulation(t *testing.T) {
	wall := base
	pinBase(t, &wall)

	SetOffset(time.Hour, time.Time{})
	Reset()
	assert.Equal(t, base, Now())
	assert.False(t, State().Active)
}

func TestState_Inactive_ReportsRealTime(t *testing.T) {
	wall := base
	pinBase(t, &wall)

	assert.Equal(t, Snapshot{Now: base}, State())
}

func TestState_Active_ReportsOffsetAndUntil(t *testing.T) {
	wall := base
	pinBase(t, &wall)

	until := base.Add(time.Hour)
	SetOffset(-2*time.Hour, until)

	assert.Equal(t, Snapshot{
		Active: true,
		Offset: -2 * time.Hour,
		Now:    base.Add(-2 * time.Hour),
		Until:  until,
	}, State())
}

func TestClock_ConcurrentAccess_RaceFree(t *testing.T) {
	t.Cleanup(Reset)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch i % 3 {
			case 0:
				SetOffset(time.Duration(i)*time.Hour, time.Time{})
			case 1:
				Reset()
			default:
				_ = Now()
				_ = State()
			}
		}()
	}
	wg.Wait()
}
