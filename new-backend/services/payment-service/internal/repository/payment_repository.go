package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
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
