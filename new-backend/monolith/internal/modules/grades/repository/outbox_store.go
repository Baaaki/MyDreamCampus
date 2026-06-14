package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type OutboxStore struct {
	repo *OutboxRepository
}

func NewOutboxStore(repo *OutboxRepository) *OutboxStore {
	return &OutboxStore{repo: repo}
}

func (s *OutboxStore) GetPending(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}

	events := make([]eventbus.OutboxEvent, len(rows))
	for i, row := range rows {
		events[i] = eventbus.OutboxEvent{
			ID:         row.ID,
			EventType:  row.EventType,
			RoutingKey: row.RoutingKey,
			Payload:    row.Payload,
			RetryCount: row.RetryCount.Int16,
			MaxRetries: row.MaxRetries.Int16,
			CreatedAt:  row.CreatedAt.Time,
		}
	}
	return events, nil
}

func (s *OutboxStore) GetFailed(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	// Not implemented in outbox_repository.go for grades yet
	return []eventbus.OutboxEvent{}, nil
}

func (s *OutboxStore) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkOutboxEventProcessed(ctx, id)
}

func (s *OutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	var errPgText pgtype.Text
	if errMsg != "" {
		errPgText = pgtype.Text{String: errMsg, Valid: true}
	}
	return s.repo.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID:           id,
		ErrorMessage: errPgText,
	})
}

func (s *OutboxStore) Reset(ctx context.Context, id uuid.UUID) error {
	return s.repo.RetryFailedOutboxEvent(ctx, id)
}
