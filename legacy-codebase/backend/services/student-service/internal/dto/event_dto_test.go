package dto

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudentCreatedEvent_DeserializeFromWire(t *testing.T) {
	body := []byte(`{
		"event_id":"e-1",
		"event_type":"student.created",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"00000000-0000-0000-0000-000000000001",
			"student_number":"20240001",
			"first_name":"Ada","last_name":"Lovelace",
			"email":"ada@x.tr","faculty":"Eng","department":"CS",
			"enrollment_year":2024,"class_level":1,
			"status":"active"
		}
	}`)
	var ev StudentCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t, "student.created", ev.EventType)
	assert.Equal(t, "20240001", ev.Data.StudentNumber)
	assert.Equal(t, int16(1), ev.Data.ClassLevel)
	assert.Nil(t, ev.Data.AdvisorID, "missing advisor stays nil")
}

func TestStudentCreatedEvent_WithAdvisor(t *testing.T) {
	body := []byte(`{
		"event_id":"e","event_type":"student.created",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"00000000-0000-0000-0000-000000000001",
			"student_number":"x","first_name":"a","last_name":"b","email":"a@b.tr",
			"faculty":"f","department":"d","enrollment_year":2024,"class_level":1,
			"advisor_id":"00000000-0000-0000-0000-000000000099",
			"status":"active"
		}
	}`)
	var ev StudentCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	require.NotNil(t, ev.Data.AdvisorID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000099", ev.Data.AdvisorID.String())
}

func TestStudentUpdatedEvent_ChangedFieldsAny(t *testing.T) {
	body := []byte(`{
		"event_id":"e","event_type":"student.updated",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"id":"00000000-0000-0000-0000-000000000001",
			"student_number":"x",
			"changed_fields":{"class_level":3,"status":"active"}
		}
	}`)
	var ev StudentUpdatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.EqualValues(t, 3, ev.Data.ChangedFields["class_level"])
}

func TestStudentDeactivatedEvent(t *testing.T) {
	id := uuid.New()
	ev := StudentDeactivatedEvent{
		EventID: "e", EventType: "student.deactivated",
		Data: StudentDeactivatedData{ID: id, StudentNumber: "x", IsActive: false},
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var got StudentDeactivatedEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev.Data.ID, got.Data.ID)
	assert.False(t, got.Data.IsActive)
}

func TestStaffDeactivatedEvent_DeserializeFromWire(t *testing.T) {
	body := []byte(`{
		"event_id":"e","event_type":"staff.deactivated",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{
			"staff_id":"00000000-0000-0000-0000-000000000005",
			"is_active":false,
			"deleted_at":"2026-04-25T12:00:00Z"
		}
	}`)
	var ev StaffDeactivatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t, "00000000-0000-0000-0000-000000000005", ev.Data.StaffID.String())
}

func TestEvent_GenericMapPayload(t *testing.T) {
	body := []byte(`{
		"event_id":"e","event_type":"any",
		"timestamp":"2026-04-25T12:00:00Z",
		"data":{"foo":"bar","n":42}
	}`)
	var ev Event
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t, "bar", ev.Data["foo"])
	assert.EqualValues(t, 42, ev.Data["n"])
}
