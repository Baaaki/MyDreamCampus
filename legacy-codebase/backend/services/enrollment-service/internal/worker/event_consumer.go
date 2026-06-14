package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/dto"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/repository"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/service"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// wrappedEvent represents an event with nested data (from student-service)
type wrappedEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// studentEventData represents student data inside the wrapped event
type studentEventData struct {
	ID            string  `json:"id"`
	StudentNumber string  `json:"student_number"`
	Email         string  `json:"email"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Department    string  `json:"department"`
	ClassLevel    int16   `json:"class_level"`
	AdvisorID     *string `json:"advisor_id"`
	Status        string  `json:"status"`
}

type EventConsumer struct {
	consumer            *rabbitmq.Consumer
	eventService        *service.EventService
	processedEventsRepo *repository.ProcessedEventsRepository
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	eventService *service.EventService,
	processedEventsRepo *repository.ProcessedEventsRepository,
) *EventConsumer {
	return &EventConsumer{
		consumer:            consumer,
		eventService:        eventService,
		processedEventsRepo: processedEventsRepo,
	}
}

// Start begins consuming events from RabbitMQ
func (c *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EventConsumer"))
	log.Info("starting event consumer")

	// Consume enrollment events (both course and student events)
	// Use a combined queue name for enrollment service
	err := c.consumer.Consume("enrollment.events", func(msg []byte) error {
		return c.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Info("event consumer started successfully")
	return nil
}

// handleMessage processes incoming messages
func (c *EventConsumer) handleMessage(ctx context.Context, msgBody []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleMessage"),
	)

	log.Info("received event")

	// Parse generic event to get event_id
	var genericEvent dto.BaseEvent
	if err := json.Unmarshal(msgBody, &genericEvent); err != nil {
		log.Error("failed to unmarshal event",
			zap.Error(err),
		)
		return err // Don't requeue malformed messages
	}

	// Check if event already processed (idempotency)
	processed, err := c.processedEventsRepo.IsEventProcessed(ctx, genericEvent.EventID)
	if err != nil {
		log.Error("failed to check event processing status",
			zap.Error(err),
			zap.String("event_id", genericEvent.EventID.String()),
		)
		return err // Requeue for retry
	}

	if processed {
		log.Info("event already processed, skipping",
			zap.String("event_id", genericEvent.EventID.String()),
			zap.String("event_type", genericEvent.EventType),
		)
		return nil
	}

	// Route to appropriate handler based on event type
	switch genericEvent.EventType {
	// Course semester events
	case events.EventCourseSemesterCreated:
		return c.handleCourseSemesterCreated(ctx, msgBody, genericEvent.EventID.String())

	// Student events
	case events.EventStudentCreated:
		return c.handleStudentCreated(ctx, msgBody, genericEvent.EventID.String())
	case events.EventStudentUpdated:
		return c.handleStudentUpdated(ctx, msgBody, genericEvent.EventID.String())
	case events.EventStudentDeactivated:
		return c.handleStudentDeactivated(ctx, msgBody, genericEvent.EventID.String())

	// Grade events
	case events.EventGradeStudentPrerequisitePassed:
		return c.handleGradeStudentPrerequisitePassed(ctx, msgBody, genericEvent.EventID.String())

	default:
		log.Warn("unknown event type",
			zap.String("event_type", genericEvent.EventType),
		)
		return nil // Acknowledge unknown events to avoid DLQ
	}
}

// handleCourseSemesterCreated handles course.semester.created events
func (c *EventConsumer) handleCourseSemesterCreated(ctx context.Context, msgBody []byte, eventID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleCourseSemesterCreated"),
	)

	var event dto.CourseSemesterCreatedEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		log.Error("failed to unmarshal course.semester.created event",
			zap.Error(err),
		)
		return err
	}

	log.Info("processing course.semester.created event",
		zap.String("event_id", eventID),
		zap.String("course_id", event.SemesterCourseID.String()),
	)

	err := c.eventService.HandleCourseSemesterCreated(ctx, event)
	if err != nil {
		log.Error("failed to process course.semester.created event",
			zap.Error(err),
			zap.String("course_id", event.SemesterCourseID.String()),
		)
		return err // Requeue for retry
	}

	log.Info("course.semester.created event processed successfully",
		zap.String("event_id", eventID),
	)

	return nil
}

// handleStudentCreated handles student.created events
func (c *EventConsumer) handleStudentCreated(ctx context.Context, msgBody []byte, eventID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentCreated"),
	)

	// Parse wrapped event format (from student-service)
	var wrapped wrappedEvent
	if err := json.Unmarshal(msgBody, &wrapped); err != nil {
		log.Error("failed to unmarshal student.created event wrapper",
			zap.Error(err),
		)
		return err
	}

	// Parse data field
	var data studentEventData
	if err := json.Unmarshal(wrapped.Data, &data); err != nil {
		log.Error("failed to unmarshal student.created event data",
			zap.Error(err),
		)
		return err
	}

	// Parse event ID
	parsedEventID, err := uuid.Parse(wrapped.EventID)
	if err != nil {
		log.Error("failed to parse event_id", zap.Error(err))
		return err
	}

	// Parse student ID
	studentID, err := uuid.Parse(data.ID)
	if err != nil {
		log.Error("failed to parse student_id", zap.Error(err), zap.String("id", data.ID))
		return err
	}

	// Parse advisor ID if present
	var advisorID uuid.UUID
	if data.AdvisorID != nil && *data.AdvisorID != "" {
		advisorID, _ = uuid.Parse(*data.AdvisorID)
	}

	// Construct the event
	event := dto.StudentCreatedEvent{
		BaseEvent: dto.BaseEvent{
			EventID:   parsedEventID,
			EventType: wrapped.EventType,
		},
		StudentID:     studentID,
		StudentNumber: data.StudentNumber,
		Email:         data.Email,
		FirstName:     data.FirstName,
		LastName:      data.LastName,
		Department:    data.Department,
		ClassLevel:    data.ClassLevel,
		AdvisorID:     advisorID,
		Status:        data.Status,
	}

	log.Info("processing student.created event",
		zap.String("event_id", eventID),
		zap.String("student_id", event.StudentID.String()),
	)

	err = c.eventService.HandleStudentCreated(ctx, event)
	if err != nil {
		log.Error("failed to process student.created event",
			zap.Error(err),
			zap.String("student_id", event.StudentID.String()),
		)
		return err
	}

	log.Info("student.created event processed successfully",
		zap.String("event_id", eventID),
	)

	return nil
}

// handleStudentUpdated handles student.updated events
func (c *EventConsumer) handleStudentUpdated(ctx context.Context, msgBody []byte, eventID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentUpdated"),
	)

	// Parse wrapped event format (from student-service)
	var wrapped wrappedEvent
	if err := json.Unmarshal(msgBody, &wrapped); err != nil {
		log.Error("failed to unmarshal student.updated event wrapper",
			zap.Error(err),
		)
		return err
	}

	// Parse data field
	var data studentEventData
	if err := json.Unmarshal(wrapped.Data, &data); err != nil {
		log.Error("failed to unmarshal student.updated event data",
			zap.Error(err),
		)
		return err
	}

	// Parse event ID
	parsedEventID, err := uuid.Parse(wrapped.EventID)
	if err != nil {
		log.Error("failed to parse event_id", zap.Error(err))
		return err
	}

	// Parse student ID
	studentID, err := uuid.Parse(data.ID)
	if err != nil {
		log.Error("failed to parse student_id", zap.Error(err), zap.String("id", data.ID))
		return err
	}

	// Parse advisor ID if present
	var advisorID *uuid.UUID
	if data.AdvisorID != nil && *data.AdvisorID != "" {
		parsed, _ := uuid.Parse(*data.AdvisorID)
		advisorID = &parsed
	}

	// Construct the event
	event := dto.StudentUpdatedEvent{
		BaseEvent: dto.BaseEvent{
			EventID:   parsedEventID,
			EventType: wrapped.EventType,
		},
		StudentID:     studentID,
		StudentNumber: data.StudentNumber,
		Email:         data.Email,
		FirstName:     data.FirstName,
		LastName:      data.LastName,
		Department:    data.Department,
		ClassLevel:    data.ClassLevel,
		AdvisorID:     advisorID,
		Status:        data.Status,
	}

	log.Info("processing student.updated event",
		zap.String("event_id", eventID),
		zap.String("student_id", event.StudentID.String()),
	)

	err = c.eventService.HandleStudentUpdated(ctx, event)
	if err != nil {
		log.Error("failed to process student.updated event",
			zap.Error(err),
			zap.String("student_id", event.StudentID.String()),
		)
		return err
	}

	log.Info("student.updated event processed successfully",
		zap.String("event_id", eventID),
	)

	return nil
}

// handleStudentDeactivated handles student.deleted events
func (c *EventConsumer) handleStudentDeactivated(ctx context.Context, msgBody []byte, eventID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentDeactivated"),
	)

	// Parse wrapped event format (from student-service)
	var wrapped wrappedEvent
	if err := json.Unmarshal(msgBody, &wrapped); err != nil {
		log.Error("failed to unmarshal student.deleted event wrapper",
			zap.Error(err),
		)
		return err
	}

	// Parse data field
	var data studentEventData
	if err := json.Unmarshal(wrapped.Data, &data); err != nil {
		log.Error("failed to unmarshal student.deleted event data",
			zap.Error(err),
		)
		return err
	}

	// Parse event ID
	parsedEventID, err := uuid.Parse(wrapped.EventID)
	if err != nil {
		log.Error("failed to parse event_id", zap.Error(err))
		return err
	}

	// Parse student ID
	studentID, err := uuid.Parse(data.ID)
	if err != nil {
		log.Error("failed to parse student_id", zap.Error(err), zap.String("id", data.ID))
		return err
	}

	// Construct the event
	event := dto.StudentDeactivatedEvent{
		BaseEvent: dto.BaseEvent{
			EventID:   parsedEventID,
			EventType: wrapped.EventType,
		},
		StudentID: studentID,
	}

	log.Info("processing student.deleted event",
		zap.String("event_id", eventID),
		zap.String("student_id", event.StudentID.String()),
	)

	err = c.eventService.HandleStudentDeactivated(ctx, event)
	if err != nil {
		log.Error("failed to process student.deleted event",
			zap.Error(err),
			zap.String("student_id", event.StudentID.String()),
		)
		return err
	}

	log.Info("student.deleted event processed successfully",
		zap.String("event_id", eventID),
	)

	return nil
}

// handleGradeStudentPrerequisitePassed handles grade.student.prerequisite.passed events
func (c *EventConsumer) handleGradeStudentPrerequisitePassed(ctx context.Context, msgBody []byte, eventID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleGradeStudentPrerequisitePassed"),
	)

	var event dto.GradeStudentPrerequisitePassedEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		log.Error("failed to unmarshal grade.student.prerequisite.passed event",
			zap.Error(err),
		)
		return err
	}

	log.Info("processing grade.student.prerequisite.passed event",
		zap.String("event_id", eventID),
		zap.String("student_id", event.StudentID.String()),
		zap.String("course_code", event.CourseCode),
	)

	err := c.eventService.HandleGradeStudentPrerequisitePassed(ctx, event)
	if err != nil {
		log.Error("failed to process grade.student.prerequisite.passed event",
			zap.Error(err),
			zap.String("student_id", event.StudentID.String()),
		)
		return err
	}

	log.Info("grade.student.prerequisite.passed event processed successfully",
		zap.String("event_id", eventID),
	)

	return nil
}
