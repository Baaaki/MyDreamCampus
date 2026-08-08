package clock

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNow_RealMode(t *testing.T) {
	Reset()
	defer Reset()

	assert.Equal(t, ModeReal, GetMode())
	a := Now()
	time.Sleep(2 * time.Millisecond)
	b := Now()
	assert.True(t, b.After(a), "real clock must advance")
}

func TestSet_FreezesTime(t *testing.T) {
	frozen := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	Set(frozen)
	defer Reset()

	assert.Equal(t, ModeSimulated, GetMode())
	assert.Equal(t, frozen, Now())

	time.Sleep(2 * time.Millisecond)
	assert.Equal(t, frozen, Now(), "simulated time must not advance with wall clock")
}

func TestReset_ReturnsToReal(t *testing.T) {
	Set(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	Reset()
	assert.Equal(t, ModeReal, GetMode())
	assert.Nil(t, SimulatedTime())
}

func TestSimulatedTime(t *testing.T) {
	defer Reset()

	Reset()
	assert.Nil(t, SimulatedTime(), "real mode must report nil simulated time")

	frozen := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	Set(frozen)
	got := SimulatedTime()
	assert.NotNil(t, got)
	assert.Equal(t, frozen, *got)
}

func TestClock_ConcurrentSafe(t *testing.T) {
	defer Reset()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				Set(time.Unix(int64(i), 0))
			} else {
				_ = Now()
				_ = GetMode()
			}
		}(i)
	}
	wg.Wait()
	// Test passes if race detector finds nothing.
}
