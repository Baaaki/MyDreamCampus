package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
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
	log := logger.WithContextAndFields(ctx, zap.String("worker", "OutboxWorker"))
	log.Info("starting outbox worker",
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
			log.Info("stopping outbox worker")
			return
		case <-ticker.C:
			w.processEvents(ctx)
		}
	}
}

// processEvents retrieves and publishes pending events
func (w *OutboxWorker) processEvents(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("method", "processEvents"),
	)

	events, err := w.outboxRepo.GetPendingEvents(ctx, w.batchSize)
	if err != nil {
		log.Error("failed to get pending events",
			zap.Error(err),
		)
		return
	}

	if len(events) == 0 {
		// Silently return when no events (avoid log spam)
		return
	}

	log.Info("processing outbox events",
		zap.Int("count", len(events)),
	)

	successCount := 0
	failCount := 0

	for _, event := range events {
		// Convert pgtype.UUID to uuid.UUID
		eventUUID := uuid.UUID(event.ID.Bytes)
		eventIDStr := utils.PgtypeToUUIDString(event.ID)

		// Parse payload to map for publishing
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Error("failed to unmarshal event payload",
				zap.Error(err),
				zap.String("event_id", eventIDStr),
			)
			failCount++
			continue
		}

		// Publish to RabbitMQ
		err := w.publisher.Publish(ctx, "course.events", event.RoutingKey, payload)
		if err != nil {
			log.Error("failed to publish event",
				zap.Error(err),
				zap.String("event_id", eventIDStr),
				zap.String("event_type", event.EventType),
				zap.String("routing_key", event.RoutingKey),
			)
			failCount++
			continue
		}

		// Mark event as processed
		err = w.outboxRepo.MarkEventProcessed(ctx, eventUUID)
		if err != nil {
			log.Error("failed to mark event as processed",
				zap.Error(err),
				zap.String("event_id", eventIDStr),
			)
			failCount++
			continue
		}

		log.Info("event published successfully",
			zap.String("event_id", eventIDStr),
			zap.String("event_type", event.EventType),
		)
		successCount++
	}

	log.Info("outbox processing completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Int("total", len(events)),
	)
}
