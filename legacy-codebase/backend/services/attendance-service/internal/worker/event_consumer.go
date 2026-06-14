package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/db"
	"github.com/baaaki/mydreamcampus/attendance-service/internal/dto"
	"github.com/baaaki/mydreamcampus/attendance-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EventConsumer struct {
	consumer  *rabbitmq.Consumer
	cacheRepo *repository.CacheRepository
	eventRepo *repository.EventRepository
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	cacheRepo *repository.CacheRepository,
	eventRepo *repository.EventRepository,
) *EventConsumer {
	return &EventConsumer{
		consumer:  consumer,
		cacheRepo: cacheRepo,
		eventRepo: eventRepo,
	}
}

func (w *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EventConsumer"))
	log.Info("Event consumer started")

	// Start consuming from the queue. The closure captures the root ctx so
	// in-flight event processing is canceled on graceful shutdown.
	if err := w.consumer.Consume("attendance.events", func(body []byte) error {
		return w.handleMessage(ctx, body)
	}); err != nil {
		log.Error("failed to start consuming", zap.Error(err))
		return err
	}

	go func() {
		<-ctx.Done()
		log.Info("Event consumer stopped")
	}()

	return nil
}

func (w *EventConsumer) handleMessage(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleMessage"),
	)

	var event dto.BaseEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal event", zap.Error(err))
		return err
	}

	// Check if already processed (idempotency)
	eventID, err := uuid.Parse(event.EventID.String())
	if err != nil {
		log.Error("invalid event ID", zap.Error(err))
		return err
	}

	processed, err := w.eventRepo.IsEventProcessed(ctx, eventID)
	if err != nil {
		log.Error("failed to check if event processed", zap.Error(err))
		return err
	}

	if processed {
		log.Debug("event already processed, skipping", zap.String("event_id", eventID.String()))
		return nil
	}

	log.Info("processing event",
		zap.String("type", event.EventType),
		zap.String("event_id", eventID.String()),
	)

	// Route to appropriate handler based on event type
	switch event.EventType {
	case "student.created":
		return w.handleStudentCreated(ctx, body, eventID)
	case "student.updated":
		return w.handleStudentUpdated(ctx, body, eventID)
	case "student.deactivated":
		return w.handleStudentDeactivated(ctx, body, eventID)
	case "course.semester.created":
		return w.handleCourseSemesterCreated(ctx, body, eventID)
	case "enrollment.program.approved":
		return w.handleEnrollmentProgramApproved(ctx, body, eventID)
	default:
		log.Warn("unknown event type", zap.String("type", event.EventType))
		// Mark as processed anyway to avoid reprocessing
		return w.eventRepo.MarkEventProcessed(ctx, eventID, event.EventType)
	}
}

func (w *EventConsumer) handleStudentCreated(ctx context.Context, body []byte, eventID uuid.UUID) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentCreated"),
	)

	eventData, err := unwrapEventData[dto.StudentCreatedEventData](body)
	if err != nil {
		return err
	}

	// Upsert student to cache
	err = w.cacheRepo.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID:            utils.UUIDToPgUUID(eventData.StudentID),
		StudentNumber: eventData.StudentNumber,
		FirstName:     utils.StringToPgText(eventData.FirstName),
		LastName:      utils.StringToPgText(eventData.LastName),
		Email:         utils.StringToPgText(eventData.Email),
		Department:    utils.StringToPgText(eventData.Department),
		IsActive:      utils.BoolToPgBool(true),
	})
	if err != nil {
		log.Error("failed to upsert student cache", zap.Error(err))
		return err
	}

	log.Info("student cache created",
		zap.String("student_id", eventData.StudentID.String()),
		zap.String("student_number", eventData.StudentNumber),
	)

	return w.eventRepo.MarkEventProcessed(ctx, eventID, "student.created")
}

func (w *EventConsumer) handleStudentUpdated(ctx context.Context, body []byte, eventID uuid.UUID) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentUpdated"),
	)

	// Two-step unwrap: BaseEvent wraps the actual data in "data" field
	var baseEvent dto.BaseEvent
	if err := json.Unmarshal(body, &baseEvent); err != nil {
		return err
	}
	dataBytes, err := json.Marshal(baseEvent.Data)
	if err != nil {
		return err
	}
	var eventData dto.StudentUpdatedEventData
	if err := json.Unmarshal(dataBytes, &eventData); err != nil {
		return err
	}

	// Upsert student to cache
	err = w.cacheRepo.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID:            utils.UUIDToPgUUID(eventData.StudentID),
		StudentNumber: eventData.StudentNumber,
		FirstName:     utils.StringToPgText(eventData.FirstName),
		LastName:      utils.StringToPgText(eventData.LastName),
		Email:         utils.StringToPgText(eventData.Email),
		Department:    utils.StringToPgText(eventData.Department),
		IsActive:      utils.BoolToPgBool(true),
	})
	if err != nil {
		log.Error("failed to upsert student cache", zap.Error(err))
		return err
	}

	log.Info("student cache updated",
		zap.String("student_id", eventData.StudentID.String()),
		zap.String("student_number", eventData.StudentNumber),
	)

	return w.eventRepo.MarkEventProcessed(ctx, eventID, "student.updated")
}

