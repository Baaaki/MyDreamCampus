package service

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
)

// fakeStore mirrors the SQL guards of the repository in memory, so the
// service's state rules are tested against the same semantics.
type fakeStore struct {
	byRef       map[string]*db.Payment
	createCalls []db.CreatePaymentParams
	expireNow   time.Time
	events      []fakeEvent
}

type fakeEvent struct {
	eventType string
	payload   map[string]any
}

func newFakeStore() *fakeStore {
	return &fakeStore{byRef: map[string]*db.Payment{}}
}

func (f *fakeStore) add(p db.Payment) *db.Payment {
	f.byRef[p.ReferenceID] = &p
	return &p
}

func (f *fakeStore) CreatePayment(_ context.Context, params db.CreatePaymentParams) (db.Payment, error) {
	f.createCalls = append(f.createCalls, params)
	if existing, ok := f.byRef[params.ReferenceID]; ok {
		return *existing, nil
	}
	p := db.Payment{
		ID:             utils.UUIDToPgtype(uuid.New()),
		ReferenceID:    params.ReferenceID,
		StudentID:      params.StudentID,
		Amount:         params.Amount,
		RefundedAmount: utils.Float64ToPgNumeric(0),
		Currency:       params.Currency,
		Description:    params.Description,
		Status:         db.PaymentPaymentStatusEnumPending,
		ExpiresAt:      params.ExpiresAt,
	}
	f.byRef[p.ReferenceID] = &p
	return p, nil
}

func (f *fakeStore) GetPaymentByReferenceID(_ context.Context, ref string) (db.Payment, error) {
	p, ok := f.byRef[ref]
	if !ok {
		return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
	}
	return *p, nil
}

func (f *fakeStore) RefundPayment(_ context.Context, ref string, amount float64) (db.Payment, error) {
	p, ok := f.byRef[ref]
	if !ok || p.Status != db.PaymentPaymentStatusEnumCompleted {
		return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
	}
	paid, _ := utils.PgNumericToFloat64(p.Amount)
	refunded, _ := utils.PgNumericToFloat64(p.RefundedAmount)
	if refunded+amount > paid {
		return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
	}
	p.RefundedAmount = utils.Float64ToPgNumeric(refunded + amount)
	if refunded+amount >= paid {
		p.Status = db.PaymentPaymentStatusEnumRefunded
	}
	return *p, nil
}

func (f *fakeStore) ExpireOverduePayments(_ context.Context, now time.Time) (int64, error) {
	f.expireNow = now
	var n int64
	for _, p := range f.byRef {
		if p.Status == db.PaymentPaymentStatusEnumPending && !p.ExpiresAt.Time.After(now) {
			p.Status = db.PaymentPaymentStatusEnumExpired
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) GetPaymentByID(_ context.Context, id uuid.UUID) (db.Payment, error) {
	if p := f.byID(id); p != nil {
		return *p, nil
	}
	return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
}

func (f *fakeStore) CompletePaymentWithEvent(_ context.Context, params db.CompletePaymentParams, payload map[string]any) (db.Payment, error) {
	p := f.byID(utils.PgtypeToUUID(params.ID))
	if p == nil || p.Status != db.PaymentPaymentStatusEnumPending {
		return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
	}
	p.Status = db.PaymentPaymentStatusEnumCompleted
	p.CardBrand, p.CardLast4, p.CompletedAt = params.CardBrand, params.CardLast4, params.CompletedAt
	f.events = append(f.events, fakeEvent{"payment.completed", payload})
	return *p, nil
}

func (f *fakeStore) FailPaymentWithEvent(_ context.Context, params db.FailPaymentParams, payload map[string]any) (db.Payment, error) {
	p := f.byID(utils.PgtypeToUUID(params.ID))
	if p == nil || p.Status != db.PaymentPaymentStatusEnumPending {
		return db.Payment{}, serviceErrors.ErrPaymentNotFoundRepo
	}
	p.Status = db.PaymentPaymentStatusEnumFailed
	p.CardBrand, p.CardLast4, p.FailureReason = params.CardBrand, params.CardLast4, params.FailureReason
	f.events = append(f.events, fakeEvent{"payment.failed", payload})
	return *p, nil
}

func (f *fakeStore) ExpirePayment(_ context.Context, id uuid.UUID) error {
	if p := f.byID(id); p != nil && p.Status == db.PaymentPaymentStatusEnumPending {
		p.Status = db.PaymentPaymentStatusEnumExpired
	}
	return nil
}

func (f *fakeStore) byID(id uuid.UUID) *db.Payment {
	for _, p := range f.byRef {
		if utils.PgtypeToUUID(p.ID) == id {
			return p
		}
	}
	return nil
}
