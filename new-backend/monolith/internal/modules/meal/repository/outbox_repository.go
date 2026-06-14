package repository

import (
	"context"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

// CreateOutboxEvent creates a new outbox event
func (r *OutboxRepository) CreateOutboxEvent(ctx context.Context, params db.CreateOutboxEventParams) (db.OutboxEvent, error) {
	event, err := r.queries.CreateOutboxEvent(ctx, params)
	if err != nil {
		return db.OutboxEvent{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return event, nil
}

// GetPendingOutboxEvents returns pending outbox events ready for retry
func (r *OutboxRepository) GetPendingOutboxEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	events, err := r.queries.GetPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending outbox events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

// MarkOutboxEventPublished marks an outbox event as published
func (r *OutboxRepository) MarkOutboxEventPublished(ctx context.Context, id uuid.UUID) error {
	err := r.queries.MarkOutboxEventPublished(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to mark outbox event as published: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// MarkOutboxEventFailed marks an outbox event as failed
func (r *OutboxRepository) MarkOutboxEventFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	err := r.queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID: utils.UUIDToPgtype(id),
		LastError: pgtype.Text{String: errorMsg, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark outbox event as failed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// UpdateOutboxEventRetry updates retry information for an outbox event
func (r *OutboxRepository) UpdateOutboxEventRetry(ctx context.Context, id uuid.UUID, nextRetryAt pgtype.Timestamptz, errorMsg string) error {
	err := r.queries.UpdateOutboxEventRetry(ctx, db.UpdateOutboxEventRetryParams{
		ID:           utils.UUIDToPgtype(id),
		NextRetryAt:  nextRetryAt,
		LastError:    pgtype.Text{String: errorMsg, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("%w: failed to update outbox event retry: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// GetFailedOutboxEvents returns all failed outbox events
func (r *OutboxRepository) GetFailedOutboxEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	events, err := r.queries.GetFailedOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get failed outbox events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

// RetryFailedOutboxEvent resets a failed event back to pending state
func (r *OutboxRepository) RetryFailedOutboxEvent(ctx context.Context, id uuid.UUID) error {
	err := r.queries.RetryFailedOutboxEvent(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to retry failed outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
