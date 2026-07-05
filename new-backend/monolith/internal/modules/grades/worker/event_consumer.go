package worker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// QueueSyncEvents feeds the grades module's local projections: student and
// course caches, registrations and attendance failures. Bindings are
// pre-declared in main.go (DeclareDownstreamBindings) so events published
// while this consumer is down stay queued.
const QueueSyncEvents = "grades.sync_events"

// outboxEnvelope is the wrapper the shared OutboxWorker puts on the wire:
// {event_id, event_type, timestamp, data}. Data stays raw because each
// producer ships a differently-shaped payload under it (flat for
// student/catalog, wrapped-once-more for enrollment/attendance).
type outboxEnvelope struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// EventConsumer keeps grades' local view tables and registrations in sync.
// Grades has no processed_events table — every handler is naturally
// idempotent (upserts, ON CONFLICT DO NOTHING, boolean set), so redelivery
// after a nack cannot corrupt state.
type EventConsumer struct {
	consumer  *rabbitmq.Consumer
	cacheRepo *repository.CacheRepository
	regRepo   *repository.RegistrationRepository
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	cacheRepo *repository.CacheRepository,
	regRepo *repository.RegistrationRepository,
) *EventConsumer {
	return &EventConsumer{
		consumer:  consumer,
		cacheRepo: cacheRepo,
		regRepo:   regRepo,
	}
}

func (w *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "GradesEventConsumer"))

	if err := w.consumer.Consume(QueueSyncEvents, func(body []byte) error {
		return w.handleMessage(ctx, body)
	}); err != nil {
		log.Error("failed to start consuming", zap.Error(err))
		return err
	}

	log.Info("grades event consumer started", zap.String("queue", QueueSyncEvents))
	return nil
}

func (w *EventConsumer) handleMessage(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "GradesEventConsumer"),
		zap.String("method", "handleMessage"),
	)

	var env outboxEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		log.Error("malformed envelope, dropping", zap.Error(err))
		return nil // requeueing a permanently broken message loops forever
	}

	switch env.EventType {
	case "student.created", "student.updated":
		return w.handleStudentUpsert(ctx, env)
	case "student.deactivated":
		return w.handleStudentDeactivated(ctx, env)
	case "course.semester.created":
		return w.handleCourseSemesterCreated(ctx, env)
	case "enrollment.program.approved":
		return w.handleEnrollmentApproved(ctx, env)
	case "attendance.semester.failed":
		return w.handleAttendanceFailed(ctx, env)
	default:
		log.Warn("unknown event type, dropping", zap.String("event_type", env.EventType))
		return nil
	}
}

func (w *EventConsumer) handleStudentUpsert(ctx context.Context, env outboxEnvelope) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "GradesEventConsumer"),
		zap.String("method", "handleStudentUpsert"),
		zap.String("event_type", env.EventType),
	)

	// student payloads are flat under the envelope's data key.
	var event dto.StudentCreatedEvent
	if err := json.Unmarshal(env.Data, &event.Data); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	_, err := w.cacheRepo.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID:            event.Data.ID,
		StudentNumber: event.Data.StudentNumber,
		FirstName:     utils.StringToPgText(event.Data.FirstName),
		LastName:      utils.StringToPgText(event.Data.LastName),
		Email:         utils.StringToPgText(event.Data.Email),
		Department:    utils.StringToPgText(event.Data.Department),
		ClassLevel:    utils.Int16ToPgInt2(event.Data.ClassLevel),
		IsActive:      utils.BoolToPgBool(true),
	})
	if err != nil {
		log.Error("failed to upsert student cache", zap.Error(err),
			zap.String("student_id", event.Data.ID.String()))
		return err
	}

	log.Info("student cache synced", zap.String("student_id", event.Data.ID.String()))
	return nil
}

func (w *EventConsumer) handleStudentDeactivated(ctx context.Context, env outboxEnvelope) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "GradesEventConsumer"),
		zap.String("method", "handleStudentDeactivated"),
	)

	var event dto.StudentDeactivatedEvent
	if err := json.Unmarshal(env.Data, &event.Data); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	if err := w.cacheRepo.DeactivateStudentCache(ctx, event.Data.ID); err != nil {
		log.Error("failed to deactivate student cache", zap.Error(err))
		return err
	}

	log.Info("student cache deactivated", zap.String("student_id", event.Data.ID.String()))
	return nil
}

