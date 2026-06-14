package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"go.uber.org/zap"
)

type OutboxWorker struct {
	outboxRepo *repository.OutboxRepository
	publisher  *rabbitmq.Publisher
}

func NewOutboxWorker(outboxRepo *repository.OutboxRepository, publisher *rabbitmq.Publisher) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log := logger.WithContextAndFields(ctx, zap.String("worker", "OutboxWorker"))
	log.Info("Outbox worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info("Outbox worker stopped")
			return
		case <-ticker.C:
			w.processEvents(ctx)
		}
	}
}

func (w *OutboxWorker) processEvents(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("method", "processEvents"),
	)

	events, err := w.outboxRepo.GetPendingOutboxEvents(ctx, 100)
	if err != nil {
		log.Error("failed to get pending outbox events", zap.Error(err))
		return
	}

	for _, event := range events {
		var payload any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			log.Error("failed to unmarshal payload", zap.Error(err), zap.String("event_id", event.ID.String()))
			w.outboxRepo.MarkOutboxEventFailed(ctx, event.ID, err.Error())
			continue
		}

		if err := w.publisher.Publish(ctx, "attendance.events", event.RoutingKey, payload); err != nil {
			log.Warn("failed to publish event, will retry on next poll", zap.Error(err), zap.String("routing_key", event.RoutingKey))
		} else {
			w.outboxRepo.MarkOutboxEventProcessed(ctx, event.ID)
			log.Debug("event published", zap.String("routing_key", event.RoutingKey))
		}
	}
}
