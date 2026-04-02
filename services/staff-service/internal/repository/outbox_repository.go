package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/staff-service/internal/db"
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

// GetUnprocessedEvents retrieves unprocessed outbox events
func (r *OutboxRepository) GetUnprocessedEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	events, err := r.queries.GetUnprocessedEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed events: %w", err)
	}
	return events, nil
}

// MarkEventProcessed marks an outbox event as processed
func (r *OutboxRepository) MarkEventProcessed(ctx context.Context, id int32) error {
	err := r.queries.MarkEventProcessed(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}
	return nil
}
