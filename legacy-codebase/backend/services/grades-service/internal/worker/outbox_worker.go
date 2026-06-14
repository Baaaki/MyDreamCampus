package worker

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/grades-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OutboxWorker struct {
	outboxRepo     *repository.OutboxRepository
	publisher      *rabbitmq.Publisher
	pollInterval   time.Duration
	batchSize      int32
}

func NewOutboxWorker(
	outboxRepo *repository.OutboxRepository,
	publisher *rabbitmq.Publisher,
	pollInterval time.Duration,
	batchSize int32,
) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo:   outboxRepo,
		publisher:    publisher,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "OutboxWorker"))
	log.Info("starting outbox worker",
		zap.Duration("poll_interval", w.pollInterval),
		zap.Int32("batch_size", w.batchSize),
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info("outbox worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("method", "processBatch"),
	)

	// Get pending events
	events, err := w.outboxRepo.GetPendingOutboxEvents(ctx, w.batchSize)
	if err != nil {
		log.Error("failed to get pending outbox events", zap.Error(err))
		return
	}

	if len(events) == 0 {
		return
	}

	log.Info("processing outbox events", zap.Int("count", len(events)))

	for _, event := range events {
		if err := w.processEvent(ctx, event.ID, event.EventType, event.RoutingKey, event.Payload); err != nil {
			log.Warn("failed to process outbox event, will retry on next poll",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
				zap.String("event_type", event.EventType),
			)
			continue
		}

		// Mark as processed
		if err := w.outboxRepo.MarkOutboxEventProcessed(ctx, event.ID); err != nil {
			log.Error("failed to mark outbox event as processed",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
			)
		}
	}
}

func (w *OutboxWorker) processEvent(ctx context.Context, eventID uuid.UUID, eventType string, routingKey string, payload []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("method", "processEvent"),
	)

	log.Debug("publishing outbox event",
		zap.String("event_id", eventID.String()),
		zap.String("event_type", eventType),
		zap.String("routing_key", routingKey),
	)

	// Determine exchange based on routing key
	exchange := "grade.events"

	// Publish to RabbitMQ
	if err := w.publisher.Publish(ctx, exchange, routingKey, payload); err != nil {
		return err
	}

	log.Info("outbox event published",
		zap.String("event_id", eventID.String()),
		zap.String("event_type", eventType),
	)

	return nil
}
