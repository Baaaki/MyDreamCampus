package repository

import (
	"context"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/google/uuid"
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

// CreateProcessedEvent marks an event as processed
func (r *ProcessedEventsRepository) CreateProcessedEvent(ctx context.Context, params db.CreateProcessedEventParams) error {
	err := r.queries.CreateProcessedEvent(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: failed to create processed event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// IsEventProcessed checks if an event has been processed
func (r *ProcessedEventsRepository) IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	exists, err := r.queries.IsEventProcessed(ctx, utils.UUIDToPgtype(eventID))
	if err != nil {
		return false, fmt.Errorf("%w: failed to check if event is processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return exists, nil
}

// CleanupOldProcessedEvents removes old processed events
func (r *ProcessedEventsRepository) CleanupOldProcessedEvents(ctx context.Context) error {
	err := r.queries.CleanupOldProcessedEvents(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to cleanup old processed events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
