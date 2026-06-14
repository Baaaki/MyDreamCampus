package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/meal-service/internal/db"
	"github.com/baaaki/mydreamcampus/meal-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// OutboxWorker polls and publishes pending outbox events
type OutboxWorker struct {
	outboxRepo   *repository.OutboxRepository
	publisher    *rabbitmq.Publisher
	pollInterval time.Duration
	batchSize    int32
	maxRetries   int
	logger       *zap.Logger
	stopChan     chan struct{}
}

func NewOutboxWorker(
	outboxRepo *repository.OutboxRepository,
	publisher *rabbitmq.Publisher,
	pollIntervalSeconds int,
	batchSize int,
	maxRetries int,
	logger *zap.Logger,
) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo:   outboxRepo,
		publisher:    publisher,
		pollInterval: time.Duration(pollIntervalSeconds) * time.Second,
		batchSize:    int32(batchSize),
		maxRetries:   maxRetries,
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the outbox polling job
func (w *OutboxWorker) Start(ctx context.Context) {
	w.logger.Info("starting outbox worker", zap.Duration("poll_interval", w.pollInterval))

	go w.runPollingJob(ctx)
}

// Stop stops the outbox worker
func (w *OutboxWorker) Stop() {
	w.logger.Info("stopping outbox worker")
	close(w.stopChan)
}

func (w *OutboxWorker) runPollingJob(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.processPendingEvents(ctx); err != nil {
				w.logger.Error("failed to process pending events", zap.Error(err))
			}
		case <-w.stopChan:
			w.logger.Info("outbox polling job stopped")
			return
		case <-ctx.Done():
			w.logger.Info("outbox polling job context cancelled")
			return
		}
	}
}

func (w *OutboxWorker) processPendingEvents(ctx context.Context) error {
	// Get pending events
	events, err := w.outboxRepo.GetPendingOutboxEvents(ctx, w.batchSize)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	w.logger.Debug("processing pending outbox events", zap.Int("count", len(events)))

	for _, event := range events {
		if err := w.publishEvent(ctx, event); err != nil {
			w.logger.Error("failed to publish event",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
				zap.String("event_type", event.EventType),
			)
		}
	}

	return nil
}

func (w *OutboxWorker) publishEvent(ctx context.Context, event db.OutboxEvent) error {
	// Unmarshal payload
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		// Invalid payload, mark as failed
		w.outboxRepo.MarkOutboxEventFailed(ctx, utils.PgtypeToUUID(event.ID), fmt.Sprintf("invalid payload: %v", err))
		return err
	}

	// Publish to RabbitMQ (routing key is the event type)
	err := w.publisher.Publish(ctx, event.EventType, event.EventType, payload)
	if err != nil {
		// Publishing failed, handle retry logic
		newRetryCount := event.RetryCount + 1

		if newRetryCount >= int16(w.maxRetries) {
			// Max retries exceeded, mark as failed
			w.outboxRepo.MarkOutboxEventFailed(ctx, utils.PgtypeToUUID(event.ID), fmt.Sprintf("max retries exceeded: %v", err))
		} else {
			// Schedule retry with exponential backoff
			nextRetryAt := clock.Now().Add(time.Duration(1<<newRetryCount) * time.Minute)
			w.outboxRepo.UpdateOutboxEventRetry(ctx, utils.PgtypeToUUID(event.ID), pgtype.Timestamptz{
				Time:  nextRetryAt,
				Valid: true,
			}, err.Error())
		}

		return err
	}

	// Successfully published, mark as published
	err = w.outboxRepo.MarkOutboxEventPublished(ctx, utils.PgtypeToUUID(event.ID))
	if err != nil {
		w.logger.Error("failed to mark event as published",
			zap.Error(err),
			zap.String("event_id", event.ID.String()),
		)
		return err
	}

	w.logger.Debug("event published successfully",
		zap.String("event_id", event.ID.String()),
		zap.String("event_type", event.EventType),
	)

	return nil
}
