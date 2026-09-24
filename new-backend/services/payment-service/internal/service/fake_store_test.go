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
