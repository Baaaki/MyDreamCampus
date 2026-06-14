package dto

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file is the consumer-side half of the contract test pair for the
// course.semester.created event. The producer-side half lives at
//   backend/services/course-catalog-service/internal/handler/event_contract_test.go
// and pins the shape catalog actually emits to RabbitMQ.
//
// CLAUDE.md memory records that this exact contract has slipped before:
// catalog renamed course_id → semester_course_id and instructor_name →
// instructor_fullname, and attendance silently consumed zeros until the bug
// surfaced in production. The literal below is the documented FLAT shape;
// renaming any tag on CourseSemesterCreatedEventData breaks this test
// immediately, while renaming a field on the producer breaks the matching
// catalog-side test. Either side drifting fails fast.

const courseSemesterCreatedFixtureFlat = `{
  "event_id": "00000000-0000-0000-0000-0000000000aa",
  "event_type": "course.semester.created",
  "timestamp": "2026-04-27T09:00:00Z",
  "semester_course_id": "11111111-1111-1111-1111-111111111111",
  "semester": "2025-2026-Fall",
  "course_code": "CS101",
  "course_name": "Intro to CS",
  "faculty": "Engineering",
  "department": "Computer Science",
  "credits": 6,
  "class_level": 1,
  "course_type": "compulsory",
  "instructor_id": "22222222-2222-2222-2222-222222222222",
  "instructor_fullname": "Jane Doe",
  "classroom_location": "B-204",
  "max_capacity": 50,
  "assessment_schema": [{"slug":"midterm","name":"Midterm","weight":40}],
  "prerequisites": [],
  "schedule_sessions": [
    {"day_of_week":"monday","slot_numbers":[1,2,3],"session_type":"theory"},
    {"day_of_week":"wednesday","slot_numbers":[4,5],"session_type":"lab"}
  ]
}`

func TestCourseSemesterCreatedEventData_UnmarshalsFlatPayload(t *testing.T) {
	var got CourseSemesterCreatedEventData
	require.NoError(t, json.Unmarshal([]byte(courseSemesterCreatedFixtureFlat), &got))

	t.Run("renamed-once fields are bound to the post-rename names", func(t *testing.T) {
		// The historical bug — catalog renamed these and attendance was
		// reading the old names — is locked here. If anyone reverts a tag
		// these assertions break.
		assert.Equal(t,
			uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			got.SemesterCourseID,
			"semester_course_id (was course_id) must populate SemesterCourseID")

		assert.Equal(t, "Jane Doe", got.InstructorFullname,
			"instructor_fullname (was instructor_name) must populate InstructorFullname")
	})

	t.Run("scalar fields land verbatim", func(t *testing.T) {
		assert.Equal(t, "CS101", got.CourseCode)
		assert.Equal(t, "Intro to CS", got.CourseName)
		assert.Equal(t, int16(6), got.Credits)
		assert.Equal(t, "2025-2026-Fall", got.Semester)
		assert.Equal(t, "Computer Science", got.Department)
		assert.Equal(t,
			uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			got.InstructorID,
		)
	})

	t.Run("schedule_sessions decode preserves session_type values", func(t *testing.T) {
		// The consumer reads only session_type from the schedule sessions —
		// it scans for "lab" to set hasLab on the cache row. A change that
		// renames session_type or drops the field would silently make every
		// course look theory-only.
		require.Len(t, got.ScheduleSessions, 2)
		types := []string{got.ScheduleSessions[0].SessionType, got.ScheduleSessions[1].SessionType}
		assert.ElementsMatch(t, []string{"theory", "lab"}, types)
	})
}

func TestCourseSemesterCreatedEventData_FlatPayloadIsNotWrapped(t *testing.T) {
	// A regression class to guard against: somebody normalises all events to
	// wrap under "data" (matching the BaseEvent pattern that student events
	// use). If that happens, json.Unmarshal(body, &eventData) — which is what
	// handleCourseSemesterCreated does — would silently produce zero values.
	wrapped := `{
      "event_id": "00000000-0000-0000-0000-0000000000aa",
      "event_type": "course.semester.created",
      "data": ` + courseSemesterCreatedFixtureFlat + `
    }`

	var got CourseSemesterCreatedEventData
	require.NoError(t, json.Unmarshal([]byte(wrapped), &got))

	// All consumer-relevant fields stay zero because the data is one level
	// down. This test pins the FLAT shape contract: if anyone changes the
	// publish path to wrap, this test must be updated together with the
	// consumer's unmarshal logic.
	assert.Equal(t, uuid.Nil, got.SemesterCourseID)
	assert.Empty(t, got.CourseCode)
	assert.Empty(t, got.InstructorFullname)
}

func TestCourseSemesterCreatedEventData_HandlesMissingOptionalFields(t *testing.T) {
	// Minimal valid payload — the consumer must not panic on a payload that
	// omits collection fields. Empty slices and zero scalars are acceptable.
	minimal := `{
      "semester_course_id": "11111111-1111-1111-1111-111111111111",
      "course_code": "CS101",
      "course_name": "Intro",
      "credits": 6,
      "semester": "2025-2026-Fall",
      "department": "CS",
      "instructor_id": "22222222-2222-2222-2222-222222222222",
      "instructor_fullname": "Jane"
    }`

	var got CourseSemesterCreatedEventData
	require.NoError(t, json.Unmarshal([]byte(minimal), &got))

	assert.Empty(t, got.ScheduleSessions, "missing schedule_sessions must decode to empty, not error")
	assert.Equal(t, "CS101", got.CourseCode)
}

