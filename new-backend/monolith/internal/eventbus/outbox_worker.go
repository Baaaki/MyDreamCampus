package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"go.uber.org/zap"
)

// OutboxWorker polls one module's outbox table and relays events to RabbitMQ.
// Plan section 5.5.2 keeps the existing per-module shape — each module spins
// up its own worker goroutine in main.go pointing at its own OutboxStore +
// exchange. The worker code itself is shared.
type OutboxWorker struct {
	module    string
	exchange  string
	store     OutboxStore
	publisher *rabbitmq.Publisher
	interval  time.Duration
	batchSize int32
}

func NewOutboxWorker(
	module, exchange string,
	store OutboxStore,
	publisher *rabbitmq.Publisher,
	interval time.Duration,
	batchSize int32,
) *OutboxWorker {
	return &OutboxWorker{
		module:    module,
		exchange:  exchange,
		store:     store,
		publisher: publisher,
		interval:  interval,
		batchSize: batchSize,
	}
}

// Start blocks on a polling ticker until ctx is cancelled.
func (w *OutboxWorker) Start(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("module", w.module),
	)
	log.Info("starting outbox worker",
		zap.String("exchange", w.exchange),
		zap.Duration("interval", w.interval),
		zap.Int32("batch_size", w.batchSize),
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

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

func (w *OutboxWorker) processEvents(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("module", w.module),
		zap.String("method", "processEvents"),
	)

	pending, err := w.store.GetPending(ctx, w.batchSize)
	if err != nil {
		log.Error("failed to get pending events", zap.Error(err))
		return
	}
	if len(pending) == 0 {
		w.processFailedEvents(ctx)
		return
	}

	log.Info("processing outbox events", zap.Int("count", len(pending)))

	var success, failed int
	for _, ev := range pending {
		var payload map[string]any
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			log.Error("malformed payload",
				zap.Error(err),
				zap.String("event_id", ev.ID.String()),
			)
			_ = w.store.MarkFailed(ctx, ev.ID, err.Error())
			failed++
			continue
		}

		// Envelope shape required by consumers (see plan section 5.7).
		message := map[string]any{
			"event_id":   ev.ID.String(),
			"event_type": ev.EventType,
			"timestamp":  ev.CreatedAt,
			"data":       payload,
		}

		if err := w.publisher.Publish(ctx, w.exchange, ev.RoutingKey, message); err != nil {
			log.Error("publish failed",
				zap.Error(err),
				zap.String("event_id", ev.ID.String()),
				zap.String("event_type", ev.EventType),
				zap.String("routing_key", ev.RoutingKey),
			)
			_ = w.store.MarkFailed(ctx, ev.ID, err.Error())
			failed++
			continue
		}

		if err := w.store.MarkProcessed(ctx, ev.ID); err != nil {
			log.Error("mark processed failed",
				zap.Error(err),
				zap.String("event_id", ev.ID.String()),
			)
			failed++
			continue
		}
		success++
	}

	log.Info("outbox cycle complete",
		zap.Int("success", success),
		zap.Int("failed", failed),
		zap.Int("total", len(pending)),
	)

	w.processFailedEvents(ctx)
}

func (w *OutboxWorker) processFailedEvents(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "OutboxWorker"),
		zap.String("module", w.module),
		zap.String("method", "processFailedEvents"),
	)

	failed, err := w.store.GetFailed(ctx, w.batchSize)
	if err != nil {
		log.Error("failed to get failed events", zap.Error(err))
		return
	}
	if len(failed) == 0 {
		return
	}

	log.Info("retrying failed events", zap.Int("count", len(failed)))

	for _, ev := range failed {
		if ev.RetryCount >= ev.MaxRetries {
			log.Warn("max retries exceeded",
				zap.String("event_id", ev.ID.String()),
				zap.Int16("retry_count", ev.RetryCount),
			)
			continue
		}
		if err := w.store.Reset(ctx, ev.ID); err != nil {
			log.Error("reset failed",
				zap.Error(err),
				zap.String("event_id", ev.ID.String()),
			)
			continue
		}
		log.Info("event reset for retry",
			zap.String("event_id", ev.ID.String()),
			zap.Int16("retry_count", ev.RetryCount),
		)
	}
}
