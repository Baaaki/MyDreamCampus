package worker

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/stretchr/testify/assert"
)

// getNext3AM owns the cleanup-job schedule. It must always return 03:00 in
// the UTC+3 day, never earlier than "now". A regression here either skips a
// daily cleanup run (drifts to next day) or fires twice in one day.
//
// All cases freeze clock.Now at a chosen instant via the shared clock
// package, then assert the returned moment.

func TestGetNext3AM_BeforeThreeAM_SchedulesToday(t *testing.T) {
	utcPlus3 := time.FixedZone("UTC+3", 3*3600)

	// 2026-05-02 02:00 in UTC+3 → next run is *today* at 03:00.
	clock.Set(time.Date(2026, 5, 2, 2, 0, 0, 0, utcPlus3))
	t.Cleanup(clock.Reset)

	w := &ReservationWorker{}
	got := w.getNext3AM()

	want := time.Date(2026, 5, 2, 3, 0, 0, 0, utcPlus3)
	assert.True(t, got.Equal(want), "expected %v, got %v", want, got)
}

func TestGetNext3AM_AfterThreeAM_SchedulesTomorrow(t *testing.T) {
	utcPlus3 := time.FixedZone("UTC+3", 3*3600)

	// 2026-05-02 04:00 → already past 03:00 today, push to tomorrow 03:00.
	clock.Set(time.Date(2026, 5, 2, 4, 0, 0, 0, utcPlus3))
	t.Cleanup(clock.Reset)

	w := &ReservationWorker{}
	got := w.getNext3AM()

	want := time.Date(2026, 5, 3, 3, 0, 0, 0, utcPlus3)
	assert.True(t, got.Equal(want), "expected %v, got %v", want, got)
}

func TestGetNext3AM_LateNight_SchedulesTomorrow(t *testing.T) {
	utcPlus3 := time.FixedZone("UTC+3", 3*3600)

	// 23:59 — same day-rollover pattern. Pinning the boundary explicitly so
	// a future "off by a day" rewrite trips here.
	clock.Set(time.Date(2026, 5, 2, 23, 59, 0, 0, utcPlus3))
	t.Cleanup(clock.Reset)

	w := &ReservationWorker{}
	got := w.getNext3AM()

	want := time.Date(2026, 5, 3, 3, 0, 0, 0, utcPlus3)
	assert.True(t, got.Equal(want), "expected %v, got %v", want, got)
}

func TestGetNext3AM_ExactlyThreeAM_StaysToday(t *testing.T) {
	utcPlus3 := time.FixedZone("UTC+3", 3*3600)

	// At exactly 03:00 the comparison is `now.After(next3AM)` which is FALSE
	// for equal instants. So we schedule for *today* at 03:00 — i.e., now.
	// This is the contract: the timer fires immediately rather than waiting
	// 24h. Pinning so a refactor doesn't silently change to "After-or-equal".
	clock.Set(time.Date(2026, 5, 2, 3, 0, 0, 0, utcPlus3))
	t.Cleanup(clock.Reset)

	w := &ReservationWorker{}
	got := w.getNext3AM()

	want := time.Date(2026, 5, 2, 3, 0, 0, 0, utcPlus3)
	assert.True(t, got.Equal(want), "at exactly 03:00, schedule is the current instant — fires immediately")
}

func TestGetNext3AM_MonthRollover(t *testing.T) {
	utcPlus3 := time.FixedZone("UTC+3", 3*3600)

	// End of month at 23:00 → next run is on the *first of next month* at
	// 03:00. Verifies time.Add(24h) is consistent with date arithmetic at
	// month boundaries.
	clock.Set(time.Date(2026, 5, 31, 23, 0, 0, 0, utcPlus3))
	t.Cleanup(clock.Reset)

	w := &ReservationWorker{}
	got := w.getNext3AM()

	want := time.Date(2026, 6, 1, 3, 0, 0, 0, utcPlus3)
	assert.True(t, got.Equal(want), "expected %v, got %v", want, got)
}

func TestGetNext3AM_InputInUTC_StillReturnsUTCPlus3Zone(t *testing.T) {
	// clock.Set with a UTC time — getNext3AM converts to UTC+3 internally.
	// 2026-05-02 00:00 UTC == 2026-05-02 03:00 UTC+3, which is exactly 03:00
	// → schedule is today UTC+3 at 03:00 (i.e., now in UTC+3 terms).
	clock.Set(time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Reset)

	w := &ReservationWorker{}
	got := w.getNext3AM()

	utcPlus3 := time.FixedZone("UTC+3", 3*3600)
	want := time.Date(2026, 5, 2, 3, 0, 0, 0, utcPlus3)
	assert.True(t, got.Equal(want), "input timezone-agnostic; output anchored to UTC+3 day boundary")
}
