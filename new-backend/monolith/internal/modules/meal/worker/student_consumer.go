package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StudentEventConsumer handles student events from RabbitMQ
type StudentEventConsumer struct {
	studentCacheRepo     *repository.StudentCacheRepository
	processedEventsRepo  *repository.ProcessedEventsRepository
	logger               *zap.Logger
}

func NewStudentEventConsumer(
	studentCacheRepo *repository.StudentCacheRepository,
	processedEventsRepo *repository.ProcessedEventsRepository,
	logger *zap.Logger,
) *StudentEventConsumer {
	return &StudentEventConsumer{
		studentCacheRepo:    studentCacheRepo,
		processedEventsRepo: processedEventsRepo,
		logger:              logger,
	}
}

// HandleStudentCreated handles student.created event
func (c *StudentEventConsumer) HandleStudentCreated(ctx context.Context, body []byte) error {
	var event dto.StudentCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("failed to unmarshal student.created event", zap.Error(err))
		return err
	}

	// Check if event already processed (idempotency)
	eventID, _ := uuid.Parse(event.EventID)
	processed, err := c.processedEventsRepo.IsEventProcessed(ctx, eventID)
	if err != nil {
		c.logger.Error("failed to check if event is processed", zap.Error(err))
		return err
	}

	if processed {
		c.logger.Debug("event already processed, skipping", zap.String("event_id", event.EventID))
		return nil
	}

	// Upsert student cache
	studentID, _ := uuid.Parse(event.Data.ID)
	_, err = c.studentCacheRepo.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID:            utils.UUIDToPgtype(studentID),
		StudentNumber: event.Data.StudentNumber,
		FirstName:     event.Data.FirstName,
		LastName:      event.Data.LastName,
		IsActive:      true,
	})
	if err != nil {
		c.logger.Error("failed to upsert student cache", zap.Error(err), zap.String("student_id", event.Data.ID))
		return err
	}

	// Mark event as processed
	if err := c.processedEventsRepo.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: event.EventType,
	}); err != nil {
		c.logger.Error("failed to mark event as processed", zap.Error(err))
		return err
	}

	c.logger.Info("student.created event processed", zap.String("student_id", event.Data.ID))
	return nil
}

// HandleStudentUpdated handles student.updated event
func (c *StudentEventConsumer) HandleStudentUpdated(ctx context.Context, body []byte) error {
	var event dto.StudentUpdatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("failed to unmarshal student.updated event", zap.Error(err))
		return err
	}

	// Check if event already processed
	eventID, _ := uuid.Parse(event.EventID)
	processed, err := c.processedEventsRepo.IsEventProcessed(ctx, eventID)
	if err != nil {
		return err
	}

	if processed {
		c.logger.Debug("event already processed, skipping", zap.String("event_id", event.EventID))
		return nil
	}

	// Upsert student cache (for out-of-order tolerance)
	studentID, _ := uuid.Parse(event.Data.ID)
	_, err = c.studentCacheRepo.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID:            utils.UUIDToPgtype(studentID),
		StudentNumber: event.Data.StudentNumber,
		FirstName:     event.Data.FirstName,
		LastName:      event.Data.LastName,
		IsActive:      true,
	})
	if err != nil {
		c.logger.Error("failed to upsert student cache", zap.Error(err))
		return err
	}

	// Mark event as processed
	if err := c.processedEventsRepo.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: event.EventType,
	}); err != nil {
		return err
	}

	c.logger.Info("student.updated event processed", zap.String("student_id", event.Data.ID))
	return nil
}

// HandleStudentDeactivated handles student.deactivated event
func (c *StudentEventConsumer) HandleStudentDeactivated(ctx context.Context, body []byte) error {
	var event dto.StudentDeactivatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("failed to unmarshal student.deactivated event", zap.Error(err))
		return err
	}

	// Check if event already processed
	eventID, _ := uuid.Parse(event.EventID)
	processed, err := c.processedEventsRepo.IsEventProcessed(ctx, eventID)
	if err != nil {
		return err
	}

	if processed {
		c.logger.Debug("event already processed, skipping", zap.String("event_id", event.EventID))
		return nil
	}

	// Delete student cache (CASCADE will delete all reservations)
	studentID, _ := uuid.Parse(event.Data.ID)
	if err := c.studentCacheRepo.DeleteStudentCache(ctx, studentID); err != nil {
		c.logger.Error("failed to delete student cache", zap.Error(err))
		return err
	}

	// Mark event as processed
	if err := c.processedEventsRepo.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: event.EventType,
	}); err != nil {
		return err
	}

	c.logger.Info("student.deactivated event processed", zap.String("student_id", event.Data.ID))
	return nil
}
