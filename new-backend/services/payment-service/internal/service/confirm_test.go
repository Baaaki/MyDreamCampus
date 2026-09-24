package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/card"
	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/clock/clocktest"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pendingPayment seeds a payable payment owned by the returned student.
func pendingPayment(t *testing.T, store *fakeStore) (studentID, paymentID uuid.UUID, ref string) {
	t.Helper()
	studentID, paymentID = uuid.New(), uuid.New()
	ref = "res_" + uuid.NewString()
	store.add(db.Payment{
		ID:             utils.UUIDToPgtype(paymentID),
		ReferenceID:    ref,
		StudentID:      utils.UUIDToPgtype(studentID),
		Amount:         utils.Float64ToPgNumeric(45),
		RefundedAmount: utils.Float64ToPgNumeric(0),
		Currency:       "TRY",
		Status:         db.PaymentPaymentStatusEnumPending,
		ExpiresAt:      utils.TimeToPgTimestamptz(fixedNow.Add(10 * time.Minute)),
	})
	return studentID, paymentID, ref
}

func cardInput(number string) card.Input {
	return card.Input{Number: number, ExpMonth: 12, ExpYear: 30, CVC: "123", Holder: "Zeynep Şahin"}
}

func TestConfirmPayment_SuccessCard_CompletesAndWritesEvent(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	studentID, paymentID, ref := pendingPayment(t, store)

	view, err := newTestService(store).ConfirmPayment(context.Background(), studentID, paymentID, cardInput("4242 4242 4242 4242"))

	require.NoError(t, err)
	assert.Equal(t, "completed", view.Status)
	assert.Equal(t, "Visa", *view.CardBrand)
	assert.Equal(t, "4242", *view.CardLast4)
	assert.Equal(t, fixedNow, *view.CompletedAt)
	require.Len(t, store.events, 1)
	assert.Equal(t, "payment.completed", store.events[0].eventType)
	assert.Equal(t, map[string]any{
		"payment_id":   paymentID.String(),
		"reference_id": ref,
		"amount":       45.0,
		"currency":     "TRY",
	}, store.events[0].payload)
}

func TestConfirmPayment_DeclineCards_FailWithReason(t *testing.T) {
	cases := map[string]string{
		"4000000000000002": "Kart reddedildi",
		"4000000000009995": "Yetersiz bakiye",
	}
	for number, reason := range cases {
		t.Run(reason, func(t *testing.T) {
			clocktest.Freeze(t, fixedNow)
			store := newFakeStore()
			studentID, paymentID, ref := pendingPayment(t, store)

			view, err := newTestService(store).ConfirmPayment(context.Background(), studentID, paymentID, cardInput(number))

			require.NoError(t, err)
			assert.Equal(t, "failed", view.Status)
			assert.Equal(t, reason, *view.FailureReason)
			require.Len(t, store.events, 1)
			assert.Equal(t, "payment.failed", store.events[0].eventType)
			assert.Equal(t, map[string]any{
				"payment_id":   paymentID.String(),
				"reference_id": ref,
				"reason":       reason,
			}, store.events[0].payload)
		})
	}
}

func TestConfirmPayment_InvalidCard_KeepsPaymentPayable(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	studentID, paymentID, ref := pendingPayment(t, store)
	svc := newTestService(store)

	_, err := svc.ConfirmPayment(context.Background(), studentID, paymentID, cardInput("4242424242424241"))
	require.ErrorIs(t, err, serviceErrors.ErrInvalidCardNumber)
	assert.Equal(t, db.PaymentPaymentStatusEnumPending, store.byRef[ref].Status)
	assert.Empty(t, store.events)

	view, err := svc.ConfirmPayment(context.Background(), studentID, paymentID, cardInput("4242424242424242"))
	require.NoError(t, err)
	assert.Equal(t, "completed", view.Status)
}

func TestConfirmPayment_OtherStudent_ReturnsNotFound(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	_, paymentID, _ := pendingPayment(t, store)

	_, err := newTestService(store).ConfirmPayment(context.Background(), uuid.New(), paymentID, cardInput("4242424242424242"))

	assert.ErrorIs(t, err, serviceErrors.ErrPaymentNotFound)
	assert.Empty(t, store.events)
}

func TestConfirmPayment_AlreadySettled_ReturnsConflict(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	studentID, paymentID, _ := pendingPayment(t, store)
	svc := newTestService(store)
	_, err := svc.ConfirmPayment(context.Background(), studentID, paymentID, cardInput("4242424242424242"))
	require.NoError(t, err)

	_, err = svc.ConfirmPayment(context.Background(), studentID, paymentID, cardInput("4242424242424242"))

	assert.ErrorIs(t, err, serviceErrors.ErrPaymentNotPending)
	assert.Len(t, store.events, 1)
}

func TestConfirmPayment_PastDeadline_ExpiresAndReturnsConflict(t *testing.T) {
	store := newFakeStore()
	studentID, paymentID, ref := pendingPayment(t, store)
	clocktest.Freeze(t, fixedNow.Add(10*time.Minute))

	_, err := newTestService(store).ConfirmPayment(context.Background(), studentID, paymentID, cardInput("4242424242424242"))

	assert.ErrorIs(t, err, serviceErrors.ErrPaymentExpired)
	assert.Equal(t, db.PaymentPaymentStatusEnumExpired, store.byRef[ref].Status)
	assert.Empty(t, store.events)
}

func TestGetPayment_PendingPastDeadline_ReportsExpired(t *testing.T) {
	store := newFakeStore()
	studentID, paymentID, ref := pendingPayment(t, store)
	clocktest.Freeze(t, fixedNow.Add(time.Hour))

	view, err := newTestService(store).GetPayment(context.Background(), studentID, paymentID)

	require.NoError(t, err)
	assert.Equal(t, "expired", view.Status)
	assert.Equal(t, db.PaymentPaymentStatusEnumExpired, store.byRef[ref].Status)
}

func TestGetPayment_OtherStudent_ReturnsNotFound(t *testing.T) {
	clocktest.Freeze(t, fixedNow)
	store := newFakeStore()
	_, paymentID, _ := pendingPayment(t, store)

	_, err := newTestService(store).GetPayment(context.Background(), uuid.New(), paymentID)

	assert.ErrorIs(t, err, serviceErrors.ErrPaymentNotFound)
}

// The payload is decoded by meal into dto.PaymentCompletedEventData; the
// key set is the contract.
func TestBuildPaymentCompletedPayload_MatchesMealContract(t *testing.T) {
	payload, err := buildPaymentCompletedPayload(db.Payment{
		ID:          utils.UUIDToPgtype(uuid.New()),
		ReferenceID: "bat_" + uuid.NewString(),
		Amount:      utils.Float64ToPgNumeric(90),
		Currency:    "TRY",
	})
	require.NoError(t, err)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &keys))
	assert.ElementsMatch(t, []string{"payment_id", "reference_id", "amount", "currency"}, mapKeys(keys))
}

func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
