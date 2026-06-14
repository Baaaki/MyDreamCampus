package worker

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// enrollment-service consumes course.semester.created and grade.* events.
// Both use the FLAT envelope (no nested "data" — see TEST_STATUS.md
// 4.7 for why). This test pins the consumer-side parse, complementing
// the producer-side test in
//   course-catalog-service/internal/handler/event_contract_test.go.

func TestCourseSemesterCreatedEvent_FlatEnvelopeParses(t *testing.T) {
	courseID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	body := mustMarshal(t, map[string]any{
		"event_id":            uuid.New().String(),
		"event_type":          "course.semester.created",
		"timestamp":           "2026-04-28T10:00:00Z",
		"semester_course_id":  courseID.String(),
		"semester":            "2025-2026-Fall",
		"course_code":         "CS101",
		"course_name":         "Intro to CS",
		"faculty":             "Engineering",
		"department":          "Computer Science",
		"credits":             6,
		"class_level":         1,
		"course_type":         "compulsory",
		"instructor_id":       uuid.New().String(),
		"instructor_fullname": "Jane Doe",
		"max_capacity":        50,
	})

	var event dto.CourseSemesterCreatedEvent
	require.NoError(t, json.Unmarshal(body, &event))

	// `semester_course_id` (NOT `course_id`) is the contracted name —
	// catalog renamed this field during a refactor and consumers had to
	// follow. Pinning it here so a future rename can't slip past CI.
	assert.Equal(t, courseID, event.SemesterCourseID,
		"field name is semester_course_id — see CLAUDE.md memory note about the rename")
	assert.Equal(t, "CS101", event.CourseCode)

	// Same situation for instructor_fullname (was instructor_name).
	assert.Equal(t, "Jane Doe", event.InstructorFullname,
		"field name is instructor_fullname — see CLAUDE.md memory note")
}

func TestGradeStudentPrerequisitePassedEvent_FlatEnvelopeParses(t *testing.T) {
	// TEST_STATUS.md 2.4.A: producer (grades-service) currently wraps
	// this event under a `data` field but enrollment's struct expects
	// flat top-level fields. The worker compensates with manual mapping.
	// This test pins the FLAT shape that the struct expects so the day
	// the producer is fixed, the change is intentional rather than
	// silent.
	studentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	body := mustMarshal(t, map[string]any{
		"event_id":    uuid.New().String(),
		"event_type":  "grade.student.prerequisite.passed",
		"timestamp":   "2026-04-28T10:00:00Z",
		"student_id":  studentID.String(),
		"course_code": "CS101",
		"semester":    "2025-2026-Fall",
		"grade_point": "AA",
	})

	var event dto.GradeStudentPrerequisitePassedEvent
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, studentID, event.StudentID)
	assert.Equal(t, "CS101", event.CourseCode)
	assert.Equal(t, "AA", event.GradePoint,
		"grade_point is the routing key for prereq-passed checks downstream")
}

func TestCourseSemesterCreatedEvent_BadJSONFailsLoudly(t *testing.T) {
	var event dto.CourseSemesterCreatedEvent
	err := json.Unmarshal([]byte("garbage"), &event)
	require.Error(t, err,
		"silent acceptance of bad JSON would acknowledge the message and lose the catalog update")
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
