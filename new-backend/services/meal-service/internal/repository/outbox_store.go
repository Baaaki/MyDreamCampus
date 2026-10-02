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
	return c.repo.MarkOutboxEventPublished(settleCtx, id)
}

func (c *outboxClaimTx) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventbus.SettleTimeout)
	defer cancel()
	return c.repo.MarkOutboxEventFailed(settleCtx, id, errMsg)
}

func (s *OutboxStore) ClaimPending(ctx context.Context, limit int32, settle func(ctx context.Context, events []eventbus.OutboxEvent, tx eventbus.ClaimTx) error) error {
	return eventbus.InClaimTx(ctx, s.repo.pool, func(tx pgx.Tx) error {
		txRepo := s.repo.withTx(tx)
		rows, err := txRepo.GetPendingOutboxEvents(ctx, limit)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return settle(ctx, nil, nil)
		}
		events := make([]eventbus.OutboxEvent, len(rows))
		for i, row := range rows {
			events[i] = eventbus.OutboxEvent{
				ID:            row.ID.Bytes,
				EventType:     row.EventType,
				RoutingKey:    row.EventType,
				Payload:       row.Payload,
				RetryCount:    row.RetryCount,
				MaxRetries:    row.MaxRetries,
				CreatedAt:     row.CreatedAt.Time,
				CorrelationID: utils.CorrelationIDString(row.CorrelationID),
			}
		}
		return settle(ctx, events, &outboxClaimTx{repo: txRepo})
	})
}

func (s *OutboxStore) GetPending(ctx context.Context, limit int32) ([]eventbus.OutboxEvent, error) {
	rows, err := s.repo.GetPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}

	events := make([]eventbus.OutboxEvent, len(rows))
	for i, row := range rows {
		events[i] = eventbus.OutboxEvent{
			ID:            row.ID.Bytes,
			EventType:     row.EventType,
			RoutingKey:    row.EventType, // using event_type as routing key
			Payload:       row.Payload,
			RetryCount:    row.RetryCount,
			MaxRetries:    row.MaxRetries,
			CreatedAt:     row.CreatedAt.Time,
			CorrelationID: utils.CorrelationIDString(row.CorrelationID),
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
			ID:            row.ID.Bytes,
			EventType:     row.EventType,
			RoutingKey:    row.EventType,
			Payload:       row.Payload,
			RetryCount:    row.RetryCount,
			MaxRetries:    row.MaxRetries,
			CreatedAt:     row.CreatedAt.Time,
			CorrelationID: utils.CorrelationIDString(row.CorrelationID),
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