func (w *EventConsumer) handleStudentDeactivated(ctx context.Context, body []byte, eventID uuid.UUID) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentDeactivated"),
	)

	// Two-step unwrap: BaseEvent wraps the actual data in "data" field
	var baseEvent dto.BaseEvent
	if err := json.Unmarshal(body, &baseEvent); err != nil {
		return err
	}
	dataBytes, err := json.Marshal(baseEvent.Data)
	if err != nil {
		return err
	}
	var eventData dto.StudentDeactivatedEventData
	if err := json.Unmarshal(dataBytes, &eventData); err != nil {
		return err
	}

	// Deactivate student in cache
	if err := w.cacheRepo.DeactivateStudentCache(ctx, eventData.StudentID); err != nil {
		log.Error("failed to deactivate student cache", zap.Error(err))
		return err
	}

	log.Info("student cache deactivated", zap.String("student_id", eventData.StudentID.String()))

	return w.eventRepo.MarkEventProcessed(ctx, eventID, "student.deactivated")
}

func (w *EventConsumer) handleCourseSemesterCreated(ctx context.Context, body []byte, eventID uuid.UUID) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleCourseSemesterCreated"),
	)

	var eventData dto.CourseSemesterCreatedEventData
	if err := json.Unmarshal(body, &eventData); err != nil {
		return err
	}

	// Check if any schedule session has type "lab"
	hasLab := false
	for _, s := range eventData.ScheduleSessions {
		if s.SessionType == "lab" {
			hasLab = true
			break
		}
	}

	// Upsert course to cache
	err := w.cacheRepo.UpsertCourseCache(ctx, db.UpsertCourseCacheParams{
		ID:                 utils.UUIDToPgUUID(eventData.SemesterCourseID),
		CourseCode:         eventData.CourseCode,
		CourseName:         eventData.CourseName,
		Credits:            eventData.Credits,
		Semester:           eventData.Semester,
		Department:         utils.StringToPgText(eventData.Department),
		InstructorID:       utils.UUIDToPgUUID(eventData.InstructorID),
		InstructorFullname: utils.StringToPgText(eventData.InstructorFullname),
		TotalWeeks:         utils.Int16ToPgInt2(14), // default, catalog doesn't send this
		HasLab:             hasLab,
	})
	if err != nil {
		log.Error("failed to upsert course cache", zap.Error(err))
		return err
	}

	log.Info("course cache created",
		zap.String("course_id", eventData.SemesterCourseID.String()),
		zap.String("course_code", eventData.CourseCode),
	)

	return w.eventRepo.MarkEventProcessed(ctx, eventID, "course.semester.created")
}

func (w *EventConsumer) handleEnrollmentProgramApproved(ctx context.Context, body []byte, eventID uuid.UUID) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleEnrollmentProgramApproved"),
	)

	// Parse the wrapped event: { event_id, event_type, data: { ... } }
	var wrapper struct {
		Data dto.EnrollmentProgramApprovedEventData `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return err
	}
	eventData := wrapper.Data

	// Create enrollment cache entries for each course
	successCount := 0
	var lastErr error
	for _, courseID := range eventData.CourseIDs {
		err := w.cacheRepo.CreateEnrollmentCache(
			ctx,
			eventData.StudentID,
			courseID,
			eventData.Semester,
		)
		if err != nil {
			log.Error("failed to create enrollment cache",
				zap.String("student_id", eventData.StudentID.String()),
				zap.String("course_id", courseID.String()),
				zap.Error(err),
			)
			lastErr = err
			continue
		}

		successCount++
		log.Info("enrollment cache created",
			zap.String("student_id", eventData.StudentID.String()),
			zap.String("course_id", courseID.String()),
			zap.String("semester", eventData.Semester),
		)
	}

	// If all enrollments failed (e.g. FK constraint: student not in cache yet), requeue for retry
	if successCount == 0 && lastErr != nil {
		log.Warn("all enrollment cache entries failed, will retry",
			zap.String("student_id", eventData.StudentID.String()),
			zap.Int("total_courses", len(eventData.CourseIDs)),
			zap.Error(lastErr),
		)
		return lastErr
	}

	return w.eventRepo.MarkEventProcessed(ctx, eventID, "enrollment.program.approved")
}
