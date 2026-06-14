package dto

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Consumer-side contract for events enrollment-service consumes off RabbitMQ.
// The producer-side mirror lives at
//   backend/services/course-catalog-service/internal/handler/event_contract_test.go
// and the parallel attendance consumer at
//   backend/services/attendance-service/internal/dto/event_contract_test.go
//
// Why this file matters: course-catalog renamed course_id → semester_course_id
// and instructor_name → instructor_fullname in the past, and the consumers
// silently parsed zero values until production paged. The literals below pin
// the post-rename names — drift on either side fails this test.

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
  "prerequisites": [
    {"id":"33333333-3333-3333-3333-333333333333","course_code":"CS100","course_name":"Pre"}
  ],
  "schedule_sessions": [
    {"day_of_week":"monday","slot_numbers":[1,2,3],"session_type":"theory"},
    {"day_of_week":"wednesday","slot_numbers":[4,5],"session_type":"lab"}
  ]
}`

func TestCourseSemesterCreatedEvent_BindsRenamedFields(t *testing.T) {
	var got CourseSemesterCreatedEvent
	require.NoError(t, json.Unmarshal([]byte(courseSemesterCreatedFixtureFlat), &got))

	// Pin the historical bug: these fields were renamed once already and the
	// consumer silently zeroed. If anyone reverts a tag this fails.
	assert.Equal(t,
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		got.SemesterCourseID,
		"semester_course_id (was course_id) must populate SemesterCourseID")
	assert.Equal(t, "Jane Doe", got.InstructorFullname,
		"instructor_fullname (was instructor_name) must populate InstructorFullname")

	// Scalar trip-through.
	assert.Equal(t, "CS101", got.CourseCode)
	assert.Equal(t, "2025-2026-Fall", got.Semester)
	assert.Equal(t, "Computer Science", got.Department)
	assert.Equal(t, int16(6), got.Credits)
	assert.Equal(t, int16(1), got.ClassLevel)
	assert.Equal(t, "compulsory", got.CourseType)
	require.NotNil(t, got.InstructorID)
	assert.Equal(t,
		uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		*got.InstructorID)
	assert.Equal(t, int16(50), got.MaxCapacity)

	// Nested decoding — prerequisite check + schedule conflict logic both walk
	// these slices in the service layer.
	require.Len(t, got.Prerequisites, 1)
	assert.Equal(t, "CS100", got.Prerequisites[0].CourseCode)

	require.Len(t, got.ScheduleSessions, 2)
	assert.ElementsMatch(t,
		[]string{"theory", "lab"},
		[]string{got.ScheduleSessions[0].SessionType, got.ScheduleSessions[1].SessionType})
	assert.ElementsMatch(t,
		[]string{"monday", "wednesday"},
		[]string{got.ScheduleSessions[0].DayOfWeek, got.ScheduleSessions[1].DayOfWeek})
}

func TestCourseSemesterCreatedEvent_FlatPayloadIsNotWrapped(t *testing.T) {
	// Regression class: a normaliser that re-wraps every event under "data"
	// (matching the BaseEvent shape used for student events) would zero every
	// consumer-relevant field here. The consumer calls
	//   json.Unmarshal(msgBody, &dto.CourseSemesterCreatedEvent)
	// directly against the flat body — pin that contract.
	wrapped := `{
      "event_id": "00000000-0000-0000-0000-0000000000aa",
      "event_type": "course.semester.created",
      "data": ` + courseSemesterCreatedFixtureFlat + `
    }`

	var got CourseSemesterCreatedEvent
	require.NoError(t, json.Unmarshal([]byte(wrapped), &got))

	assert.Equal(t, uuid.Nil, got.SemesterCourseID)
	assert.Empty(t, got.CourseCode)
	assert.Empty(t, got.InstructorFullname)
}

func TestCourseSemesterCreatedEvent_MissingOptionalFieldsDecodeToZero(t *testing.T) {
	minimal := `{
      "semester_course_id": "11111111-1111-1111-1111-111111111111",
      "course_code": "CS101",
      "course_name": "Intro",
      "credits": 3,
      "semester": "2025-2026-Fall"
    }`

	var got CourseSemesterCreatedEvent
	require.NoError(t, json.Unmarshal([]byte(minimal), &got))

	assert.Empty(t, got.ScheduleSessions, "missing schedule_sessions must decode to empty, not error")
	assert.Empty(t, got.Prerequisites)
	assert.Nil(t, got.InstructorID)
}

// student.* events arrive wrapped under "data" — opposite shape from
// course.semester.created. The consumer (worker/event_consumer.go) does a
// two-step unwrap: BaseEvent first, then re-marshal data and decode into the
// typed struct. Mirror that exact path so the test exercises production logic.

func TestStudentCreatedEvent_RequiresWrappedShape(t *testing.T) {
	wrapped := `{
      "event_id": "00000000-0000-0000-0000-0000000000bb",
      "event_type": "student.created",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "id": "33333333-3333-3333-3333-333333333333",
        "student_number": "20210001",
        "first_name": "Ahmet",
        "last_name": "Yilmaz",
        "email": "ahmet@univ.edu",
        "department": "Computer Science",
        "class_level": 2,
        "advisor_id": "44444444-4444-4444-4444-444444444444",
        "status": "active"
      }
    }`

	// The consumer parses BaseEvent + a private studentEventData. The producer
	// contract is "data.id" (not "data.student_id") — student-service emits the
	// row's primary key as "id". Document that explicitly here.
	var envelope struct {
		EventID   string          `json:"event_id"`
		EventType string          `json:"event_type"`
		Data      json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(wrapped), &envelope))

	var data struct {
		ID            string  `json:"id"`
		StudentNumber string  `json:"student_number"`
		Department    string  `json:"department"`
		ClassLevel    int16   `json:"class_level"`
		AdvisorID     *string `json:"advisor_id"`
	}
	require.NoError(t, json.Unmarshal(envelope.Data, &data))

	assert.Equal(t, "33333333-3333-3333-3333-333333333333", data.ID,
		"json tag is 'id', not 'student_id' — historical contract from student-service")
	assert.Equal(t, "20210001", data.StudentNumber)
	assert.Equal(t, "Computer Science", data.Department)
	assert.Equal(t, int16(2), data.ClassLevel)
	require.NotNil(t, data.AdvisorID)
	assert.Equal(t, "44444444-4444-4444-4444-444444444444", *data.AdvisorID)
}

func TestStudentUpdatedEvent_AdvisorIDIsNullable(t *testing.T) {
	// On update, advisor can be unassigned — null in payload, *uuid.UUID in
	// struct. Pin both nullability and presence shapes.
	withAdvisor := `{
      "event_id": "00000000-0000-0000-0000-0000000000bb",
      "data": {
        "id": "33333333-3333-3333-3333-333333333333",
        "advisor_id": "44444444-4444-4444-4444-444444444444"
      }
    }`
	withoutAdvisor := `{
      "event_id": "00000000-0000-0000-0000-0000000000bb",
      "data": {
        "id": "33333333-3333-3333-3333-333333333333",
        "advisor_id": null
      }
    }`

	parse := func(t *testing.T, s string) *string {
		t.Helper()
		var env struct {
			Data struct {
				AdvisorID *string `json:"advisor_id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal([]byte(s), &env))
		return env.Data.AdvisorID
	}

	got := parse(t, withAdvisor)
	require.NotNil(t, got)
	assert.Equal(t, "44444444-4444-4444-4444-444444444444", *got)

	got = parse(t, withoutAdvisor)
	assert.Nil(t, got, "null advisor_id must decode to nil pointer, not empty string")
}

