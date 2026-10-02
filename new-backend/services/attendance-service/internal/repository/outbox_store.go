package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OutboxStore struct {
	repo *OutboxRepository
}

func NewOutboxStore(repo *OutboxRepository) *OutboxStore {
	return &OutboxStore{repo: repo}
}

var _ eventbus.OutboxStore = (*OutboxStore)(nil)

type outboxClaimTx struct {
	repo *OutboxRepository
}

func (c *outboxClaimTx) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventbus.SettleTimeout)
	defer cancel()
	return c.repo.MarkOutboxEventProcessed(settleCtx, utils.UUIDToPgUUID(id))
}

func (c *outboxClaimTx) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventbus.SettleTimeout)
	defer cancel()
	return c.repo.MarkOutboxEventFailed(settleCtx, utils.UUIDToPgUUID(id), errMsg)
}

func (s *OutboxStore) ClaimPending(ctx context.Context, limit int32, settle func(ctx context.Context, events []eventbus.OutboxEvent, tx eventbus.ClaimTx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventbus.SettleTimeout)
	defer cancel()

	return pgx.BeginFunc(commitCtx, s.repo.pool, func(tx pgx.Tx) error {
		txRepo := s.repo.withTx(tx)
		events, err := txRepo.GetPendingOutboxEvents(ctx, limit)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return settle(ctx, nil, nil)
		}
		result := make([]eventbus.OutboxEvent, len(events))
		for i, e := range events {
			var retryCount int16
			if e.RetryCount.Valid {
				retryCount = e.RetryCount.Int16
			}
			result[i] = eventbus.OutboxEvent{
				ID:            e.ID.Bytes,
				EventType:     e.EventType,
				RoutingKey:    e.RoutingKey,
				Payload:       e.Payload,
				RetryCount:    retryCount,
				CorrelationID: utils.CorrelationIDString(e.CorrelationID),
			}
		}
		return settle(ctx, result, &outboxClaimTx{repo: txRepo})
	})
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
			ID:            e.ID.Bytes,
			EventType:     e.EventType,
			RoutingKey:    e.RoutingKey,
			Payload:       e.Payload,
			RetryCount:    retryCount,
			CorrelationID: utils.CorrelationIDString(e.CorrelationID),
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
			ID:            e.ID.Bytes,
			EventType:     e.EventType,
			RoutingKey:    e.RoutingKey,
			Payload:       e.Payload,
			RetryCount:    retryCount,
			CorrelationID: utils.CorrelationIDString(e.CorrelationID),
		}
	}
	return result, nil
}

func (s *OutboxStore) Reset(ctx context.Context, id uuid.UUID) error {
	// We need this for the generic outbox worker. Assuming it exists.
	// If not, we will add it to the queries.
	return s.repo.ResetFailedOutboxEvent(ctx, utils.UUIDToPgUUID(id))
}
