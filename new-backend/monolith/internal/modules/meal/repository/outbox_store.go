package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"

	"github.com/google/uuid"
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
			ID:         row.ID.Bytes,
			EventType:  row.EventType,
			RoutingKey: row.EventType, // using event_type as routing key
			Payload:    row.Payload,
			RetryCount: row.RetryCount,
			MaxRetries: row.MaxRetries,
			CreatedAt:  row.CreatedAt.Time,
		}
	}
	return events, nil
}

func (s *OutboxStore) GetFailed(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetFailedOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}

	events := make([]eventbus.OutboxEvent, len(rows))
	for i, row := range rows {
		events[i] = eventbus.OutboxEvent{
			ID:         row.ID.Bytes,
			EventType:  row.EventType,
			RoutingKey: row.EventType,
			Payload:    row.Payload,
			RetryCount: row.RetryCount,
			MaxRetries: row.MaxRetries,
			CreatedAt:  row.CreatedAt.Time,
		}
	}
	return events, nil
}

func (s *OutboxStore) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkOutboxEventPublished(ctx, id)
}

func (s *OutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	return s.repo.MarkOutboxEventFailed(ctx, id, errMsg)
}

func (s *OutboxStore) Reset(ctx context.Context, id uuid.UUID) error {
	return s.repo.RetryFailedOutboxEvent(ctx, id)
}
