package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

// settleTimeout bounds one bookkeeping write about an event that has already
// been handled.
const SettleTimeout = 10 * time.Second
const settleTimeout = SettleTimeout

// Publisher is the part of rabbitmq.Publisher the relay needs. Publish
// returns nil only once the broker has confirmed the message, which is what
// lets the worker treat a nil as "safe to mark sent". It is an interface so a
// test can stand a fake broker in for the real one.
type Publisher interface {
	Publish(ctx context.Context, exchange, routingKey string, payload any) error
}

// OutboxWorker polls one module's outbox table and relays events to RabbitMQ.
// Each module spins
// up its own worker goroutine in main.go pointing at its own OutboxStore +
// exchange. The worker code itself is shared.
type OutboxWorker struct {
	module    string
	exchange  string
	store     OutboxStore
	publisher Publisher
	interval  time.Duration
	batchSize int32
}

func NewOutboxWorker(
	module, exchange string,
	store OutboxStore,
	publisher Publisher,
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

	for {
		if ctx.Err() != nil {
			return
		}

		var batchCount int
		var stoppedEarly bool
		err := w.store.ClaimPending(ctx, w.batchSize, func(ctx context.Context, pending []OutboxEvent, tx ClaimTx) error {
			batchCount = len(pending)
			if len(pending) == 0 {
				return nil
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
					if txErr := tx.MarkFailed(ctx, ev.ID, err.Error()); txErr != nil {
						stoppedEarly = true
					}
					failed++
					continue
				}

				// Envelope shape shared with all consumers (notification service +
				// module workers) — changing a key here is a breaking contract change.
				// correlation_id is omitted when absent, so consumers that predate it
				// keep parsing the envelope unchanged.
				message := map[string]any{
					"event_id":   ev.ID.String(),
					"event_type": ev.EventType,
					"timestamp":  ev.CreatedAt,
					"data":       payload,
				}
				if ev.CorrelationID != "" {
					message["correlation_id"] = ev.CorrelationID
				}

				if err := w.publisher.Publish(ctx, w.exchange, ev.RoutingKey, message); err != nil {
					// The broker refused, not the event: leaving the row pending with
					// its retries intact is what lets an outage of any length end
					// without losing anything. Counting it as a failed attempt parked
					// events for good after ~25 s. The rest of the batch would each sit
					// out their own confirm timeout, so the batch ends here.
					log.Error("publish failed, leaving the rest of the batch pending",
						zap.Error(err),
						zap.String("event_id", ev.ID.String()),
						zap.String("event_type", ev.EventType),
						zap.String("routing_key", ev.RoutingKey),
					)
					stoppedEarly = true
					break
				}

				if err := tx.MarkProcessed(ctx, ev.ID); err != nil {
					log.Error("mark processed failed",
						zap.Error(err),
						zap.String("event_id", ev.ID.String()),
					)
					failed++
					stoppedEarly = true
					continue
				}
				success++
			}

			log.Info("outbox cycle complete",
				zap.Int("success", success),
				zap.Int("failed", failed),
				zap.Int("total", len(pending)),
			)

			return nil
		})

		if err != nil {
			log.Error("failed to claim pending events", zap.Error(err))
			return
		}

		if stoppedEarly || batchCount < int(w.batchSize) {
			w.processFailedEvents(ctx)
			return
		}
	}
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
