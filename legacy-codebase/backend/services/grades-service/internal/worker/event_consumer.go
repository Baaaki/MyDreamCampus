package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/grades-service/internal/db"
	"github.com/baaaki/mydreamcampus/grades-service/internal/dto"
	"github.com/baaaki/mydreamcampus/grades-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"go.uber.org/zap"
)

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
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EventConsumer"))
	log.Info("starting event consumer")

	// Setup student events queue
	studentQueue := "grades-service-student"
	if err := w.consumer.DeclareQueue(studentQueue); err != nil {
		return err
	}
	studentRoutingKeys := []string{
		"student.created",
		"student.updated",
		"student.deactivated",
	}
	for _, key := range studentRoutingKeys {
		if err := w.consumer.BindQueue(studentQueue, "student.events", key); err != nil {
			return err
		}
	}
	if err := w.consumer.Consume(studentQueue, func(body []byte) error {
		return w.handleStudentEvent(ctx, body)
	}); err != nil {
		return err
	}

	// Setup course events queue
	courseQueue := "grades-service-course"
	if err := w.consumer.DeclareQueue(courseQueue); err != nil {
		return err
	}
	courseRoutingKeys := []string{
		"course.semester.created",
	}
	for _, key := range courseRoutingKeys {
		if err := w.consumer.BindQueue(courseQueue, "course.events", key); err != nil {
			return err
		}
	}
	if err := w.consumer.Consume(courseQueue, func(body []byte) error {
		return w.handleCourseEvent(ctx, body)
	}); err != nil {
		return err
	}

	// Setup enrollment events queue
	enrollmentQueue := "grades-service-enrollment"
	if err := w.consumer.DeclareQueue(enrollmentQueue); err != nil {
		return err
	}
	if err := w.consumer.BindQueue(enrollmentQueue, "enrollment.events", "enrollment.program.approved"); err != nil {
		return err
	}
	if err := w.consumer.Consume(enrollmentQueue, func(body []byte) error {
		return w.handleEnrollmentEvent(ctx, body)
	}); err != nil {
		return err
	}

	// Setup attendance events queue
	attendanceQueue := "grades-service-attendance"
	if err := w.consumer.DeclareQueue(attendanceQueue); err != nil {
		return err
	}
	if err := w.consumer.BindQueue(attendanceQueue, "attendance.events", events.EventAttendanceSemesterFailed); err != nil {
		return err
	}
	if err := w.consumer.Consume(attendanceQueue, func(body []byte) error {
		return w.handleAttendanceEvent(ctx, body)
	}); err != nil {
		return err
	}

	log.Info("event consumer started")

	// Block until context is cancelled
	<-ctx.Done()
	log.Info("event consumer stopped")

	return nil
}

// ============================================
// Student Event Handlers
// ============================================

func (w *EventConsumer) handleStudentEvent(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentEvent"),
	)

	// Parse event type
	var baseEvent struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(body, &baseEvent); err != nil {
		log.Error("failed to unmarshal base event", zap.Error(err))
		return err
	}

	switch baseEvent.EventType {
	case "student.created":
		var event dto.StudentCreatedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error("failed to unmarshal student.created event", zap.Error(err))
			return err
		}
		return w.handleStudentCreated(ctx, event)

	case "student.updated":
		var event dto.StudentUpdatedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error("failed to unmarshal student.updated event", zap.Error(err))
			return err
		}
		return w.handleStudentUpdated(ctx, event)

	case "student.deactivated":
		var event dto.StudentDeactivatedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error("failed to unmarshal student.deactivated event", zap.Error(err))
			return err
		}
		return w.handleStudentDeactivated(ctx, event)

	default:
		log.Warn("unknown student event type", zap.String("event_type", baseEvent.EventType))
		return nil
	}
}

func (w *EventConsumer) handleStudentCreated(ctx context.Context, event dto.StudentCreatedEvent) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentCreated"),
	)

	log.Info("handling student.created event", zap.String("student_id", event.Data.ID.String()))

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

	return err
}

func (w *EventConsumer) handleStudentUpdated(ctx context.Context, event dto.StudentUpdatedEvent) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentUpdated"),
	)

	log.Info("handling student.updated event", zap.String("student_id", event.Data.ID.String()))

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

	return err
}

func (w *EventConsumer) handleStudentDeactivated(ctx context.Context, event dto.StudentDeactivatedEvent) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentDeactivated"),
	)

	log.Info("handling student.deactivated event", zap.String("student_id", event.Data.ID.String()))

	return w.cacheRepo.DeactivateStudentCache(ctx, event.Data.ID)
}

// ============================================
// Course Event Handlers
// ============================================

func (w *EventConsumer) handleCourseEvent(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleCourseEvent"),
	)

	// Parse event type
	var baseEvent struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(body, &baseEvent); err != nil {
		log.Error("failed to unmarshal base event", zap.Error(err))
		return err
	}

	switch baseEvent.EventType {
	case "course.semester.created":
		var event dto.CourseSemesterCreatedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error("failed to unmarshal course.semester.created event", zap.Error(err))
			return err
		}
		return w.handleCourseSemesterCreated(ctx, event)

	default:
		log.Warn("unknown course event type", zap.String("event_type", baseEvent.EventType))
		return nil
	}
}

func (w *EventConsumer) handleCourseSemesterCreated(ctx context.Context, event dto.CourseSemesterCreatedEvent) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleCourseSemesterCreated"),
	)

	log.Info("handling course.semester.created event", zap.String("course_id", event.SemesterCourseID.String()))

	// Marshal assessment schema to JSONB
	schemaJSON, err := json.Marshal(event.AssessmentSchema)
	if err != nil {
		log.Error("failed to marshal assessment schema", zap.Error(err))
		return err
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

	return err
}

// ============================================
// Enrollment Event Handlers
// ============================================

func (w *EventConsumer) handleEnrollmentEvent(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleEnrollmentEvent"),
	)

	var event dto.EnrollmentProgramApprovedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal enrollment.program_approved event", zap.Error(err))
		return err
	}

	log.Info("handling enrollment.program_approved event",
		zap.String("student_id", event.Data.StudentID.String()),
		zap.Int("courses", len(event.Data.CourseIDs)),
	)

	// Create registrations for each course
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
			log.Error("failed to create registration",
				zap.Error(err),
				zap.String("student_id", event.Data.StudentID.String()),
				zap.String("course_id", courseID.String()),
			)
			lastErr = err
			continue
		}
		successCount++
	}

	// If all registrations failed (e.g. FK constraint: student/course not in cache yet), requeue for retry
	if successCount == 0 && lastErr != nil {
		log.Warn("all registrations failed, will retry",
			zap.String("student_id", event.Data.StudentID.String()),
			zap.Int("total_courses", len(event.Data.CourseIDs)),
			zap.Error(lastErr),
		)
		return lastErr
	}

	return nil
}

// ============================================
// Attendance Event Handlers
// ============================================

func (w *EventConsumer) handleAttendanceEvent(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleAttendanceEvent"),
	)

	var event dto.AttendanceSemesterFailedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal attendance.semester.failed event", zap.Error(err))
		return err
	}

	log.Info("handling attendance.semester.failed event",
		zap.String("student_id", event.Data.StudentID.String()),
		zap.String("course_id", event.Data.CourseID.String()),
	)

	return w.regRepo.MarkAttendanceFailed(ctx, db.MarkAttendanceFailedParams{
		StudentID: event.Data.StudentID,
		CourseID:  event.Data.CourseID,
	})
}
