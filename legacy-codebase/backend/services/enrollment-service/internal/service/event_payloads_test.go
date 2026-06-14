package service

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the wire contract for the enrollment.* outbox events.
// Consumers (grades, attendance, auth) read these keys directly. A silent
// rename here is a production incident — JSON marshal won't catch it.

func TestBuildEnrollmentApprovedPayload_WrappedShape(t *testing.T) {
	programID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	studentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	advisorID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	courseIDs := []uuid.UUID{
		uuid.MustParse("44444444-4444-4444-4444-444444444444"),
	}

	payload := buildEnrollmentApprovedPayload(EnrollmentApprovedInputs{
		ProgramID:  programID,
		StudentID:  studentID,
		Semester:   "2025-2026-Fall",
		CourseIDs:  courseIDs,
		ApprovedBy: advisorID,
	})

	// approved is the WRAPPED contract — top-level event_id/event_type/timestamp/data.
	// rejected/cancelled/submitted use the FLAT contract (worker wraps later).
	// This asymmetry is the contract — see TestBuildEnrollmentRejectedPayload_FlatShape.
	for _, k := range []string{"event_id", "event_type", "timestamp", "data"} {
		assert.Contains(t, payload, k, "approved is wrapped — key %q must exist at the top level", k)
	}

	assert.Equal(t, events.EventEnrollmentProgramApproved, payload["event_type"])
	assert.Equal(t, "enrollment.program.approved", payload["event_type"],
		"the canonical name is the wire format — the constant exists to prevent typos")

	eventID, ok := payload["event_id"].(string)
	require.True(t, ok, "event_id must be a string")
	_, err := uuid.Parse(eventID)
	require.NoError(t, err, "event_id must be a parseable UUID")

	data, ok := payload["data"].(map[string]any)
	require.True(t, ok, "data must be a nested map")

	for _, k := range []string{"program_id", "student_id", "semester", "course_ids", "approved_by", "approved_at"} {
		assert.Contains(t, data, k, "data.%s must exist", k)
	}

	assert.Equal(t, programID.String(), data["program_id"])
	assert.Equal(t, studentID.String(), data["student_id"])
	assert.Equal(t, advisorID.String(), data["approved_by"])
	assert.Equal(t, "2025-2026-Fall", data["semester"])
	assert.Equal(t, courseIDs, data["course_ids"], "course_ids stays []uuid.UUID — JSON-encodes as string array")
}

func TestBuildEnrollmentApprovedPayload_JSONRoundTrip(t *testing.T) {
	payload := buildEnrollmentApprovedPayload(EnrollmentApprovedInputs{
		ProgramID:  uuid.New(),
		StudentID:  uuid.New(),
		Semester:   "2025-2026-Fall",
		CourseIDs:  []uuid.UUID{uuid.New()},
		ApprovedBy: uuid.New(),
	})

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "enrollment.program.approved", decoded["event_type"])

	data, ok := decoded["data"].(map[string]any)
	require.True(t, ok)
	courseIDs, ok := data["course_ids"].([]any)
	require.True(t, ok, "course_ids must JSON-decode as []any (string array), not base64")
	require.Len(t, courseIDs, 1)
}

func TestBuildEnrollmentRejectedPayload_FlatShape(t *testing.T) {
	programID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	studentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	advisorID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	courseIDs := []uuid.UUID{uuid.New()}

	payload := buildEnrollmentRejectedPayload(EnrollmentRejectedInputs{
		ProgramID:       programID,
		StudentID:       studentID,
		Semester:        "2025-2026-Fall",
		CourseIDs:       courseIDs,
		RejectedBy:      advisorID,
		RejectionReason: "ders catismasi",
	})

	// rejected is FLAT — worker wraps later.
	assert.NotContains(t, payload, "event_id",
		"rejected payload must be flat — wrapping is done by the outbox worker")
	assert.NotContains(t, payload, "data",
		"rejected payload must be flat — wrapping by the consumer reading worker bypasses this layer")

	required := []string{
		"program_id", "student_id", "semester", "course_ids",
		"rejected_by", "rejection_reason", "rejected_at",
	}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Equal(t, programID.String(), payload["program_id"])
	assert.Equal(t, studentID.String(), payload["student_id"])
	assert.Equal(t, advisorID.String(), payload["rejected_by"])
	assert.Equal(t, "ders catismasi", payload["rejection_reason"])
}

func TestBuildEnrollmentCancelledPayload_FlatShape(t *testing.T) {
	programID := uuid.New()
	studentID := uuid.New()
	courseIDs := []uuid.UUID{uuid.New(), uuid.New()}

	payload := buildEnrollmentCancelledPayload(EnrollmentCancelledInputs{
		ProgramID:   programID,
		StudentID:   studentID,
		Semester:    "2025-2026-Fall",
		CourseIDs:   courseIDs,
		CancelledBy: "student",
		CancelType:  "manual",
	})

	required := []string{
		"program_id", "student_id", "semester", "course_ids",
		"cancelled_by", "cancel_type", "cancelled_at",
	}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Equal(t, "student", payload["cancelled_by"])
	assert.Equal(t, "manual", payload["cancel_type"])
}

func TestBuildEnrollmentCancelledPayload_AutoReplaceVariant(t *testing.T) {
	// auto_replace fires when student resubmits — keys are identical to manual,
	// only cancel_type differs. If this divergence is ever forgotten downstream
	// the consumer can't distinguish the two.
	payload := buildEnrollmentCancelledPayload(EnrollmentCancelledInputs{
		ProgramID:   uuid.New(),
		StudentID:   uuid.New(),
		Semester:    "2025-2026-Fall",
		CourseIDs:   []uuid.UUID{uuid.New()},
		CancelledBy: "student",
		CancelType:  "auto_replace",
	})

	assert.Equal(t, "auto_replace", payload["cancel_type"],
		"auto_replace is what tells consumers this is a re-submit, not a manual cancel")
}

func TestBuildEnrollmentSubmittedPayload_FlatShape(t *testing.T) {
	studentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	courseIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	payload := buildEnrollmentSubmittedPayload(EnrollmentSubmittedInputs{
		StudentID: studentID,
		Semester:  "2025-2026-Fall",
		CourseIDs: courseIDs,
	})

	required := []string{
		"program_id", "student_id", "semester", "course_ids",
		"total_courses", "submitted_at",
	}
	for _, k := range required {
		assert.Contains(t, payload, k, "wire contract: key %q must not be removed", k)
	}

	assert.Nil(t, payload["program_id"], "program_id is filled by the repository after insert")
	assert.Equal(t, studentID.String(), payload["student_id"])
	assert.Equal(t, "2025-2026-Fall", payload["semester"])
	assert.Equal(t, 3, payload["total_courses"], "total_courses must equal len(course_ids) — defensive assertion")
}

func TestBuildEnrollmentRejectedPayload_JSONRoundTrip(t *testing.T) {
	payload := buildEnrollmentRejectedPayload(EnrollmentRejectedInputs{
		ProgramID:       uuid.New(),
		StudentID:       uuid.New(),
		Semester:        "2025-2026-Fall",
		CourseIDs:       []uuid.UUID{uuid.New()},
		RejectedBy:      uuid.New(),
		RejectionReason: "nedensiz",
	})

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "nedensiz", decoded["rejection_reason"])
}
