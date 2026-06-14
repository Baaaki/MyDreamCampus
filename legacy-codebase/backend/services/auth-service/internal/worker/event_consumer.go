package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	"github.com/baaaki/mydreamcampus/auth-service/internal/service"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"go.uber.org/zap"
)

type EventConsumer struct {
	consumer     *rabbitmq.Consumer
	eventService *service.EventService
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	eventService *service.EventService,
) *EventConsumer {
	return &EventConsumer{
		consumer:     consumer,
		eventService: eventService,
	}
}

// Start begins consuming events from RabbitMQ
func (w *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EventConsumer"))
	log.Info("starting event consumer")

	// Declare queue using shared events constants
	err := w.consumer.DeclareQueue(events.QueueAuthStaffEvents)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Create message handler
	handler := func(body []byte) error {
		// Parse base event to get routing key
		var baseEvent dto.BaseEvent
		if err := json.Unmarshal(body, &baseEvent); err != nil {
			return fmt.Errorf("failed to unmarshal base event: %w", err)
		}

		log.Info("received event",
			zap.String("event_type", baseEvent.EventType),
			zap.String("event_id", baseEvent.EventID),
		)

		// Route to appropriate handler using shared events constants
		switch baseEvent.EventType {
		case events.EventStudentCreated:
			return w.handleStudentCreated(ctx, body)
		case events.EventStaffCreated:
			return w.handleStaffCreated(ctx, body)
		case events.EventStudentUpdated, events.EventStaffUpdated:
			return w.handleUserUpdated(ctx, body)
		case events.EventStudentDeactivated, events.EventStaffDeactivated:
			return w.handleUserDeactivated(ctx, body)
		default:
			log.Warn("unknown event type",
				zap.String("event_type", baseEvent.EventType),
			)
			return nil // Ack unknown events to avoid infinite loop
		}
	}

	// Start consuming
	err = w.consumer.Consume(events.QueueAuthStaffEvents, handler)
	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	log.Info("event consumer started successfully")
	return nil
}


// handleStudentCreated processes student.created event
func (w *EventConsumer) handleStudentCreated(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStudentCreated"),
	)

	var event dto.StudentCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal student.created event",
			zap.Error(err),
		)
		return err
	}

	return w.eventService.HandleStudentCreated(ctx, event)
}

// handleStaffCreated processes staff.created event
func (w *EventConsumer) handleStaffCreated(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStaffCreated"),
	)

	var event dto.StaffCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal staff.created event",
			zap.Error(err),
		)
		return err
	}

	return w.eventService.HandleStaffCreated(ctx, event)
}

// handleUserUpdated processes student.updated and staff.updated events
func (w *EventConsumer) handleUserUpdated(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleUserUpdated"),
	)

	var event dto.UserUpdatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal user.updated event",
			zap.Error(err),
		)
		return err
	}

	return w.eventService.HandleUserUpdated(ctx, event)
}

// handleUserDeactivated processes student.deactivated and staff.deactivated events
func (w *EventConsumer) handleUserDeactivated(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleUserDeactivated"),
	)

	var event dto.UserDeactivatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal user.deactivated event",
			zap.Error(err),
		)
		return err
	}

	return w.eventService.HandleUserDeactivated(ctx, event)
}
