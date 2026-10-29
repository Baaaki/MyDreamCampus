package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCacheWriter struct {
	failFor    map[uuid.UUID]error
	created    []uuid.UUID
	enrolledAt []time.Time
	programIDs []uuid.UUID

	deletedProgram uuid.UUID
	deletedCourses []uuid.UUID
}

func (f *fakeCacheWriter) UpsertStudentCache(context.Context, db.UpsertStudentCacheParams) error {
	return nil
}

func (f *fakeCacheWriter) DeactivateStudentCache(context.Context, uuid.UUID) error { return nil }

func (f *fakeCacheWriter) UpsertCourseCache(context.Context, db.UpsertCourseCacheParams) error {
	return nil
}

func (f *fakeCacheWriter) CreateEnrollmentCache(_ context.Context, programID, _, courseID uuid.UUID, _ string, enrolledAt time.Time) error {
	if err := f.failFor[courseID]; err != nil {
		return err
	}
	f.created = append(f.created, courseID)
	f.enrolledAt = append(f.enrolledAt, enrolledAt)
	f.programIDs = append(f.programIDs, programID)
	return nil
}

func (f *fakeCacheWriter) DeleteProgramEnrollments(_ context.Context, programID, _ uuid.UUID, courseIDs []uuid.UUID) (int64, error) {
	f.deletedProgram = programID
	f.deletedCourses = courseIDs
	return int64(len(courseIDs)), nil
}

type fakeProcessedEvents struct {
	marked []uuid.UUID
}

func (f *fakeProcessedEvents) IsEventProcessed(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func (f *fakeProcessedEvents) MarkEventProcessed(_ context.Context, eventID uuid.UUID, _ string) error {
	f.marked = append(f.marked, eventID)
	return nil
}

var approvedAt = time.Date(2026, 10, 29, 9, 30, 0, 0, time.UTC)

// programApprovedBody builds enrollment.program.approved the way it arrives:
// the enrollment outbox payload is itself an envelope, wrapped once more by
// the outbox worker.
func programApprovedBody(t *testing.T, eventID uuid.UUID, courseIDs []uuid.UUID) []byte {
	t.Helper()
	return programApprovedBodyFor(t, eventID, uuid.New(), courseIDs)
}

func programApprovedBodyFor(t *testing.T, eventID, programID uuid.UUID, courseIDs []uuid.UUID) []byte {
	t.Helper()
	ids := make([]string, len(courseIDs))
	for i, id := range courseIDs {
		ids[i] = id.String()
	}
	return mustMarshal(t, map[string]any{
		"event_id":   eventID.String(),
		"event_type": "enrollment.program.approved",
		"data": map[string]any{
			"event_id":   eventID.String(),
			"event_type": "enrollment.program.approved",
			"data": map[string]any{
				"program_id":  programID.String(),
				"student_id":  uuid.New().String(),
				"semester":    "2025-2026-Fall",
				"course_ids":  ids,
				"approved_at": approvedAt.Format(time.RFC3339Nano),
			},
		},
	})
}

func newTestEventConsumer(t *testing.T, cache *fakeCacheWriter, events *fakeProcessedEvents) *EventConsumer {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	return NewEventConsumer(nil, cache, events)
}

func TestHandleEnrollmentProgramApproved_OneCourseFails_ReturnsErrorAndStaysUnprocessed(t *testing.T) {
	courses := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	dbErr := errors.New("connection reset")
	cache := &fakeCacheWriter{failFor: map[uuid.UUID]error{courses[2]: dbErr}}
	events := &fakeProcessedEvents{}
	w := newTestEventConsumer(t, cache, events)

	err := w.handleMessage(t.Context(), programApprovedBody(t, uuid.New(), courses))

	require.ErrorIs(t, err, dbErr, "the event must be redelivered, or course 3 never reaches the roster")
	assert.Len(t, cache.created, 4)
	assert.Empty(t, events.marked, "a processed mark would make the redelivery skip the missing course")
}

func TestHandleEnrollmentProgramApproved_AllCoursesSucceed_MarksProcessed(t *testing.T) {
	courses := []uuid.UUID{uuid.New(), uuid.New()}
	cache := &fakeCacheWriter{}
	events := &fakeProcessedEvents{}
	w := newTestEventConsumer(t, cache, events)
	eventID, programID := uuid.New(), uuid.New()

	err := w.handleMessage(t.Context(), programApprovedBodyFor(t, eventID, programID, courses))

	require.NoError(t, err)
	assert.Equal(t, courses, cache.created)
	assert.Equal(t, []time.Time{approvedAt, approvedAt}, cache.enrolledAt, "the student is held to sessions from the approval on")
	assert.Equal(t, []uuid.UUID{programID, programID}, cache.programIDs, "the rows must name their program, or its cancel cannot find them")
	assert.Equal(t, []uuid.UUID{eventID}, events.marked)
}

// The cancelled payload is flat under the envelope's data, one level less
// than approved's: reading it as approved's shape found no program at all.
func TestHandleEnrollmentProgramCancelled_RemovesThatProgramsCourses(t *testing.T) {
	courses := []uuid.UUID{uuid.New(), uuid.New()}
	cache := &fakeCacheWriter{}
	events := &fakeProcessedEvents{}
	w := newTestEventConsumer(t, cache, events)
	eventID, programID := uuid.New(), uuid.New()

	body := mustMarshal(t, map[string]any{
		"event_id":   eventID.String(),
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

	require.NoError(t, w.handleMessage(t.Context(), body))
	assert.Equal(t, programID, cache.deletedProgram)
	assert.Equal(t, courses, cache.deletedCourses)
	assert.Equal(t, []uuid.UUID{eventID}, events.marked)
}
