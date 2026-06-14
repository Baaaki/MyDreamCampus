package service

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pin the wire contract for staff.* outbox events. Auth-service consumes
// these to maintain its user projection — a silent rename here breaks
// login for every newly-created staff member.

func TestBuildStaffCreatedPayload_ContractKeys(t *testing.T) {
	payload := buildStaffCreatedPayload(dto.CreateStaffRequest{
		Email:          "jane@univ.edu",
		FirstName:      "Jane",
		LastName:       "Doe",
		Role:           "teacher",
		Department:     "Computer Science",
		Phone:          "555-1234",
		OfficeLocation: "B-204",
	})

	required := []string{"id", "email", "first_name", "last_name", "role", "department"}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Nil(t, payload["id"], "id is filled by repository after insert — staying nil is the contract")
	assert.Equal(t, "jane@univ.edu", payload["email"])
	assert.Equal(t, "Jane", payload["first_name"])
	assert.Equal(t, "Doe", payload["last_name"])
	assert.Equal(t, "teacher", payload["role"])
}

func TestBuildStaffCreatedPayload_JSONRoundTrip(t *testing.T) {
	payload := buildStaffCreatedPayload(dto.CreateStaffRequest{
		Email:     "x@x.x",
		FirstName: "X",
		LastName:  "X",
		Role:      "admin",
	})

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "x@x.x", decoded["email"])
	assert.Equal(t, "admin", decoded["role"])
}

func TestBuildStaffUpdatedPayload_ContractKeys(t *testing.T) {
	dept := "CS"
	payload := buildStaffUpdatedPayload(StaffUpdatedInputs{
		ID:         "11111111-1111-1111-1111-111111111111",
		Department: &dept,
	})

	required := []string{"staff_id", "department", "phone", "office_location"}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	// `staff_id` (not `id`) — the rename happened during a refactor; auth uses
	// staff_id to match the worker's manual mapping. Locking the name down.
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", payload["staff_id"])
}

func TestBuildStaffUpdatedPayload_NilOptionalsArePreserved(t *testing.T) {
	// auth-service uses pointer presence to distinguish "not changed" from
	// "explicitly cleared". If the helper drops nil keys silently, the
	// distinction is lost.
	payload := buildStaffUpdatedPayload(StaffUpdatedInputs{
		ID: "id",
	})

	assert.Contains(t, payload, "department", "department key must exist even when nil")
	assert.Nil(t, payload["department"])
	assert.Contains(t, payload, "phone")
	assert.Nil(t, payload["phone"])
	assert.Contains(t, payload, "office_location")
	assert.Nil(t, payload["office_location"])
}

func TestBuildStaffDeactivatedPayload_ContractKeys(t *testing.T) {
	payload := buildStaffDeactivatedPayload("11111111-1111-1111-1111-111111111111")

	require.Contains(t, payload, "staff_id")
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", payload["staff_id"])
	assert.Len(t, payload, 1, "deactivation is a single-key event — adding fields here without coordinating with auth is a bug")
}
