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

type OutboxRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *OutboxRepository) GetPendingOutboxEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	events, err := r.queries.GetPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending outbox events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

func (r *OutboxRepository) MarkOutboxEventProcessed(ctx context.Context, id uuid.UUID) error {
	err := r.queries.MarkOutboxEventProcessed(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to mark outbox event as processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

func (r *OutboxRepository) MarkOutboxEventFailed(ctx context.Context, id uuid.UUID) error {
	err := r.queries.MarkOutboxEventFailed(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to mark outbox event as failed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
