package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseEvent_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	e := BaseEvent{
		EventID:   "evt-1",
		EventType: "student.created",
		Timestamp: now,
	}
	data, err := json.Marshal(e)
	require.NoError(t, err)

	var got BaseEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, e, got)
}

func TestStudentCreatedEvent_DeserializeFromWire(t *testing.T) {
	body := []byte(`{
		"event_id":"e-1",
		"event_type":"student.created",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"u-1",
			"email":"s@university.tr",
			"first_name":"Ada",
			"last_name":"Lovelace",
			"department":"CS"
		}
	}`)
	var ev StudentCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t, "student.created", ev.EventType)
	assert.Equal(t, "u-1", ev.Data.ID)
	assert.Equal(t, "Ada", ev.Data.FirstName)
}

func TestStaffCreatedEvent_DeserializeFromWire(t *testing.T) {
	body := []byte(`{
		"event_id":"e-2",
		"event_type":"staff.created",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"u-2",
			"email":"t@university.tr",
			"role":"teacher",
			"first_name":"Alan",
			"last_name":"Turing",
			"department":"Math"
		}
	}`)
	var ev StaffCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t, "teacher", ev.Data.Role)
	assert.Equal(t, "Alan", ev.Data.FirstName)
}

func TestUserUpdatedEvent_ChangedFieldsMap(t *testing.T) {
	body := []byte(`{
		"event_id":"e-3",
		"event_type":"student.updated",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"u-1",
			"changed_fields":{"email":"new@x.tr","department":"EE"}
		}
	}`)
	var ev UserUpdatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t, "new@x.tr", ev.Data.ChangedFields["email"])
	assert.Equal(t, "EE", ev.Data.ChangedFields["department"])
}

func TestUserDeactivatedEvent(t *testing.T) {
	body := []byte(`{
		"event_id":"e-4",
		"event_type":"staff.deactivated",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"u-1",
			"is_active":false,
			"deleted_at":"2026-04-25T12:00:00Z"
		}
	}`)
	var ev UserDeactivatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.False(t, ev.Data.IsActive)
	assert.NotZero(t, ev.Data.DeletedAt)
}

func TestEvent_PartialJSONStillUnmarshals(t *testing.T) {
	// missing data block — Go zero-values it without error
	body := []byte(`{"event_id":"e","event_type":"student.created","timestamp":"2026-04-25T12:00:00Z"}`)
	var ev StudentCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Empty(t, ev.Data.ID)
}
