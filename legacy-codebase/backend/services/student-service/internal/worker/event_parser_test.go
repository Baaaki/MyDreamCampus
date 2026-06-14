package worker

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// student-service consumes staff events to maintain its advisor cache.
// These tests pin the wire-format expectations of that consumer.

func TestEvent_DispatchKeysMatchSharedEventsConstants(t *testing.T) {
	assert.Equal(t, "staff.created", events.EventStaffCreated,
		"shared/events constant must match the producer's wire format — never rename without coordination across services")
	assert.Equal(t, "staff.updated", events.EventStaffUpdated)
	assert.Equal(t, "staff.deactivated", events.EventStaffDeactivated)
}

func TestEvent_ParsesGenericEnvelope(t *testing.T) {
	body := mustMarshal(t, map[string]any{
		"event_id":   "11111111-1111-1111-1111-111111111111",
		"event_type": "staff.created",
		"timestamp":  "2026-04-28T10:00:00Z",
		"data": map[string]any{
			"id":         "22222222-2222-2222-2222-222222222222",
			"email":      "jane@university.edu.tr",
			"first_name": "Jane",
			"last_name":  "Doe",
		},
	})

	var event dto.Event
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, "11111111-1111-1111-1111-111111111111", event.EventID,
		"event_id is the idempotency key — drift breaks dedup")
	assert.Equal(t, "staff.created", event.EventType,
		"event_type drives the dispatch — drift silently routes to default branch")

	require.NotNil(t, event.Data, "data must round-trip as map[string]any so handlers can re-marshal it into typed structs")
	assert.Equal(t, "jane@university.edu.tr", event.Data["email"])
}

func TestEvent_BadJSONFailsLoudly(t *testing.T) {
	var event dto.Event
	err := json.Unmarshal([]byte("garbage"), &event)
	require.Error(t, err,
		"the consumer's first line is the BaseEvent unmarshal — silent acceptance would ack and lose the message")
}

func TestEvent_DataAsMapPreservesAllKeys(t *testing.T) {
	// student-service worker uses map[string]any for Data so it can do
	// a second-pass typed unmarshal per event type. This test pins that
	// a key the worker doesn't know about today (e.g. a producer-added
	// field) survives the first decode rather than getting dropped.
	body := mustMarshal(t, map[string]any{
		"event_id":   "11111111-1111-1111-1111-111111111111",
		"event_type": "staff.created",
		"timestamp":  "2026-04-28T10:00:00Z",
		"data": map[string]any{
			"id":            "22222222-2222-2222-2222-222222222222",
			"email":         "jane@univ.edu",
			"future_field":  "added by a newer producer",
		},
	})

	var event dto.Event
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, "added by a newer producer", event.Data["future_field"],
		"forward compatibility: an unknown key must NOT be dropped during base parsing — typed handlers may add support without consumer redeploy")
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
