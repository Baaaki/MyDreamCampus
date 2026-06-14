package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file pins the wire shape of every event the grades-service touches.
// Each test marshals or unmarshals against a canonical JSON literal — when
// somebody renames a struct tag, the matching test breaks immediately and
// names the offending field. The pattern mirrors the contract pair set up
// for course.semester.created in catalog-service ↔ attendance-service.

// =============================================================================
// OUTBOUND — published by grades-service
// =============================================================================

func TestGradeSubmittedEvent_PublishedShape(t *testing.T) {
	studentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC)

	ev := GradeSubmittedEvent{
		EventType: "grade.submitted",
		Timestamp: ts,
	}
	ev.Data.StudentID = studentID
	ev.Data.CourseCode = "CS101"
	ev.Data.Slug = "midterm"
	ev.Data.Score = 87.5

	raw, err := json.Marshal(ev)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	assert.Equal(t, "grade.submitted", decoded["event_type"])
	require.Contains(t, decoded, "data", "grade.submitted is wrapped under data")

	data := decoded["data"].(map[string]any)
	assert.Equal(t, studentID.String(), data["student_id"])
	assert.Equal(t, "CS101", data["course_code"])
	assert.Equal(t, "midterm", data["slug"])
	assert.InDelta(t, 87.5, data["score"], 0.0001)
}

func TestGradeFinalizeRequestedEvent_PublishedShape(t *testing.T) {
	courseID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	instructorID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	ev := GradeFinalizeRequestedEvent{
		EventType: "grade.finalize.requested",
		Timestamp: time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
	}
	ev.Data.CourseID = courseID
	ev.Data.InstructorID = instructorID
	ev.Data.TriggeredBy = "instructor_lock_assessment"

	raw, err := json.Marshal(ev)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	data := decoded["data"].(map[string]any)
	assert.Equal(t, courseID.String(), data["course_id"])
	assert.Equal(t, instructorID.String(), data["instructor_id"])
	assert.Equal(t, "instructor_lock_assessment", data["triggered_by"])
}

func TestGradeFinalizedEvent_PublishedShape(t *testing.T) {
	courseID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	ev := GradeFinalizedEvent{
		EventType: "grade.finalized",
		Timestamp: time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
	}
	ev.Data.CourseID = courseID
	ev.Data.CourseCode = "CS101"
	ev.Data.Semester = "2025-2026-Fall"
	ev.Data.GradingType = "absolute"
	ev.Data.TotalStudents = 50
	ev.Data.PassingCount = 42
	ev.Data.FailingCount = 6
	ev.Data.AttendanceFailedCount = 2
	ev.Data.ClassMean = 73.5

	raw, err := json.Marshal(ev)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	data := decoded["data"].(map[string]any)

	// Pin every field grading-summary consumers might read. Renaming any of
	// these silently zeroes them downstream.
	for _, k := range []string{
		"course_id", "course_code", "semester", "grading_type",
		"total_students", "passing_count", "failing_count",
		"attendance_failed_count", "class_mean",
	} {
		assert.Contains(t, data, k, "grade.finalized must include %q in data", k)
	}
	assert.Equal(t, "absolute", data["grading_type"])
	assert.InDelta(t, 73.5, data["class_mean"], 0.0001)
}

func TestGradeStudentPrerequisitePassedEvent_PublishedShape(t *testing.T) {
	// ⚠ Production note: enrollment-service's consumer DTO for this event
	// (enrollment-service/internal/dto/event_dto.go:GradeStudentPrerequisitePassedEvent)
	// declares the fields at TOP LEVEL via embedded BaseEvent — i.e. expects a
	// FLAT shape. But grades-service publishes them WRAPPED under "data".
	// In practice the worker layer in enrollment-service does manual
	// unmarshalling that bridges the gap (it doesn't json.Unmarshal directly
	// into the DTO struct), so production works. This test pins the producer
	// side of the contract; if the worker is ever simplified to a direct
	// json.Unmarshal(body, &GradeStudentPrerequisitePassedEvent), it will
	// silently zero every field. Documenting the trap here.
	studentID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	courseID := uuid.MustParse("66666666-6666-6666-6666-666666666666")

	ev := GradeStudentPrerequisitePassedEvent{
		EventType: "grade.student.prerequisite.passed",
		Timestamp: time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
	}
	ev.Data.StudentID = studentID
	ev.Data.CourseID = courseID
	ev.Data.CourseCode = "CS101"
	ev.Data.Semester = "2025-2026-Fall"
	ev.Data.GradePoint = "4.00"

	raw, err := json.Marshal(ev)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	require.Contains(t, decoded, "data",
		"grade.student.prerequisite.passed is wrapped under data — see comment above")
	data := decoded["data"].(map[string]any)
	assert.Equal(t, studentID.String(), data["student_id"])
	assert.Equal(t, courseID.String(), data["course_id"])
	assert.Equal(t, "CS101", data["course_code"])
	assert.Equal(t, "2025-2026-Fall", data["semester"])
	assert.Equal(t, "4.00", data["grade_point"])
}

