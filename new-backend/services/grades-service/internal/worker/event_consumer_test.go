package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRegistrationWriter struct {
	failFor    map[uuid.UUID]error
	created    []uuid.UUID
	programIDs []uuid.UUID

	deletedProgram uuid.UUID
	deletedCourses []uuid.UUID
}

func (f *fakeRegistrationWriter) CreateRegistration(_ context.Context, arg db.CreateRegistrationParams) (db.GradesStudentCourseRegistration, error) {
	if err := f.failFor[arg.CourseID]; err != nil {
		return db.GradesStudentCourseRegistration{}, err
	}
	f.created = append(f.created, arg.CourseID)
	f.programIDs = append(f.programIDs, utils.PgtypeToUUID(arg.ProgramID))
	return db.GradesStudentCourseRegistration{}, nil
}

func (f *fakeRegistrationWriter) DeleteProgramRegistrations(_ context.Context, programID, _ uuid.UUID, courseIDs []uuid.UUID) (int64, error) {
	f.deletedProgram = programID
	f.deletedCourses = courseIDs
	return int64(len(courseIDs)), nil
}

func (f *fakeRegistrationWriter) MarkAttendanceFailed(context.Context, db.MarkAttendanceFailedParams) error {
	return nil
}

// programApprovedBody builds enrollment.program.approved the way it arrives:
// the enrollment outbox payload is itself an envelope, wrapped once more by
// the outbox worker.
func programApprovedBody(t *testing.T, courseIDs []uuid.UUID) []byte {
	t.Helper()
	return programApprovedBodyFor(t, uuid.New(), courseIDs)
}

func programApprovedBodyFor(t *testing.T, programID uuid.UUID, courseIDs []uuid.UUID) []byte {
	t.Helper()
	ids := make([]string, len(courseIDs))
	for i, id := range courseIDs {
		ids[i] = id.String()
	}
	body, err := json.Marshal(map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "enrollment.program.approved",
		"data": map[string]any{
			"event_id":   uuid.New().String(),
			"event_type": "enrollment.program.approved",
			"data": map[string]any{
				"program_id": programID.String(),
				"student_id": uuid.New().String(),
				"semester":   "2025-2026-Fall",
				"course_ids": ids,
			},
		},
	})
	require.NoError(t, err)
	return body
}

func newTestEventConsumer(t *testing.T, reg *fakeRegistrationWriter) *EventConsumer {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	return NewEventConsumer(nil, nil, reg)
}

func TestHandleEnrollmentApproved_OneCourseFails_ReturnsError(t *testing.T) {
	courses := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	dbErr := errors.New("violates foreign key constraint")
	reg := &fakeRegistrationWriter{failFor: map[uuid.UUID]error{courses[4]: dbErr}}
	w := newTestEventConsumer(t, reg)

	err := w.handleMessage(t.Context(), programApprovedBody(t, courses))

	require.ErrorIs(t, err, dbErr, "acking would leave course 5 without a registration for good")
	assert.Len(t, reg.created, 4)
}

func TestHandleEnrollmentApproved_AllCoursesSucceed_ReturnsNil(t *testing.T) {
	courses := []uuid.UUID{uuid.New(), uuid.New()}
	reg := &fakeRegistrationWriter{}
	w := newTestEventConsumer(t, reg)
	programID := uuid.New()

	err := w.handleMessage(t.Context(), programApprovedBodyFor(t, programID, courses))

	require.NoError(t, err)
	assert.Equal(t, courses, reg.created)
	assert.Equal(t, []uuid.UUID{programID, programID}, reg.programIDs, "the rows must name their program, or its cancel cannot find them")
}

// The cancelled payload is flat under the envelope's data, one level less
// than approved's.
func TestHandleEnrollmentCancelled_RemovesThatProgramsCourses(t *testing.T) {
	courses := []uuid.UUID{uuid.New(), uuid.New()}
	reg := &fakeRegistrationWriter{}
	w := newTestEventConsumer(t, reg)
	programID := uuid.New()
	body, err := json.Marshal(map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "enrollment.program.cancelled",
		"data": map[string]any{
			"program_id":   programID.String(),
			"student_id":   uuid.New().String(),
			"semester":     "2026-2027-Spring",
			"course_ids":   []string{courses[0].String(), courses[1].String()},
			"cancelled_by": "student",
			"cancel_type":  "auto_replace",
		},
	})
	require.NoError(t, err)

	require.NoError(t, w.handleMessage(t.Context(), body))
	assert.Equal(t, programID, reg.deletedProgram)
	assert.Equal(t, courses, reg.deletedCourses)
}
