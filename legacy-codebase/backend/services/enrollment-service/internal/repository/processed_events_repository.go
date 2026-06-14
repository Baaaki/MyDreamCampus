package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
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

func (r *ProcessedEventsRepository) IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	processed, err := r.queries.IsEventProcessed(ctx, utils.UUIDToPgtype(eventID))
	if err != nil {
		return false, fmt.Errorf("%w: failed to check if event is processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return processed, nil
}

func (r *ProcessedEventsRepository) CreateProcessedEvent(ctx context.Context, eventID uuid.UUID, eventType string) error {
	_, err := r.queries.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: eventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to create processed event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
