package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
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
