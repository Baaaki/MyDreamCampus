package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// CreateOutboxEventWithTx creates a new outbox event within a transaction
func (r *OutboxRepository) CreateOutboxEventWithTx(ctx context.Context, tx pgx.Tx, params db.CreateOutboxEventParams) (db.OutboxEvent, error) {
	qtx := r.queries.WithTx(tx)
	event, err := qtx.CreateOutboxEvent(ctx, params)
	if err != nil {
		return db.OutboxEvent{}, fmt.Errorf("%w: failed to create outbox event in transaction: %v", sharedErrors.ErrQueryFailed, err)
	}
	return event, nil
}

// GetPendingEvents retrieves pending outbox events with limit
func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	events, err := r.queries.GetPendingEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending events: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

// MarkEventProcessed marks an outbox event as processed
func (r *OutboxRepository) MarkEventProcessed(ctx context.Context, id uuid.UUID) error {
	err := r.queries.MarkEventProcessed(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to mark event as processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// MarkEventFailed marks an outbox event as failed with error message
func (r *OutboxRepository) MarkEventFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	err := r.queries.MarkEventFailed(ctx, db.MarkEventFailedParams{
		ID:           utils.UUIDToPgtype(id),
		ErrorMessage: utils.StringToPgText(errorMessage),
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event as failed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// GetFailedEventsForRetry retrieves failed events that can be retried
func (r *OutboxRepository) GetFailedEventsForRetry(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	events, err := r.queries.GetFailedEventsForRetry(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get failed events for retry: %v", sharedErrors.ErrQueryFailed, err)
	}
	return events, nil
}

// BeginTx starts a new transaction
func (r *OutboxRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	return tx, nil
}