// =============================================================================
// OUTBOUND — published by enrollment-service via outbox
// =============================================================================
//
// These events are consumed by attendance-service, grades-service, meal-service.
// The matching consumer-side test for the approved variant lives at
//   backend/services/attendance-service/internal/dto/event_contract_test.go
//   (TestEnrollmentProgramApprovedEventData_DecodesWrapped).
// Pinning the producer-side wire shape here closes the contract loop.

func TestEnrollmentProgramApprovedEvent_PublishedShape(t *testing.T) {
	studentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	programID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	approvedBy := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	courseID1 := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	courseID2 := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	evt := EnrollmentProgramApprovedEvent{
		BaseEvent: BaseEvent{
			EventID:   uuid.New(),
			EventType: "enrollment.program.approved",
		},
		ProgramID:  programID,
		StudentID:  studentID,
		Semester:   "2025-2026-Fall",
		CourseIDs:  []uuid.UUID{courseID1, courseID2},
		ApprovedBy: approvedBy,
	}

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	// Every field a downstream consumer might bind to. Renaming any of these
	// silently zeroes the consumer's read.
	for _, k := range []string{
		"event_id", "event_type",
		"program_id", "student_id", "semester", "course_ids", "approved_by",
	} {
		assert.Contains(t, decoded, k, "outbound payload must include %q", k)
	}
	assert.Equal(t, "enrollment.program.approved", decoded["event_type"])
	assert.Equal(t, programID.String(), decoded["program_id"])
	assert.Equal(t, "2025-2026-Fall", decoded["semester"])

	courseIDs, ok := decoded["course_ids"].([]any)
	require.True(t, ok)
	require.Len(t, courseIDs, 2)
	assert.Equal(t, courseID1.String(), courseIDs[0])
}

func TestEnrollmentProgramRejectedEvent_PublishedShape(t *testing.T) {
	evt := EnrollmentProgramRejectedEvent{
		BaseEvent: BaseEvent{
			EventID:   uuid.New(),
			EventType: "enrollment.program.rejected",
		},
		ProgramID:       uuid.New(),
		StudentID:       uuid.New(),
		Semester:        "2025-2026-Fall",
		CourseIDs:       []uuid.UUID{uuid.New()},
		RejectedBy:      uuid.New(),
		RejectionReason: "Schedule conflicts with mandatory course",
	}

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	for _, k := range []string{
		"event_id", "event_type",
		"program_id", "student_id", "semester", "course_ids",
		"rejected_by", "rejection_reason",
	} {
		assert.Contains(t, decoded, k, "outbound payload must include %q", k)
	}
	assert.Equal(t, "Schedule conflicts with mandatory course", decoded["rejection_reason"])
}
