package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/repository"
	"go.uber.org/zap"
)

type OutboxWorker struct {
	outboxRepo *repository.OutboxRepository
	publisher  *rabbitmq.Publisher
	interval   time.Duration
	batchSize  int32
}

func NewOutboxWorker(
	outboxRepo *repository.OutboxRepository,
	publisher *rabbitmq.Publisher,
	interval time.Duration,
	batchSize int32,
) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		interval:   interval,
		batchSize:  batchSize,
	}
}

// Start begins the outbox worker polling loop
func (w *OutboxWorker) Start(ctx context.Context) {
	logger.Info("starting outbox worker",
		zap.Duration("interval", w.interval),
		zap.Int32("batch_size", w.batchSize),
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start
	w.processEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping outbox worker")
			return
		case <-ticker.C:
			w.processEvents(ctx)
		}
	}
}

// processEvents retrieves and publishes pending events
func (w *OutboxWorker) processEvents(ctx context.Context) {
	events, err := w.outboxRepo.GetPendingEvents(ctx, w.batchSize)
	if err != nil {
		logger.Error("failed to get pending events",
			zap.Error(err),
		)
		return
	}

	if len(events) == 0 {
		// Silently return when no events (avoid log spam)
		return
	}

	logger.Info("processing outbox events",
		zap.Int("count", len(events)),
	)

	successCount := 0
	failCount := 0

	for _, event := range events {
		eventID := utils.PgtypeToUUIDString(event.ID)

		// Parse payload to map for publishing
		var payload map[string]interface{}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			logger.Error("failed to unmarshal event payload",
				zap.Error(err),
				zap.String("event_id", eventID),
			)
			w.outboxRepo.MarkEventFailed(ctx, utils.PgtypeToUUID(event.ID), err.Error())
			failCount++
			continue
		}

		// Publish to RabbitMQ
		err := w.publisher.Publish(ctx, "student.events", event.RoutingKey, payload)
		if err != nil {
			logger.Error("failed to publish event",
				zap.Error(err),
				zap.String("event_id", eventID),
				zap.String("event_type", event.EventType),
				zap.String("routing_key", event.RoutingKey),
			)
			w.outboxRepo.MarkEventFailed(ctx, utils.PgtypeToUUID(event.ID), err.Error())
			failCount++
			continue
		}

		// Mark event as processed
		err = w.outboxRepo.MarkEventProcessed(ctx, utils.PgtypeToUUID(event.ID))
		if err != nil {
			logger.Error("failed to mark event as processed",
				zap.Error(err),
				zap.String("event_id", eventID),
			)
			failCount++
			continue
		}

		logger.Info("event published successfully",
			zap.String("event_id", eventID),
			zap.String("event_type", event.EventType),
		)
		successCount++
	}

	logger.Info("outbox processing completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Int("total", len(events)),
	)

	// Process failed events that can be retried
	w.processFailedEvents(ctx)
}

// processFailedEvents retries failed events
func (w *OutboxWorker) processFailedEvents(ctx context.Context) {
	events, err := w.outboxRepo.GetFailedEvents(ctx, w.batchSize)
	if err != nil {
		logger.Error("failed to get failed events",
			zap.Error(err),
		)
		return
	}

	if len(events) == 0 {
		return
	}

	logger.Info("retrying failed events",
		zap.Int("count", len(events)),
	)

	for _, event := range events {
		eventID := utils.PgtypeToUUIDString(event.ID)

		// Check if max retries exceeded
		if event.RetryCount.Int16 >= event.MaxRetries.Int16 {
			logger.Warn("max retries exceeded for event",
				zap.String("event_id", eventID),
				zap.Int16("retry_count", event.RetryCount.Int16),
			)
			continue
		}

		// Reset to pending for retry
		err := w.outboxRepo.ResetFailedEvent(ctx, utils.PgtypeToUUID(event.ID))
		if err != nil {
			logger.Error("failed to reset failed event",
				zap.Error(err),
				zap.String("event_id", eventID),
			)
			continue
		}

		logger.Info("event reset for retry",
			zap.String("event_id", eventID),
			zap.Int16("retry_count", event.RetryCount.Int16),
		)
	}
}
