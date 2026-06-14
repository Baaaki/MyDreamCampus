package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/db"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
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

func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int32) ([]db.GetPendingOutboxEventsRow, error) {
	events, err := r.queries.GetPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending outbox events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

func (r *OutboxRepository) MarkEventProcessed(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.MarkOutboxEventProcessed(ctx, utils.UUIDToPgtype(id)); err != nil {
		return fmt.Errorf("%w: failed to mark outbox event as processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

func (r *OutboxRepository) MarkEventFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	if err := r.queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID:           utils.UUIDToPgtype(id),
		ErrorMessage: utils.StringToPgText(errorMessage),
	}); err != nil {
		return fmt.Errorf("%w: failed to mark outbox event as failed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

func (r *OutboxRepository) GetFailedEvents(ctx context.Context, limit int32) ([]db.GetFailedOutboxEventsRow, error) {
	events, err := r.queries.GetFailedOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get failed outbox events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

func (r *OutboxRepository) ResetFailedEvent(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.ResetFailedOutboxEvent(ctx, utils.UUIDToPgtype(id)); err != nil {
		return fmt.Errorf("%w: failed to reset failed outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
