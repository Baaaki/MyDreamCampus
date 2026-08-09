package eventbus

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

// RetentionStore is the cleanup half of a service's event tables. A service
// that publishes but never consumes has no processed_events table and
// returns 0 from DeleteProcessedEvents — the worker does not care which.
type RetentionStore interface {
	// DeleteRelayedOutboxEvents removes outbox rows that were already handed
	// to the broker before the cutoff.
	DeleteRelayedOutboxEvents(ctx context.Context, before time.Time) (int64, error)
	// DeleteProcessedEvents removes dedup-ledger rows older than the cutoff.
	DeleteProcessedEvents(ctx context.Context, before time.Time) (int64, error)
}

const (
	defaultRetentionDays  = 30
	defaultIntervalHours  = 1
	retentionQueryTimeout = 30 * time.Second
)

// RetentionWorker prunes one service's outbox_events and processed_events.
// Neither table was ever cleaned: rows are marked relayed or recorded as
// seen and then kept forever, which on a machine that runs for years is a
// disk and query-plan problem rather than a correctness one.
type RetentionWorker struct {
	service   string
	store     RetentionStore
	retention time.Duration
	interval  time.Duration
}

// NewRetentionWorker takes the window in days and the sweep cadence in hours
// so it maps straight onto PROCESSED_EVENTS_RETENTION_DAYS and
// CLEANUP_SCHEDULER_INTERVAL_HOURS. Non-positive values fall back to the
// config defaults rather than sweeping constantly or never.
func NewRetentionWorker(service string, store RetentionStore, retentionDays, intervalHours int) *RetentionWorker {
	if retentionDays <= 0 {
		retentionDays = defaultRetentionDays
	}
	if intervalHours <= 0 {
		intervalHours = defaultIntervalHours
	}
	return &RetentionWorker{
		service:   service,
		store:     store,
		retention: time.Duration(retentionDays) * 24 * time.Hour,
		interval:  time.Duration(intervalHours) * time.Hour,
	}
}

// Start blocks on a ticker until ctx is cancelled. The first sweep waits one
// interval: startup is the busiest moment and nothing is urgent here.
func (w *RetentionWorker) Start(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "RetentionWorker"),
		zap.String("service", w.service),
	)
	log.Info("starting retention worker",
		zap.Duration("retention", w.retention),
		zap.Duration("interval", w.interval),
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping retention worker")
			return
		case <-ticker.C:
			w.sweep(ctx)
		}
	}
}

func (w *RetentionWorker) sweep(ctx context.Context) {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "RetentionWorker"),
		zap.String("service", w.service),
	)

	sweepCtx, cancel := context.WithTimeout(ctx, retentionQueryTimeout)
	defer cancel()

	before := time.Now().Add(-w.retention)

	outbox, err := w.store.DeleteRelayedOutboxEvents(sweepCtx, before)
	if err != nil {
		log.Error("outbox retention sweep failed", zap.Error(err))
	}

	processed, err := w.store.DeleteProcessedEvents(sweepCtx, before)
	if err != nil {
		log.Error("processed-events retention sweep failed", zap.Error(err))
	}

	if outbox > 0 || processed > 0 {
		log.Info("retention sweep complete",
			zap.Int64("outbox_deleted", outbox),
			zap.Int64("processed_deleted", processed),
			zap.Time("before", before),
		)
	}
}
