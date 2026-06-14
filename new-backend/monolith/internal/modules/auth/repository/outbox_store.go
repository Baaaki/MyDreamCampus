package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
)

// OutboxStore adapts the sqlc-shaped OutboxRepository to eventbus.OutboxStore
// so the generic eventbus.OutboxWorker can drive this module's outbox table.
// The conversion is mechanical — drop pgx wrappers, expose plain Go types.
type OutboxStore struct {
	repo *OutboxRepository
}

// NewOutboxStore wraps an OutboxRepository for the generic worker.
func NewOutboxStore(repo *OutboxRepository) *OutboxStore {
	return &OutboxStore{repo: repo}
}

// Compile-time assertion that the adapter satisfies the worker's contract.
var _ eventbus.OutboxStore = (*OutboxStore)(nil)

func (s *OutboxStore) GetPending(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetPendingEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]eventbus.OutboxEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, pendingRowToEvent(r))
	}
	return out, nil
}

func (s *OutboxStore) GetFailed(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetFailedEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]eventbus.OutboxEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, failedRowToEvent(r))
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

func pendingRowToEvent(r db.GetPendingOutboxEventsRow) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{
		ID:         utils.PgtypeToUUID(r.ID),
		EventType:  r.EventType,
		RoutingKey: r.RoutingKey,
		Payload:    r.Payload,
		CreatedAt:  r.CreatedAt.Time,
		RetryCount: r.RetryCount,
		MaxRetries: r.MaxRetries,
	}
}

func failedRowToEvent(r db.GetFailedOutboxEventsRow) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{
		ID:         utils.PgtypeToUUID(r.ID),
		EventType:  r.EventType,
		RoutingKey: r.RoutingKey,
		Payload:    r.Payload,
		CreatedAt:  r.CreatedAt.Time,
		RetryCount: r.RetryCount,
		MaxRetries: r.MaxRetries,
	}
}