// =============================================================================
// INBOUND — consumed by grades-service from other services
// =============================================================================

func TestStudentCreatedEvent_ConsumedShape(t *testing.T) {
	// Producer: student-service (outbox worker wraps under data).
	body := []byte(`{
      "event_id": "00000000-0000-0000-0000-0000000000aa",
      "event_type": "student.created",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "id": "11111111-1111-1111-1111-111111111111",
        "student_number": "20210001",
        "first_name": "Ahmet",
        "last_name": "Yilmaz",
        "email": "ahmet@univ.edu",
        "department": "Computer Science",
        "class_level": 2
      }
    }`)

	var ev StudentCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))

	// Pin both the wrapper fields and the data payload so a refactor
	// flattening the event is caught.
	assert.Equal(t, "student.created", ev.EventType)
	assert.Equal(t, "00000000-0000-0000-0000-0000000000aa", ev.EventID)
	assert.Equal(t,
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ev.Data.ID, "json tag is 'id', not 'student_id'")
	assert.Equal(t, "20210001", ev.Data.StudentNumber)
	assert.Equal(t, int16(2), ev.Data.ClassLevel)
	assert.Equal(t, "Computer Science", ev.Data.Department)
}

func TestStudentDeactivatedEvent_ConsumedShape(t *testing.T) {
	body := []byte(`{
      "event_id": "deact-1",
      "event_type": "student.deactivated",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {"id": "11111111-1111-1111-1111-111111111111"}
    }`)

	var ev StudentDeactivatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))
	assert.Equal(t,
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ev.Data.ID)
}

func TestCourseSemesterCreatedEvent_ConsumedFlatShape(t *testing.T) {
	// catalog-service publishes this FLAT (not wrapped under data) — see
	// the matching producer test at
	// course-catalog-service/internal/handler/event_contract_test.go.
	body := []byte(`{
      "event_id": "evt-1",
      "event_type": "course.semester.created",
      "timestamp": "2026-04-27T09:00:00Z",
      "semester_course_id": "11111111-1111-1111-1111-111111111111",
      "course_code": "CS101",
      "course_name": "Intro",
      "credits": 6,
      "semester": "2025-2026-Fall",
      "department": "CS",
      "instructor_id": "22222222-2222-2222-2222-222222222222",
      "instructor_fullname": "Jane Doe",
      "assessment_schema": [{"slug":"midterm","name":"Midterm","weight":40}]
    }`)

	var ev CourseSemesterCreatedEvent
	require.NoError(t, json.Unmarshal(body, &ev))

	assert.Equal(t,
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ev.SemesterCourseID,
		"semester_course_id was course_id pre-rename — keep the post-rename binding")
	assert.Equal(t, "Jane Doe", ev.InstructorFullname)
	require.Len(t, ev.AssessmentSchema, 1)
	assert.Equal(t, "midterm", ev.AssessmentSchema[0].Slug)
	assert.Equal(t, 40, ev.AssessmentSchema[0].Weight)
}

func TestEnrollmentProgramApprovedEvent_ConsumedShape(t *testing.T) {
	body := []byte(`{
      "event_id": "00000000-0000-0000-0000-0000000000ee",
      "event_type": "enrollment.program.approved",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "student_id": "11111111-1111-1111-1111-111111111111",
        "semester": "2025-2026-Fall",
        "course_ids": [
          "22222222-2222-2222-2222-222222222222",
          "33333333-3333-3333-3333-333333333333"
        ],
        "approved_at": "2026-04-27T10:00:00Z"
      }
    }`)

	var ev EnrollmentProgramApprovedEvent
	require.NoError(t, json.Unmarshal(body, &ev))

	assert.Equal(t, "2025-2026-Fall", ev.Data.Semester)
	require.Len(t, ev.Data.CourseIDs, 2)
	assert.Equal(t,
		uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ev.Data.CourseIDs[0])
	assert.False(t, ev.Data.ApprovedAt.IsZero())
}

func TestAttendanceSemesterFailedEvent_ConsumedShape(t *testing.T) {
	body := []byte(`{
      "event_id": "00000000-0000-0000-0000-0000000000af",
      "event_type": "attendance.semester.failed",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "student_id": "11111111-1111-1111-1111-111111111111",
        "course_id": "22222222-2222-2222-2222-222222222222",
        "semester": "2025-2026-Fall",
        "failed_at": "2026-04-27T10:00:00Z"
      }
    }`)

	var ev AttendanceSemesterFailedEvent
	require.NoError(t, json.Unmarshal(body, &ev))

	assert.Equal(t, "2025-2026-Fall", ev.Data.Semester)
	assert.False(t, ev.Data.FailedAt.IsZero())
}
