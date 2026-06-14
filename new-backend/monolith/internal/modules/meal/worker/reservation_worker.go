package worker

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"go.uber.org/zap"
)

// ReservationWorker handles background jobs for reservations
type ReservationWorker struct {
	reservationRepo *repository.ReservationRepository
	logger          *zap.Logger
	stopChan        chan struct{}
}

func NewReservationWorker(
	reservationRepo *repository.ReservationRepository,
	logger *zap.Logger,
) *ReservationWorker {
	return &ReservationWorker{
		reservationRepo: reservationRepo,
		logger:          logger,
		stopChan:        make(chan struct{}),
	}
}

// Start starts all background jobs
func (w *ReservationWorker) Start(ctx context.Context) {
	w.logger.Info("starting reservation worker")

	// Start expiry job (runs every 1 minute)
	go w.runExpiryJob(ctx)

	// Start cleanup job (runs daily at 03:00 UTC+3)
	go w.runCleanupJob(ctx)
}

// Stop stops all background jobs
func (w *ReservationWorker) Stop() {
	w.logger.Info("stopping reservation worker")
	close(w.stopChan)
}

// runExpiryJob expires pending reservations that have timed out
func (w *ReservationWorker) runExpiryJob(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.expirePendingReservations(ctx); err != nil {
				w.logger.Error("failed to expire pending reservations", zap.Error(err))
			}
		case <-w.stopChan:
			w.logger.Info("expiry job stopped")
			return
		case <-ctx.Done():
			w.logger.Info("expiry job context cancelled")
			return
		}
	}
}

// runCleanupJob cleans up old expired reservations
func (w *ReservationWorker) runCleanupJob(ctx context.Context) {
	// Calculate next 03:00 UTC+3
	nextRun := w.getNext3AM()
	timer := time.NewTimer(time.Until(nextRun))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := w.cleanupExpiredReservations(ctx); err != nil {
				w.logger.Error("failed to cleanup expired reservations", zap.Error(err))
			}
			// Schedule next run
			nextRun = w.getNext3AM()
			timer.Reset(time.Until(nextRun))

		case <-w.stopChan:
			w.logger.Info("cleanup job stopped")
			return
		case <-ctx.Done():
			w.logger.Info("cleanup job context cancelled")
			return
		}
	}
}

func (w *ReservationWorker) expirePendingReservations(ctx context.Context) error {
	batchSize := int32(100)
	err := w.reservationRepo.ExpirePendingReservations(ctx, batchSize)
	if err != nil {
		return err
	}

	w.logger.Debug("expired pending reservations", zap.Int32("batch_size", batchSize))
	return nil
}

func (w *ReservationWorker) cleanupExpiredReservations(ctx context.Context) error {
	batchSize := int32(500)
	err := w.reservationRepo.CleanupExpiredReservations(ctx, batchSize)
	if err != nil {
		return err
	}

	w.logger.Info("cleaned up expired reservations", zap.Int32("batch_size", batchSize))
	return nil
}

func (w *ReservationWorker) getNext3AM() time.Time {
	now := clock.Now().In(time.FixedZone("UTC+3", 3*3600))
	next3AM := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())

	// If it's already past 03:00 today, schedule for tomorrow
	if now.After(next3AM) {
		next3AM = next3AM.Add(24 * time.Hour)
	}

	return next3AM
}
