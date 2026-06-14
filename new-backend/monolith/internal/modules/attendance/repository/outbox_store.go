package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
)

type OutboxStore struct {
	repo *OutboxRepository
}

func NewOutboxStore(repo *OutboxRepository) *OutboxStore {
	return &OutboxStore{repo: repo}
}

func (s *OutboxStore) GetPending(ctx context.Context, batchSize int32) ([]eventbus.OutboxEvent, error) {
	events, err := s.repo.GetPendingOutboxEvents(ctx, batchSize)
	if err != nil {
		return nil, err
	}

	result := make([]eventbus.OutboxEvent, len(events))
	for i, e := range events {
		var retryCount int16
		if e.RetryCount.Valid {
			retryCount = e.RetryCount.Int16
		}

		result[i] = eventbus.OutboxEvent{
			ID:         e.ID.Bytes,
			EventType:  e.EventType,
			RoutingKey: e.RoutingKey,
			Payload:    e.Payload,
			RetryCount: retryCount,
		}
	}
	return result, nil
}

func (s *OutboxStore) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkOutboxEventProcessed(ctx, utils.UUIDToPgUUID(id))
}

func (s *OutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	return s.repo.MarkOutboxEventFailed(ctx, utils.UUIDToPgUUID(id), errMsg)
}

func (s *OutboxStore) GetFailed(ctx context.Context, batchSize int32) ([]eventbus.OutboxEvent, error) {
	// The outbox_worker needs this, let's assume attendance outbox repository has it.
	// If it doesn't, we will fix it later. For now we stub it or use the repo method.
	events, err := s.repo.GetFailedOutboxEvents(ctx, batchSize)
	if err != nil {
		return nil, err
	}

	result := make([]eventbus.OutboxEvent, len(events))
	for i, e := range events {
		var retryCount int16
		if e.RetryCount.Valid {
			retryCount = e.RetryCount.Int16
		}

		result[i] = eventbus.OutboxEvent{
			ID:         e.ID.Bytes,
			EventType:  e.EventType,
			RoutingKey: e.RoutingKey,
			Payload:    e.Payload,
			RetryCount: retryCount,
		}
	}
	return result, nil
}

func (s *OutboxStore) Reset(ctx context.Context, id uuid.UUID) error {
	// We need this for the generic outbox worker. Assuming it exists.
	// If not, we will add it to the queries.
	return s.repo.ResetFailedOutboxEvent(ctx, utils.UUIDToPgUUID(id))
}
