package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the wire contract for the student.* outbox events. Every
// downstream consumer (auth, attendance, enrollment, grades) reads these keys
// directly. A silent rename is a production incident — JSON marshal won't
// catch it; this test will.
//
// Companion: backend/services/auth-service/internal/dto/event_dto_test.go,
// course-catalog/internal/handler/event_contract_test.go.

func TestBuildStudentCreatedPayload_ContractKeys(t *testing.T) {
	req := dto.CreateStudentRequest{
		StudentNumber:  "20210001",
		FirstName:      "Ahmet",
		LastName:       "Yilmaz",
		Email:          "ahmet@univ.edu",
		Faculty:        "Engineering",
		Department:     "CS",
		EnrollmentYear: 2021,
		ClassLevel:     2,
	}

	payload := buildStudentCreatedPayload(req, nil)

	required := []string{
		"id", "student_number", "first_name", "last_name", "email",
		"faculty", "department", "enrollment_year", "class_level", "status",
	}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Nil(t, payload["id"], "id is filled by repository after insert")
	assert.Equal(t, "20210001", payload["student_number"])
	assert.Equal(t, "Ahmet", payload["first_name"])
	assert.Equal(t, "Yilmaz", payload["last_name"])
	assert.Equal(t, "ahmet@univ.edu", payload["email"])
	assert.Equal(t, "Engineering", payload["faculty"])
	assert.Equal(t, "CS", payload["department"])
	assert.Equal(t, 2021, payload["enrollment_year"])
	assert.Equal(t, int16(2), payload["class_level"])
	assert.Equal(t, "active", payload["status"])

	assert.NotContains(t, payload, "advisor_id",
		"advisor_id must be omitted when no advisor — auth treats present+nil differently from absent")
}

func TestBuildStudentCreatedPayload_WithAdvisor(t *testing.T) {
	advisorID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	req := dto.CreateStudentRequest{
		StudentNumber: "20210001",
		FirstName:     "Ahmet",
		LastName:      "Yilmaz",
		Email:         "ahmet@univ.edu",
	}

	payload := buildStudentCreatedPayload(req, &advisorID)

	require.Contains(t, payload, "advisor_id")
	assert.Equal(t, advisorID.String(), payload["advisor_id"])
}

func TestBuildStudentCreatedPayload_JSONRoundTrip(t *testing.T) {
	req := dto.CreateStudentRequest{
		StudentNumber:  "X",
		FirstName:      "X",
		LastName:       "X",
		Email:          "X",
		Faculty:        "X",
		Department:     "X",
		EnrollmentYear: 2021,
		ClassLevel:     1,
	}

	payload := buildStudentCreatedPayload(req, nil)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "X", decoded["student_number"])
	assert.Equal(t, "active", decoded["status"])
}

func TestBuildStudentUpdatedPayload_ContractKeys(t *testing.T) {
	advisorID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	current := db.Student{
		StudentNumber:  "20210001",
		FirstName:      "Ahmet",
		LastName:       "Yilmaz",
		Email:          "ahmet@univ.edu",
		Faculty:        "Engineering",
		Department:     "CS",
		EnrollmentYear: 2021,
		AdvisorID:      pgtype.UUID{Bytes: advisorID, Valid: true},
		Status:         pgtype.Text{String: "active", Valid: true},
	}

	payload := buildStudentUpdatedPayload(StudentUpdatedInputs{
		ID:            "11111111-1111-1111-1111-111111111111",
		Current:       current,
		FirstName:     "AhmetUpdated",
		LastName:      "Yilmaz",
		Email:         "ahmet@univ.edu",
		ClassLevel:    3,
		ChangedFields: map[string]any{"first_name": "AhmetUpdated", "class_level": int16(3)},
	})

	required := []string{
		"id", "student_number", "first_name", "last_name", "email",
		"faculty", "department", "enrollment_year", "class_level",
		"advisor_id", "status", "changed_fields",
	}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Equal(t, "11111111-1111-1111-1111-111111111111", payload["id"])
	assert.Equal(t, "AhmetUpdated", payload["first_name"])
	assert.Equal(t, int16(3), payload["class_level"])
	assert.Equal(t, advisorID.String(), payload["advisor_id"])
	assert.Equal(t, "active", payload["status"])
	assert.Equal(t, 2021, payload["enrollment_year"], "enrollment_year is int (not int32) — auth-service casts as int")

	cf, ok := payload["changed_fields"].(map[string]any)
	require.True(t, ok, "changed_fields must be a map")
	assert.Equal(t, "AhmetUpdated", cf["first_name"])
}

func TestBuildStudentUpdatedPayload_StatusOverride(t *testing.T) {
	current := db.Student{
		StudentNumber: "20210001",
		Status:        pgtype.Text{String: "active", Valid: true},
	}

	deactivated := "deactivated"
	payload := buildStudentUpdatedPayload(StudentUpdatedInputs{
		ID:             "id",
		Current:        current,
		ChangedFields:  map[string]any{"status": deactivated},
		StatusOverride: &deactivated,
	})

	assert.Equal(t, "deactivated", payload["status"],
		"StatusOverride must overwrite the current status — without override, auth would sync stale state")
}

func TestBuildStudentUpdatedPayload_AdvisorOverride(t *testing.T) {
	currentAdvisor := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	newAdvisor := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	current := db.Student{
		AdvisorID: pgtype.UUID{Bytes: currentAdvisor, Valid: true},
		Status:    pgtype.Text{String: "active", Valid: true},
	}

	payload := buildStudentUpdatedPayload(StudentUpdatedInputs{
		ID:              "id",
		Current:         current,
		ChangedFields:   map[string]any{"advisor_id": newAdvisor.String()},
		AdvisorOverride: &newAdvisor,
	})

	assert.Equal(t, newAdvisor.String(), payload["advisor_id"],
		"AdvisorOverride must overwrite the current advisor_id")
}

func TestBuildStudentDeactivatedPayload_ContractKeys(t *testing.T) {
	payload := buildStudentDeactivatedPayload("11111111-1111-1111-1111-111111111111", "20210001")

	required := []string{"id", "student_number", "is_active", "deleted_at"}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Equal(t, "11111111-1111-1111-1111-111111111111", payload["id"])
	assert.Equal(t, "20210001", payload["student_number"])
	assert.Equal(t, false, payload["is_active"], "is_active must be the literal false — auth uses it as the deactivation flag")

	deletedAt, ok := payload["deleted_at"].(string)
	require.True(t, ok, "deleted_at must be a string (RFC3339)")
	require.NotEmpty(t, deletedAt)
}

func TestBuildStudentDeactivatedPayload_DeletedAtIsRFC3339(t *testing.T) {
	payload := buildStudentDeactivatedPayload("id", "20210001")

	deletedAt, ok := payload["deleted_at"].(string)
	require.True(t, ok)

	// Consumers parse this with time.Parse(time.RFC3339). If the format drifts,
	// every downstream silently sets deleted_at to zero time.
	_, err := time.Parse(time.RFC3339, deletedAt)
	require.NoError(t, err, "deleted_at must be parseable as RFC3339")
}