// The student.* events are wrapped under "data" — opposite shape. Pinning
// the wrapped contract here so a future cleanup that flattens these doesn't
// silently break the existing consumer paths in attendance-service.

func TestStudentCreatedEventData_RequiresWrappedShape(t *testing.T) {
	wrapped := `{
      "event_id": "00000000-0000-0000-0000-0000000000bb",
      "event_type": "student.created",
      "data": {
        "id": "33333333-3333-3333-3333-333333333333",
        "student_number": "20210001",
        "first_name": "Ahmet",
        "last_name": "Yilmaz",
        "email": "ahmet@univ.edu",
        "department": "Computer Science"
      }
    }`

	// The consumer does a two-step unwrap: BaseEvent first, then re-marshal
	// the data field and decode it into the typed struct. Mirror that path
	// here so the test covers the production logic, not just a happy struct.
	var base BaseEvent
	require.NoError(t, json.Unmarshal([]byte(wrapped), &base))

	dataBytes, err := json.Marshal(base.Data)
	require.NoError(t, err)

	var got StudentCreatedEventData
	require.NoError(t, json.Unmarshal(dataBytes, &got))

	assert.Equal(t,
		uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		got.StudentID,
		"json tag is 'id', not 'student_id' — historical contract from student-service")
	assert.Equal(t, "20210001", got.StudentNumber)
	assert.Equal(t, "Computer Science", got.Department)
}

// =============================================================================
// OUTBOUND — published by attendance-service
// =============================================================================

func TestAttendanceSemesterFailedEvent_PublishedShape(t *testing.T) {
	// publishFailedAttendanceEvent (attendance_service.go:777) wraps the data
	// in BaseEvent before publishing. Pin the resulting wire shape so a
	// future unwrap-flatten refactor breaks this test loudly. grades-service
	// consumes this event (grades-service/.../dto/event_dto.go) and reads
	// "data.{student_id, course_id, semester, failed_at}" — those tag names
	// must be preserved.

	studentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	courseID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	data := AttendanceSemesterFailedEventData{
		StudentID:     studentID,
		StudentNumber: "20210001",
		StudentEmail:  "ahmet@univ.edu",
		CourseID:      courseID,
		CourseCode:    "CS101",
		CourseName:    "Intro to CS",
		Semester:      "2025-2026-Fall",
		TotalWeeks:    14,
		FailedType:    "theory",
		Theory: &AttendanceFailedTypeDetail{
			TotalSessions: 14,
			PresentCount:  4,
			AbsentCount:   10,
			MinRequired:   10,
		},
	}

	envelope := BaseEvent{
		EventID:   uuid.New(),
		EventType: "attendance.semester.failed",
		Data:      data,
	}

	raw, err := json.Marshal(envelope)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	// Wrapper-level shape.
	assert.Contains(t, decoded, "event_id")
	assert.Equal(t, "attendance.semester.failed", decoded["event_type"])
	require.Contains(t, decoded, "data")

	// Data-level shape — every field a downstream might bind to.
	dataMap := decoded["data"].(map[string]any)
	for _, k := range []string{
		"student_id", "student_number", "student_email",
		"course_id", "course_code", "course_name",
		"semester", "total_weeks", "failed_type",
	} {
		assert.Contains(t, dataMap, k, "data must include %q", k)
	}
	assert.Equal(t, studentID.String(), dataMap["student_id"])
	assert.Equal(t, "theory", dataMap["failed_type"])

	// Pointer-typed Theory/Lab serialise as nested objects, not as null when
	// populated. Lab is nil here so it must be omitted (omitempty).
	require.Contains(t, dataMap, "theory")
	theory := dataMap["theory"].(map[string]any)
	assert.Contains(t, theory, "total_sessions")
	assert.Contains(t, theory, "min_required")

	assert.NotContains(t, dataMap, "lab", "nil Lab should be omitted via omitempty")
}

func TestAttendanceFailedTypeDetail_AllFieldsRequired(t *testing.T) {
	// The detail object describes which threshold was missed. Renaming any of
	// these zeroes the consumer's downstream calculations silently — keep
	// them all pinned.
	d := AttendanceFailedTypeDetail{
		TotalSessions: 14,
		PresentCount:  4,
		AbsentCount:   10,
		MinRequired:   10,
	}
	raw, err := json.Marshal(d)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	for _, k := range []string{"total_sessions", "present_count", "absent_count", "min_required"} {
		assert.Contains(t, decoded, k)
	}
}

func TestEnrollmentProgramApprovedEventData_DecodesWrapped(t *testing.T) {
	wrapped := `{
      "event_id": "00000000-0000-0000-0000-0000000000cc",
      "event_type": "enrollment.program.approved",
      "data": {
        "program_id": "44444444-4444-4444-4444-444444444444",
        "student_id": "55555555-5555-5555-5555-555555555555",
        "semester": "2025-2026-Fall",
        "course_ids": [
          "66666666-6666-6666-6666-666666666666",
          "77777777-7777-7777-7777-777777777777"
        ],
        "approved_by": "88888888-8888-8888-8888-888888888888"
      }
    }`

	var base BaseEvent
	require.NoError(t, json.Unmarshal([]byte(wrapped), &base))
	dataBytes, err := json.Marshal(base.Data)
	require.NoError(t, err)

	var got EnrollmentProgramApprovedEventData
	require.NoError(t, json.Unmarshal(dataBytes, &got))

	assert.Equal(t,
		uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		got.ProgramID)
	assert.Equal(t, "2025-2026-Fall", got.Semester)
	require.Len(t, got.CourseIDs, 2)
	assert.Equal(t,
		uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		got.CourseIDs[0])
}
