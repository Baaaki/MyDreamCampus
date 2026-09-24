package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/events"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreatePayment inserts a pending payment, or returns the existing one when
// the reference was already initiated.
func (r *PaymentRepository) CreatePayment(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
	payment, err := r.queries.CreatePayment(ctx, params)
	if err != nil {
		return db.Payment{}, fmt.Errorf("%w: failed to create payment: %v", sharedErrors.ErrQueryFailed, err)
	}
	return payment, nil
}

func (r *PaymentRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (db.Payment, error) {
	payment, err := r.queries.GetPaymentByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return db.Payment{}, notFoundOr(err, "failed to get payment")
	}
	return payment, nil
}

// CompletePaymentWithEvent settles a pending payment as paid and writes
// payment.completed in the same transaction. ErrPaymentNotFoundRepo means
// the payment was no longer pending.
func (r *PaymentRepository) CompletePaymentWithEvent(ctx context.Context, params db.CompletePaymentParams, payload map[string]any) (db.Payment, error) {
	return r.settleWithEvent(ctx, events.EventPaymentCompleted, events.RoutingKeyPaymentCompleted, payload,
		func(q *db.Queries) (db.Payment, error) { return q.CompletePayment(ctx, params) })
}

// FailPaymentWithEvent settles a pending payment as declined and writes
// payment.failed in the same transaction.
func (r *PaymentRepository) FailPaymentWithEvent(ctx context.Context, params db.FailPaymentParams, payload map[string]any) (db.Payment, error) {
	return r.settleWithEvent(ctx, events.EventPaymentFailed, events.RoutingKeyPaymentFailed, payload,
		func(q *db.Queries) (db.Payment, error) { return q.FailPayment(ctx, params) })
}

func (r *PaymentRepository) settleWithEvent(
	ctx context.Context,
	eventType, routingKey string,
	payload map[string]any,
	update func(*db.Queries) (db.Payment, error),
) (db.Payment, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return db.Payment{}, fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Payment{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)
	payment, err := update(qtx)
	if err != nil {
		return db.Payment{}, notFoundOr(err, "failed to settle payment")
	}

	if _, err := qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     eventType,
		RoutingKey:    routingKey,
		Payload:       body,
	}); err != nil {
		return db.Payment{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Payment{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	return payment, nil
}

// ExpirePayment closes one pending payment whose deadline has passed. A
// payment that is no longer pending is left alone.
func (r *PaymentRepository) ExpirePayment(ctx context.Context, id uuid.UUID) error {
	if _, err := r.queries.ExpirePayment(ctx, utils.UUIDToPgtype(id)); err != nil {
		return fmt.Errorf("%w: failed to expire payment: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

func (r *PaymentRepository) GetPaymentByReferenceID(ctx context.Context, referenceID string) (db.Payment, error) {
	payment, err := r.queries.GetPaymentByReferenceID(ctx, referenceID)
	if err != nil {
		return db.Payment{}, notFoundOr(err, "failed to get payment by reference")
	}
	return payment, nil
}

// RefundPayment adds amount to the refunded total. ErrPaymentNotFoundRepo
// means no row qualified — missing, not completed, or over the paid amount;
// the caller re-reads the payment to tell which.
func (r *PaymentRepository) RefundPayment(ctx context.Context, referenceID string, amount float64) (db.Payment, error) {
	payment, err := r.queries.RefundPayment(ctx, db.RefundPaymentParams{
		Refund:      utils.Float64ToPgNumeric(amount),
		ReferenceID: referenceID,
	})
	if err != nil {
		return db.Payment{}, notFoundOr(err, "failed to refund payment")
	}
	return payment, nil
}

// ExpireOverduePayments marks every pending payment whose deadline is at or
// before now as expired and reports how many there were.
func (r *PaymentRepository) ExpireOverduePayments(ctx context.Context, now time.Time) (int64, error) {
	n, err := r.queries.ExpireOverduePayments(ctx, utils.TimeToPgTimestamptz(now))
	if err != nil {
		return 0, fmt.Errorf("%w: failed to expire payments: %v", sharedErrors.ErrQueryFailed, err)
	}
	return n, nil
}

func notFoundOr(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", serviceErrors.ErrPaymentNotFoundRepo, msg)
	}
	return fmt.Errorf("%w: %s: %v", sharedErrors.ErrQueryFailed, msg, err)
}
