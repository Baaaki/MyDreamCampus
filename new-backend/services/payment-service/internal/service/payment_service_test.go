package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/clock/clocktest"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var fixedNow = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

func newTestService(store Store) *PaymentService {
	return NewPaymentService(store, 15*time.Minute, zap.NewNop())
}

func validInitiate() InitiatePaymentRequest {
	return InitiatePaymentRequest{
		ReferenceID: "res_" + uuid.NewString(),
		Amount:      45,
		Currency:    "TRY",
		Description: "Meal reservation",
		StudentID:   uuid.NewString(),
	}
}

func completedPayment(ref string, amount float64) db.Payment {
	return db.Payment{
		ID:             utils.UUIDToPgtype(uuid.New()),
		ReferenceID:    ref,
		StudentID:      utils.UUIDToPgtype(uuid.New()),
		Amount:         utils.Float64ToPgNumeric(amount),
		RefundedAmount: utils.Float64ToPgNumeric(0),
		Currency:       "TRY",
		Status:         db.PaymentPaymentStatusEnumCompleted,
		ExpiresAt:      utils.TimeToPgTimestamptz(fixedNow),
	}
}

func TestInitiatePayment_ValidRequest_CreatesPendingPaymentOnServiceClock(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	req := validInitiate()

	resp, err := newTestService(store).InitiatePayment(context.Background(), req)

	require.NoError(t, err)
	require.Len(t, store.createCalls, 1)
	assert.Equal(t, fixedNow.Add(15*time.Minute), store.createCalls[0].ExpiresAt.Time)
	assert.Equal(t, req.StudentID, utils.PgtypeToUUIDString(store.createCalls[0].StudentID))
	assert.Equal(t, fixedNow.Add(15*time.Minute), resp.ExpiresAt)
	assert.InDelta(t, 45.0, resp.Amount, 0.001)
	assert.Equal(t, "TRY", resp.Currency)
	assert.Equal(t, db.PaymentPaymentStatusEnumPending, store.byRef[req.ReferenceID].Status)
}

func TestInitiatePayment_RetriedReference_ReturnsFirstPayment(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	svc := newTestService(store)
	req := validInitiate()

	first, err := svc.InitiatePayment(context.Background(), req)
	require.NoError(t, err)
	second, err := svc.InitiatePayment(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, first.PaymentID, second.PaymentID)
	assert.Len(t, store.byRef, 1)
}

func TestInitiatePayment_InvalidInput_ReturnsValidationError(t *testing.T) {
	cases := map[string]func(*InitiatePaymentRequest){
		"student id is not a uuid":  func(r *InitiatePaymentRequest) { r.StudentID = "abc" },
		"amount below one kurus":    func(r *InitiatePaymentRequest) { r.Amount = 0.001 },
		"currency is not three":     func(r *InitiatePaymentRequest) { r.Currency = "TL" },
		"reference has no prefix":   func(r *InitiatePaymentRequest) { r.ReferenceID = uuid.NewString() },
		"reference suffix not uuid": func(r *InitiatePaymentRequest) { r.ReferenceID = "bat_123" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			store := newFakeStore()
			req := validInitiate()
			mutate(&req)

			_, err := newTestService(store).InitiatePayment(context.Background(), req)

			assert.ErrorIs(t, err, serviceErrors.ErrInvalidInitiate)
			assert.Empty(t, store.createCalls)
		})
	}
}

func TestRequestRefund_FullAmount_MarksRefunded(t *testing.T) {
	store := newFakeStore()
	ref := "res_" + uuid.NewString()
	store.add(completedPayment(ref, 45))

	resp, err := newTestService(store).RequestRefund(context.Background(), RefundRequest{
		ReferenceID: ref, Amount: 45, Currency: "TRY",
	})

	require.NoError(t, err)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, db.PaymentPaymentStatusEnumRefunded, store.byRef[ref].Status)
}

func TestRequestRefund_OneMealOfBatch_StaysCompleted(t *testing.T) {
	store := newFakeStore()
	ref := "bat_" + uuid.NewString()
	store.add(completedPayment(ref, 90))

	_, err := newTestService(store).RequestRefund(context.Background(), RefundRequest{
		ReferenceID: ref, Amount: 45, Currency: "TRY",
	})

	require.NoError(t, err)
	assert.Equal(t, db.PaymentPaymentStatusEnumCompleted, store.byRef[ref].Status)
}

func TestRequestRefund_Rejected_ReturnsReason(t *testing.T) {
	ref := "res_" + uuid.NewString()
	pending := completedPayment(ref, 45)
	pending.Status = db.PaymentPaymentStatusEnumPending

	cases := []struct {
		name    string
		payment *db.Payment
		amount  float64
		want    error
	}{
		{"unknown reference", nil, 45, serviceErrors.ErrPaymentNotFound},
		{"payment not completed", &pending, 45, serviceErrors.ErrRefundNotAllowed},
		{"more than was paid", ptr(completedPayment(ref, 45)), 50, serviceErrors.ErrRefundTooLarge},
		{"non-positive amount", ptr(completedPayment(ref, 45)), 0, sharedErrors.ErrValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeStore()
			if tc.payment != nil {
				store.add(*tc.payment)
			}

			_, err := newTestService(store).RequestRefund(context.Background(), RefundRequest{
				ReferenceID: ref, Amount: tc.amount, Currency: "TRY",
			})

			assert.True(t, errors.Is(err, tc.want), "got %v, want %v", err, tc.want)
		})
	}
}

func TestExpireOverdue_PendingPastDeadline_ExpiresOnServiceClock(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	overdue := completedPayment("res_"+uuid.NewString(), 45)
	overdue.Status = db.PaymentPaymentStatusEnumPending
	overdue.ExpiresAt = utils.TimeToPgTimestamptz(fixedNow.Add(-time.Second))
	store.add(overdue)
	open := completedPayment("res_"+uuid.NewString(), 45)
	open.Status = db.PaymentPaymentStatusEnumPending
	open.ExpiresAt = utils.TimeToPgTimestamptz(fixedNow.Add(time.Minute))
	store.add(open)

	n, err := newTestService(store).ExpireOverdue(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.Equal(t, fixedNow, store.expireNow)
	assert.Equal(t, db.PaymentPaymentStatusEnumExpired, store.byRef[overdue.ReferenceID].Status)
	assert.Equal(t, db.PaymentPaymentStatusEnumPending, store.byRef[open.ReferenceID].Status)
}

func ptr[T any](v T) *T { return &v }
