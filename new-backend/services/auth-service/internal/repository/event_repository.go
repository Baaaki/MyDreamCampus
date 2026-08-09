package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// IsEventProcessed checks if an event has been processed
func (r *EventRepository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	exists, err := r.queries.IsEventProcessed(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("%w: failed to check if event processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return exists, nil
}

// MarkEventProcessed marks an event as processed
func (r *EventRepository) MarkEventProcessed(ctx context.Context, eventID, eventType string) error {
	err := r.queries.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   eventID,
		EventType: eventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
