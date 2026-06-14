package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/db"
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

func (r *OutboxRepository) CreateOutboxEvent(ctx context.Context, arg db.CreateOutboxEventParams) (db.GradesOutboxEvent, error) {
	return r.queries.CreateOutboxEvent(ctx, arg)
}

func (r *OutboxRepository) GetPendingOutboxEvents(ctx context.Context, limit int32) ([]db.GradesOutboxEvent, error) {
	return r.queries.GetPendingOutboxEvents(ctx, limit)
}

func (r *OutboxRepository) MarkOutboxEventProcessed(ctx context.Context, id uuid.UUID) error {
	return r.queries.MarkOutboxEventProcessed(ctx, id)
}

func (r *OutboxRepository) MarkOutboxEventFailed(ctx context.Context, arg db.MarkOutboxEventFailedParams) error {
	return r.queries.MarkOutboxEventFailed(ctx, arg)
}

func (r *OutboxRepository) RetryFailedOutboxEvent(ctx context.Context, id uuid.UUID) error {
	return r.queries.RetryFailedOutboxEvent(ctx, id)
}
