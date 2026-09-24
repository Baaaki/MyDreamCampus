package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Expirer is the one call the worker makes on the payment service.
type Expirer interface {
	ExpireOverdue(ctx context.Context) (int64, error)
}

// ExpiryWorker closes pending payments nobody confirmed in time. The
// interval is on the real clock; the deadline it checks is on the service
// clock, inside ExpireOverdue.
type ExpiryWorker struct {
	expirer  Expirer
	interval time.Duration
	logger   *zap.Logger
}

func NewExpiryWorker(expirer Expirer, interval time.Duration, logger *zap.Logger) *ExpiryWorker {
	return &ExpiryWorker{
		expirer:  expirer,
		interval: interval,
		logger:   logger,
	}
}

// Start blocks until ctx is cancelled.
func (w *ExpiryWorker) Start(ctx context.Context) {
	w.logger.Info("starting payment expiry worker", zap.Duration("interval", w.interval))

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.run(ctx)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("stopping payment expiry worker")
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *ExpiryWorker) run(ctx context.Context) {
	n, err := w.expirer.ExpireOverdue(ctx)
	if err != nil {
		w.logger.Error("failed to expire payments", zap.Error(err))
		return
	}
	if n > 0 {
		w.logger.Info("expired overdue payments", zap.Int64("count", n))
	}
}
