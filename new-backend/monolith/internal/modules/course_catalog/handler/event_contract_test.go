package handler

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/db"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file pins the wire contract for the course.semester.created event that
// catalog publishes to RabbitMQ. The payload is built as a literal map[string]any
// in buildCourseSemesterCreatedPayload (semester_status_handler.go) — the struct
// SemesterCourseCreatedEvent in dto/event_dto.go is documentation, not the
// source of truth. Renaming a key in that map silently breaks every consumer
// (attendance-service, enrollment-service); this test catches such renames.
//
// The matching consumer-side contract test lives at
//   backend/services/attendance-service/internal/dto/event_contract_test.go

// makeRow builds a fully populated row that buildCourseSemesterCreatedPayload
// can consume. Field values are arbitrary but distinct so the test can verify
// each one survives the trip through the payload builder.
func makeRow(t *testing.T) (db.ListSemesterCoursesForActivationRow, uuid.UUID, uuid.UUID) {
	t.Helper()

	courseID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	instructorID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	row := db.ListSemesterCoursesForActivationRow{
		ID:                 pgtype.UUID{Bytes: courseID, Valid: true},
		Semester:           "2025-2026-Fall",
		CourseCode:         "CS101",
		CourseName:         "Intro to CS",
		Faculty:            "Engineering",
		Department:         "Computer Science",
		Credits:            6,
		ClassLevel:         1,
		CourseType:         db.CourseTypeEnum("compulsory"),
		InstructorID:       pgtype.UUID{Bytes: instructorID, Valid: true},
		InstructorFullname: "Jane Doe",
		ClassroomLocation:  "B-204",
		MaxCapacity:        50,
		AssessmentSchema:   []byte(`[{"slug":"midterm","name":"Midterm","weight":40},{"slug":"final","name":"Final","weight":60}]`),
		Prerequisites:      []byte(`[]`),
		ScheduleSessions:   []byte(`[{"day_of_week":"monday","slot_numbers":[1,2,3],"session_type":"theory"}]`),
	}
	return row, courseID, instructorID
}

func TestBuildCourseSemesterCreatedPayload_TopLevelFlatShape(t *testing.T) {
	row, _, _ := makeRow(t)

	payload := buildCourseSemesterCreatedPayload(row)

	// The payload MUST be a flat map. Consumers (attendance-service,
	// enrollment-service) call json.Unmarshal(body, &eventData) directly
	// against the body — there is no "data" wrapper in this contract.
	assert.NotContains(t, payload, "data",
		"course.semester.created is intentionally flat — wrapping under data would break attendance-service consumer")

	// Every contracted top-level key must be present. This list is the
	// load-bearing assertion: drop a key here and you've broken a downstream.
	required := []string{
		"event_id",
		"event_type",
		"timestamp",
		"semester_course_id",
		"semester",
		"course_code",
		"course_name",
		"faculty",
		"department",
		"credits",
		"class_level",
		"course_type",
		"instructor_id",
		"instructor_fullname",
		"classroom_location",
		"max_capacity",
		"assessment_schema",
		"prerequisites",
		"schedule_sessions",
	}
	for _, k := range required {
		assert.Contains(t, payload, k, "missing required event key %q", k)
	}
}

func TestBuildCourseSemesterCreatedPayload_FieldValues(t *testing.T) {
	row, courseID, instructorID := makeRow(t)
	payload := buildCourseSemesterCreatedPayload(row)

	t.Run("event_type uses the canonical constant", func(t *testing.T) {
		assert.Equal(t, events.EventCourseSemesterCreated, payload["event_type"])
		assert.Equal(t, "course.semester.created", payload["event_type"])
	})

	t.Run("event_id is a parseable UUID string", func(t *testing.T) {
		raw, ok := payload["event_id"].(string)
		require.True(t, ok, "event_id must be a string")
		_, err := uuid.Parse(raw)
		assert.NoError(t, err)
	})

	t.Run("UUIDs are stringified, not pgtype objects", func(t *testing.T) {
		// Consumers Parse these as strings — never as pgtype.UUID. Catching a
		// silent change to raw struct values that would JSON-encode as objects.
		assert.Equal(t, courseID.String(), payload["semester_course_id"])
		assert.Equal(t, instructorID.String(), payload["instructor_id"])
	})

	t.Run("scalar fields land verbatim", func(t *testing.T) {
		assert.Equal(t, "2025-2026-Fall", payload["semester"])
		assert.Equal(t, "CS101", payload["course_code"])
		assert.Equal(t, "Intro to CS", payload["course_name"])
		assert.Equal(t, "Engineering", payload["faculty"])
		assert.Equal(t, "Computer Science", payload["department"])
		assert.Equal(t, int16(6), payload["credits"])
		assert.Equal(t, int16(1), payload["class_level"])
		assert.Equal(t, "compulsory", payload["course_type"])
		assert.Equal(t, "Jane Doe", payload["instructor_fullname"])
		assert.Equal(t, "B-204", payload["classroom_location"])
		assert.Equal(t, int16(50), payload["max_capacity"])
	})

	t.Run("nested JSONB fields are decoded into typed slices, not raw bytes", func(t *testing.T) {
		// AssessmentSchema, Prerequisites, ScheduleSessions arrive as []byte
		// from sqlc and must be unmarshalled — not embedded as base64.
		schema, ok := payload["assessment_schema"].([]map[string]any)
		require.True(t, ok, "assessment_schema must be a slice of maps; got %T", payload["assessment_schema"])
		require.Len(t, schema, 2)
		assert.Equal(t, "midterm", schema[0]["slug"])

		sessions, ok := payload["schedule_sessions"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, sessions, 1)
		assert.Equal(t, "theory", sessions[0]["session_type"])

		prereqs, ok := payload["prerequisites"].([]map[string]any)
		require.True(t, ok)
		assert.Empty(t, prereqs)
	})

	t.Run("payload survives JSON marshal round-trip", func(t *testing.T) {
		// The payload is published as JSON over RabbitMQ. Any value that fails
		// to marshal here is also unconsumable downstream. This is a smoke
		// check on top of the type assertions above.
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, "course.semester.created", decoded["event_type"])
		assert.Equal(t, "CS101", decoded["course_code"])
	})
}
