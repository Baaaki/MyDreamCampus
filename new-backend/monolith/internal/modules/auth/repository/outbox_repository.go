package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/db"
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

// GetPendingEvents retrieves pending outbox events
func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int32) ([]db.GetPendingOutboxEventsRow, error) {
	events, err := r.queries.GetPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

// MarkEventProcessed marks an outbox event as processed
func (r *OutboxRepository) MarkEventProcessed(ctx context.Context, id uuid.UUID) error {
	err := r.queries.MarkOutboxEventProcessed(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to mark event as processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// MarkEventFailed marks an outbox event as failed with error message
func (r *OutboxRepository) MarkEventFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	var errMsg *string
	if errorMessage != "" {
		errMsg = &errorMessage
	}
	
	err := r.queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID:           utils.UUIDToPgtype(id),
		ErrorMessage: errMsg,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event as failed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// GetFailedEvents retrieves failed events that can be retried
func (r *OutboxRepository) GetFailedEvents(ctx context.Context, limit int32) ([]db.GetFailedOutboxEventsRow, error) {
	events, err := r.queries.GetFailedOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get failed events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

// ResetFailedEvent resets a failed event to pending status
func (r *OutboxRepository) ResetFailedEvent(ctx context.Context, id uuid.UUID) error {
	err := r.queries.ResetFailedOutboxEvent(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to reset failed event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
