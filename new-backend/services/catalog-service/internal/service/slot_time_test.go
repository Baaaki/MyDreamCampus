package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The clients draw the weekly grid from their own TIME_SLOTS; the teacher's
// course list prints these strings next to it, so a drift shows up as a
// lesson at two different hours on one screen.
func TestSlotTimeRange_EachSlot_MatchesBellSchedule(t *testing.T) {
	want := map[int16]string{
		1: "08:30-09:15",
		2: "09:25-10:10",
		3: "10:20-11:05",
		4: "11:15-12:00",
		5: "12:10-12:55",
		6: "13:00-13:45",
		7: "13:55-14:40",
		8: "14:50-15:35",
		9: "15:45-16:30",
	}
	for slot, time := range want {
		assert.Equal(t, time, slotTimeRange(slot), "slot %d", slot)
	}
}

func TestSlotTimeRange_OutOfRange_ReturnsEmpty(t *testing.T) {
	for _, slot := range []int16{-1, 0, 10} {
		assert.Empty(t, slotTimeRange(slot), "slot %d", slot)
	}
}
