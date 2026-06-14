package worker

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/grades-service/internal/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// grades-service consumes course.semester.created (FLAT) and
// enrollment.program.approved (WRAPPED). Each shape has its own pin.

func TestCourseSemesterCreatedEvent_FlatEnvelope(t *testing.T) {
	courseID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	body := mustMarshal(t, map[string]any{
		"event_id":            uuid.New().String(),
		"event_type":          "course.semester.created",
		"timestamp":           "2026-04-28T10:00:00Z",
		"semester_course_id":  courseID.String(),
		"semester":            "2025-2026-Fall",
		"course_code":         "CS101",
		"course_name":         "Intro",
		"credits":             6,
	})

	var event dto.CourseSemesterCreatedEvent
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, courseID, event.SemesterCourseID,
		"semester_course_id is the contracted name — renaming silently breaks grades-service's course cache hydration")
	assert.Equal(t, "CS101", event.CourseCode)
}

func TestEnrollmentProgramApprovedEvent_WrappedEnvelope(t *testing.T) {
	// enrollment.program.approved uses the WRAPPED contract — top-level
	// event_id/event_type/timestamp + a nested `data` field. This is the
	// asymmetry we lock in via TestBuildEnrollmentApprovedPayload_WrappedShape
	// on the producer side; this test is the consumer's mirror.
	studentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	courseID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	body := mustMarshal(t, map[string]any{
		"event_id":   "11111111-1111-1111-1111-111111111111",
		"event_type": "enrollment.program.approved",
		"timestamp":  "2026-04-28T10:00:00Z",
		"data": map[string]any{
			"student_id":  studentID.String(),
			"semester":    "2025-2026-Fall",
			"course_ids":  []string{courseID.String()},
			"approved_at": "2026-04-28T10:00:00Z",
		},
	})

	var event dto.EnrollmentProgramApprovedEvent
	require.NoError(t, json.Unmarshal(body, &event))

	assert.Equal(t, studentID, event.Data.StudentID)
	assert.Equal(t, "2025-2026-Fall", event.Data.Semester)
	require.Len(t, event.Data.CourseIDs, 1)
	assert.Equal(t, courseID, event.Data.CourseIDs[0],
		"course_ids must round-trip from JSON string array — grades uses them to enumerate which courses to register a registration row for")
}

func TestEnrollmentProgramApprovedEvent_MissingDataIsZero(t *testing.T) {
	// Defensive: producer that drops the `data` field. Should NOT panic;
	// should leave Data with zero values so the handler can detect it.
	body := mustMarshal(t, map[string]any{
		"event_id":   "11111111-1111-1111-1111-111111111111",
		"event_type": "enrollment.program.approved",
		"timestamp":  "2026-04-28T10:00:00Z",
	})

	var event dto.EnrollmentProgramApprovedEvent
	require.NoError(t, json.Unmarshal(body, &event))
	assert.Empty(t, event.Data.CourseIDs,
		"missing data must yield empty course_ids — handler must treat empty as a programmer error rather than skip silently")
}

func TestEnrollmentProgramApprovedEvent_BadJSONFails(t *testing.T) {
	var event dto.EnrollmentProgramApprovedEvent
	err := json.Unmarshal([]byte("not json"), &event)
	require.Error(t, err)
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
