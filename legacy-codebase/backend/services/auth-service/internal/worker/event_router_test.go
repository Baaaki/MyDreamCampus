package worker

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the consumer-side parsing of incoming events. The router
// (event_consumer.go Start handler) reads BaseEvent.EventType and dispatches
// to a typed handler — if the wire types or constants drift, dispatch
// silently falls through to "unknown event type" and the user appears not
// to be syncable at all.

func TestBaseEvent_DispatchKeysMatchSharedEventsConstants(t *testing.T) {
	// Lock the wire-format event_type strings against the shared/events
	// constants used by every router. If these names drift apart, the
	// dispatch quietly fails on real production traffic.
	cases := map[string]string{
		"shared/events EventStudentCreated":      events.EventStudentCreated,
		"shared/events EventStaffCreated":        events.EventStaffCreated,
		"shared/events EventStudentUpdated":      events.EventStudentUpdated,
		"shared/events EventStaffUpdated":        events.EventStaffUpdated,
		"shared/events EventStudentDeactivated":  events.EventStudentDeactivated,
		"shared/events EventStaffDeactivated":    events.EventStaffDeactivated,
	}

	expected := map[string]string{
		"shared/events EventStudentCreated":     "student.created",
		"shared/events EventStaffCreated":       "staff.created",
		"shared/events EventStudentUpdated":     "student.updated",
		"shared/events EventStaffUpdated":       "staff.updated",
		"shared/events EventStudentDeactivated": "student.deactivated",
		"shared/events EventStaffDeactivated":   "staff.deactivated",
	}

	for label, got := range cases {
		assert.Equal(t, expected[label], got,
			"%s: drift detected. shared/events/events.go is the contract — never rename without coordinated migration.",
			label)
	}
}

func TestStudentCreatedEvent_ParsesRealisticPayload(t *testing.T) {
	body := mustMarshal(t, map[string]any{
		"event_id":   "11111111-1111-1111-1111-111111111111",
		"event_type": "student.created",
		"timestamp":  "2026-04-28T10:00:00Z",
		"data": map[string]any{
			"id":         "22222222-2222-2222-2222-222222222222",
			"email":      "ahmet@university.edu.tr",
			"first_name": "Ahmet",
			"last_name":  "Yilmaz",
			"department": "CS",
		},
	})

	var event dto.StudentCreatedEvent
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, "student.created", event.EventType,
		"event_type must round-trip — auth-service uses it to detect the routing key")
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", event.EventID)
	assert.Equal(t, "ahmet@university.edu.tr", event.Data.Email)
	assert.Equal(t, "Ahmet", event.Data.FirstName)
	assert.Equal(t, "CS", event.Data.Department,
		"department field must round-trip — auth uses it to populate the user's department for RBAC scoping")
}

func TestStaffCreatedEvent_ParsesRealisticPayload(t *testing.T) {
	body := mustMarshal(t, map[string]any{
		"event_id":   "11111111-1111-1111-1111-111111111111",
		"event_type": "staff.created",
		"timestamp":  "2026-04-28T10:00:00Z",
		"data": map[string]any{
			"id":         "22222222-2222-2222-2222-222222222222",
			"email":      "jane@university.edu.tr",
			"first_name": "Jane",
			"last_name":  "Doe",
			"role":       "teacher",
		},
	})

	var event dto.StaffCreatedEvent
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, "staff.created", event.EventType)
	assert.Equal(t, "jane@university.edu.tr", event.Data.Email)
	assert.Equal(t, "teacher", event.Data.Role,
		"role must round-trip — auth uses it to set the user's RBAC class")
}

func TestBaseEvent_RejectsBadJSON(t *testing.T) {
	var event dto.BaseEvent
	err := json.Unmarshal([]byte("not json"), &event)
	require.Error(t, err,
		"the consumer's first line of defence is the BaseEvent unmarshal — silent acceptance of garbage would acknowledge bad messages and lose them")
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
