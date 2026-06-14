package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
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

func (r *OutboxRepository) GetPendingOutboxEvents(ctx context.Context, limit int32) ([]db.GetPendingOutboxEventsRow, error) {
	return r.queries.GetPendingOutboxEvents(ctx, limit)
}

func (r *OutboxRepository) MarkOutboxEventProcessed(ctx context.Context, id pgtype.UUID) error {
	return r.queries.MarkOutboxEventProcessed(ctx, id)
}

func (r *OutboxRepository) MarkOutboxEventFailed(ctx context.Context, id pgtype.UUID, errorMsg string) error {
	return r.queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID:           id,
		ErrorMessage: utils.StringToPgText(errorMsg),
	})
}

func (r *OutboxRepository) CreateOutboxEvent(ctx context.Context, eventType, routingKey string, payload []byte) error {
	return r.queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  eventType,
		RoutingKey: routingKey,
		Payload:    payload,
	})
}
