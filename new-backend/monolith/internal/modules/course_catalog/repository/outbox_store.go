package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
)

// OutboxStore adapts course_catalog's OutboxRepository to eventbus.OutboxStore
// so the generic worker can drive it.
type OutboxStore struct {
	repo *OutboxRepository
}

func NewOutboxStore(repo *OutboxRepository) *OutboxStore {
	return &OutboxStore{repo: repo}
}

var _ eventbus.OutboxStore = (*OutboxStore)(nil)

func (s *OutboxStore) GetPending(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetPendingEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]eventbus.OutboxEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, rowToEvent(r))
	}
	return out, nil
}

func (s *OutboxStore) GetFailed(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetFailedEventsForRetry(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]eventbus.OutboxEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, rowToEvent(r))
	}
	return out, nil
}

func (s *OutboxStore) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkEventProcessed(ctx, id)
}

func (s *OutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	return s.repo.MarkEventFailed(ctx, id, msg)
}

func (s *OutboxStore) Reset(ctx context.Context, id uuid.UUID) error {
	return s.repo.ResetFailedEvent(ctx, id)
}

func rowToEvent(r db.OutboxEvent) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{
		ID:         utils.PgtypeToUUID(r.ID),
		EventType:  r.EventType,
		RoutingKey: r.RoutingKey,
		Payload:    r.Payload,
		CreatedAt:  r.CreatedAt.Time,
		RetryCount: r.RetryCount.Int16,
		MaxRetries: r.MaxRetries.Int16,
	}
}
