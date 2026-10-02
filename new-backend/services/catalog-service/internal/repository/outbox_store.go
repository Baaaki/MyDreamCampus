package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type outboxClaimTx struct {
	repo *OutboxRepository
}

func (c *outboxClaimTx) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventbus.SettleTimeout)
	defer cancel()
	return c.repo.MarkEventProcessed(settleCtx, id)
}

func (c *outboxClaimTx) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventbus.SettleTimeout)
	defer cancel()
	return c.repo.MarkEventFailed(settleCtx, id, msg)
}

func (s *OutboxStore) ClaimPending(ctx context.Context, limit int32, settle func(ctx context.Context, events []eventbus.OutboxEvent, tx eventbus.ClaimTx) error) error {
	return eventbus.InClaimTx(ctx, s.repo.pool, func(tx pgx.Tx) error {
		txRepo := s.repo.withTx(tx)
		rows, err := txRepo.GetPendingEvents(ctx, limit)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return settle(ctx, nil, nil)
		}
		events := make([]eventbus.OutboxEvent, len(rows))
		for i, r := range rows {
			events[i] = rowToEvent(r)
		}
		return settle(ctx, events, &outboxClaimTx{repo: txRepo})
	})
}

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
		ID:            utils.PgtypeToUUID(r.ID),
		EventType:     r.EventType,
		RoutingKey:    r.RoutingKey,
		Payload:       r.Payload,
		CreatedAt:     r.CreatedAt.Time,
		RetryCount:    r.RetryCount.Int16,
		MaxRetries:    r.MaxRetries.Int16,
		CorrelationID: utils.CorrelationIDString(r.CorrelationID),
	}
}
