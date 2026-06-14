package repository

import (
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProcessedEventsRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewProcessedEventsRepository(pool *pgxpool.Pool) *ProcessedEventsRepository {
	return &ProcessedEventsRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// IsEventProcessed checks if an event has already been processed
func (r *ProcessedEventsRepository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	processed, err := r.queries.IsEventProcessed(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("%w: failed to check event processing status: %v", sharedErrors.ErrQueryFailed, err)
	}
	return processed, nil
}

// MarkEventProcessed marks an event as processed (idempotency)
func (r *ProcessedEventsRepository) MarkEventProcessed(ctx context.Context, eventID, eventType string) error {
	err := r.queries.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   eventID,
		EventType: eventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event as processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// DeleteOldProcessedEvents deletes processed events older than 30 days
func (r *ProcessedEventsRepository) DeleteOldProcessedEvents(ctx context.Context) error {
	err := r.queries.DeleteOldProcessedEvents(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to delete old processed events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