func (w *EventConsumer) handleCourseSemesterCreated(ctx context.Context, env outboxEnvelope) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "GradesEventConsumer"),
		zap.String("method", "handleCourseSemesterCreated"),
	)

	// catalog's payload is flat under data (it carries its own event meta
	// alongside the course fields — extra keys are ignored here).
	var event dto.CourseSemesterCreatedEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	schemaJSON, err := json.Marshal(event.AssessmentSchema)
	if err != nil {
		log.Error("failed to marshal assessment schema, dropping", zap.Error(err))
		return nil
	}

	_, err = w.cacheRepo.UpsertCourseCache(ctx, db.UpsertCourseCacheParams{
		ID:                 event.SemesterCourseID,
		CourseCode:         event.CourseCode,
		CourseName:         event.CourseName,
		Credits:            event.Credits,
		Semester:           event.Semester,
		Department:         utils.StringToPgText(event.Department),
		InstructorID:       event.InstructorID,
		InstructorFullname: utils.StringToPgText(event.InstructorFullname),
		AssessmentSchema:   schemaJSON,
	})
	if err != nil {
		log.Error("failed to upsert course cache", zap.Error(err),
			zap.String("course_id", event.SemesterCourseID.String()))
		return err
	}

	// Every prerequisite of this course is itself "a course that is a
	// prerequisite of something" — that is what prerequisite_courses_view
	// tracks, and what gates the prerequisite.passed publish at finalize.
	for _, prereq := range event.Prerequisites {
		if err := w.cacheRepo.UpsertPrerequisiteCourse(ctx, db.UpsertPrerequisiteCourseParams{
			CourseCode: prereq.CourseCode,
			CourseID:   prereq.ID,
		}); err != nil {
			log.Error("failed to upsert prerequisite course", zap.Error(err),
				zap.String("prerequisite_code", prereq.CourseCode))
			return err
		}
	}

	log.Info("course cache synced",
		zap.String("course_id", event.SemesterCourseID.String()),
		zap.String("course_code", event.CourseCode),
		zap.Int("prerequisite_count", len(event.Prerequisites)),
	)
	return nil
}

func (w *EventConsumer) handleEnrollmentApproved(ctx context.Context, env outboxEnvelope) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "GradesEventConsumer"),
		zap.String("method", "handleEnrollmentApproved"),
	)

	// enrollment's outbox payload is itself wrapped, so env.Data is
	// {event_id, ..., data: {student_id, course_ids, ...}}.
	var event dto.EnrollmentProgramApprovedEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	successCount := 0
	var lastErr error
	for _, courseID := range event.Data.CourseIDs {
		_, err := w.regRepo.CreateRegistration(ctx, db.CreateRegistrationParams{
			StudentID:          event.Data.StudentID,
			CourseID:           courseID,
			Semester:           event.Data.Semester,
			IsAttendanceFailed: utils.BoolToPgBool(false),
		})
		if err != nil {
			// ON CONFLICT DO NOTHING + :one → pgx.ErrNoRows on redelivery
			// of an already-registered pair; that is success, not failure.
			if errors.Is(err, pgx.ErrNoRows) {
				successCount++
				continue
			}
			log.Error("failed to create registration", zap.Error(err),
				zap.String("student_id", event.Data.StudentID.String()),
				zap.String("course_id", courseID.String()),
			)
			lastErr = err
			continue
		}
		successCount++
	}

	// All failed usually means the student/course rows haven't landed in
	// the local caches yet (FK) — requeue and let the sync events win the race.
	if successCount == 0 && lastErr != nil {
		log.Warn("all registrations failed, requeueing",
			zap.String("student_id", event.Data.StudentID.String()),
			zap.Int("total_courses", len(event.Data.CourseIDs)),
			zap.Error(lastErr),
		)
		return lastErr
	}

	log.Info("registrations created",
		zap.String("student_id", event.Data.StudentID.String()),
		zap.Int("count", successCount),
	)
	return nil
}

func (w *EventConsumer) handleAttendanceFailed(ctx context.Context, env outboxEnvelope) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "GradesEventConsumer"),
		zap.String("method", "handleAttendanceFailed"),
	)

	// attendance's outbox payload is BaseEvent-shaped, so env.Data is
	// {event_id, ..., data: {student_id, course_id, ...}}.
	var event dto.AttendanceSemesterFailedEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	if err := w.regRepo.MarkAttendanceFailed(ctx, db.MarkAttendanceFailedParams{
		StudentID: event.Data.StudentID,
		CourseID:  event.Data.CourseID,
	}); err != nil {
		log.Error("failed to mark attendance failed", zap.Error(err))
		return err
	}

	log.Info("registration marked attendance-failed",
		zap.String("student_id", event.Data.StudentID.String()),
		zap.String("course_id", event.Data.CourseID.String()),
	)
	return nil
}
