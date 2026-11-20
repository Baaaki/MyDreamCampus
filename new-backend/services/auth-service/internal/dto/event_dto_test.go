package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDeactivatedEvent_UnmarshalStaffIDAndID(t *testing.T) {
	var staffEv UserDeactivatedEvent
	require.NoError(t, json.Unmarshal([]byte(`{
		"event_id":"ev-1","event_type":"staff.deactivated",
		"data":{"staff_id":"11111111-1111-1111-1111-111111111111"}
	}`), &staffEv))
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", staffEv.Data.EffectiveID())

	var studentEv UserDeactivatedEvent
	require.NoError(t, json.Unmarshal([]byte(`{
		"event_id":"ev-2","event_type":"student.deactivated",
		"data":{"id":"22222222-2222-2222-2222-222222222222"}
	}`), &studentEv))
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", studentEv.Data.EffectiveID())
}

func TestUserUpdatedEvent_UnmarshalMixedChangedFieldsAndStaffID(t *testing.T) {
	var studentEv UserUpdatedEvent
	require.NoError(t, json.Unmarshal([]byte(`{
		"event_id":"ev-3","event_type":"student.updated",
		"data":{
			"id":"22222222-2222-2222-2222-222222222222",
			"changed_fields":{"email":"new@uni.edu.tr","class_level":3}
		}
	}`), &studentEv))
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", studentEv.Data.EffectiveID())
	assert.Equal(t, "new@uni.edu.tr", studentEv.Data.ChangedFields["email"])

	var staffEv UserUpdatedEvent
	require.NoError(t, json.Unmarshal([]byte(`{
		"event_id":"ev-4","event_type":"staff.updated",
		"data":{"staff_id":"11111111-1111-1111-1111-111111111111","department":"CS"}
	}`), &staffEv))
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", staffEv.Data.EffectiveID())
	require.NotNil(t, staffEv.Data.Department)
	assert.Equal(t, "CS", *staffEv.Data.Department)
}
