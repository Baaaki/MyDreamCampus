package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/baaaki/mydreamcampus/student-service/internal/repository"
	"go.uber.org/zap"
)

type EventConsumer struct {
	consumer              *rabbitmq.Consumer
	studentRepo           *repository.StudentRepository
	processedEventsRepo   *repository.ProcessedEventsRepository
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	studentRepo *repository.StudentRepository,
	processedEventsRepo *repository.ProcessedEventsRepository,
) *EventConsumer {
	return &EventConsumer{
		consumer:            consumer,
		studentRepo:         studentRepo,
		processedEventsRepo: processedEventsRepo,
	}
}

// Start begins consuming events from RabbitMQ
func (c *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EventConsumer"))
	log.Info("starting event consumer")

	// Consume staff events using shared events constants
	err := c.consumer.Consume(events.QueueStudentStaffEvents, func(msg []byte) error {
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
	var genericEvent dto.Event
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
			zap.String("event_id", genericEvent.EventID),
		)
		return err // Requeue for retry
	}

	if processed {
		log.Info("event already processed, skipping",
			zap.String("event_id", genericEvent.EventID),
			zap.String("event_type", genericEvent.EventType),
		)
		return nil
	}

	// Route to appropriate handler using shared events constants
	switch genericEvent.EventType {
	case events.EventStaffDeactivated:
		return c.handleStaffDeactivated(ctx, msgBody, genericEvent.EventID)
	default:
		log.Warn("unknown event type",
			zap.String("event_type", genericEvent.EventType),
		)
		return nil // Acknowledge unknown events to avoid DLQ
	}
}

// handleStaffDeactivated handles staff.deactivated events
func (c *EventConsumer) handleStaffDeactivated(ctx context.Context, msgBody []byte, eventID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EventConsumer"),
		zap.String("method", "handleStaffDeactivated"),
	)

	var event dto.StaffDeactivatedEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		log.Error("failed to unmarshal staff.deactivated event",
			zap.Error(err),
		)
		return err
	}

	log.Info("processing staff.deactivated event",
		zap.String("event_id", eventID),
		zap.String("staff_id", event.Data.StaffID.String()),
	)

	// Unassign advisor and mark event as processed in single transaction (atomicity)
	err := c.studentRepo.UnassignAdvisorByStaffIDWithEventMarking(ctx, event.Data.StaffID, eventID, event.EventType)
	if err != nil {
		log.Error("failed to process staff.deactivated event",
			zap.Error(err),
			zap.String("staff_id", event.Data.StaffID.String()),
		)
		return err // Requeue for retry
	}

	log.Info("staff.deactivated event processed successfully",
		zap.String("event_id", eventID),
		zap.String("staff_id", event.Data.StaffID.String()),
	)

	return nil
}

// SetupStaffEventsQueue sets up the queue for staff events
func SetupStaffEventsQueue(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare queue using shared events constants
	_, err := channel.QueueDeclare(
		events.QueueStudentStaffEvents, // queue name
		true,                            // durable
		false,                           // delete when unused
		false,                           // exclusive
		false,                           // no-wait
		nil,                             // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to staff.events exchange with staff.deleted routing key
	err = channel.QueueBind(
		events.QueueStudentStaffEvents, // queue name
		events.RoutingKeyStaffDeactivated, // routing key
		"staff.events",                  // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	logger.Info("staff events queue setup completed",
		zap.String("queue", events.QueueStudentStaffEvents),
		zap.String("routing_key", events.RoutingKeyStaffDeactivated),
	)

	return nil
}
