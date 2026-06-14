package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *EventRepository) IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	count, err := r.queries.CheckEventProcessed(ctx, utils.UUIDToPgUUID(eventID))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *EventRepository) MarkEventProcessed(ctx context.Context, eventID uuid.UUID, eventType string) error {
	return r.queries.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgUUID(eventID),
		EventType: eventType,
	})
}
