package service

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertSessionsMarshalEmpty pins the wire shape the schedule grids rely on:
// a course without sessions sends [], never null.
func assertSessionsMarshalEmpty(t *testing.T, sessions []dto.ScheduleSession) {
	t.Helper()
	raw, err := json.Marshal(sessions)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(raw))
}

func TestToScheduleSessions_NoSessions_MarshalsEmptyArray(t *testing.T) {
	assertSessionsMarshalEmpty(t, toScheduleSessions(nil))
}

func TestToScheduleSessions_SessionWithoutSlots_MarshalsEmptySlots(t *testing.T) {
	got := toScheduleSessions([]contracts.ScheduleSession{{DayOfWeek: "monday", SessionType: "lab"}})

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"day_of_week":"monday","slot_numbers":[],"session_type":"lab"}]`, string(raw))
}

func TestToScheduleSessions_ConvertsSlots(t *testing.T) {
	got := toScheduleSessions([]contracts.ScheduleSession{{DayOfWeek: "friday", SlotNumbers: []int16{3, 4}, SessionType: "theory"}})

	assert.Equal(t, []dto.ScheduleSession{{DayOfWeek: "friday", SlotNumbers: []int{3, 4}, SessionType: "theory"}}, got)
}
