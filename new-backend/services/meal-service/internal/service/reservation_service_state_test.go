package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/meal/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// raceStore mimics two requests that both read the reservation before either
// writes: reads always return the original confirmed, unused row, while the
// writes apply the same state guard as the SQL UPDATEs.
type raceStore struct {
	ReservationStore
	reservation db.GetReservationByIDRow
	used        bool
	cancelled   bool
}

func (f *raceStore) FindReservationForQR(context.Context, db.FindReservationForQRParams) (db.FindReservationForQRRow, error) {
	return db.FindReservationForQRRow(f.reservation), nil
}

func (f *raceStore) GetReservationByID(context.Context, uuid.UUID) (db.GetReservationByIDRow, error) {
	return f.reservation, nil
}

func (f *raceStore) MarkReservationUsed(context.Context, uuid.UUID) (db.Reservation, error) {
	if f.used || f.cancelled {
		return db.Reservation{}, serviceErrors.ErrReservationNotFoundRepo
	}
	f.used = true
	return db.Reservation{}, nil
}

func (f *raceStore) CancelReservationWithRefund(context.Context, uuid.UUID, map[string]any) (db.Reservation, error) {
	if f.used || f.cancelled {
		return db.Reservation{}, serviceErrors.ErrInvalidStatusForCancel
	}
	f.cancelled = true
	return db.Reservation{}, nil
}

type activeStudent struct{ id uuid.UUID }

func (s activeStudent) GetStudentCacheByID(context.Context, uuid.UUID) (db.StudentView, error) {
	return db.StudentView{ID: utils.UUIDToPgtype(s.id), StudentNumber: "20260001", IsActive: true}, nil
}

type countingPayment struct {
	refunds    int
	lastRefund dto.RefundRequest
}

func (p *countingPayment) InitiatePayment(context.Context, dto.InitiatePaymentRequest) (*dto.InitiatePaymentResponse, error) {
	return &dto.InitiatePaymentResponse{}, nil
}

func (p *countingPayment) RequestRefund(_ context.Context, req dto.RefundRequest) (*dto.RefundResponse, error) {
	p.refunds++
	p.lastRefund = req
	return &dto.RefundResponse{Status: "completed"}, nil
}

func newRaceFixture(t *testing.T, reservationDate time.Time) (*ReservationService, *raceStore, *countingPayment, uuid.UUID) {
	t.Helper()
	studentID := uuid.New()
	store := &raceStore{reservation: db.GetReservationByIDRow{
		ID:              utils.UUIDToPgtype(uuid.New()),
		StudentID:       utils.UUIDToPgtype(studentID),
		CafeteriaID:     utils.UUIDToPgtype(uuid.New()),
		ReservationDate: pgtype.Date{Time: reservationDate, Valid: true},
		MealTime:        db.MealMealTimeEnumLunch,
		MenuType:        db.MealMenuTypeEnumNormal,
		Status:          db.MealReservationStatusEnumConfirmed,
		CafeteriaName:   "Merkez Yemekhane",
	}}
	payment := &countingPayment{}
	svc := &ReservationService{
		reservationRepo:  store,
		studentCacheRepo: activeStudent{id: studentID},
		paymentClient:    payment,
		cfg:              defaultCfg(),
		logger:           zap.NewNop(),
	}
	return svc, store, payment, studentID
}

func TestUseReservation_SecondScan_ReturnsAlreadyUsed(t *testing.T) {
	freezeAt(t, 2026, time.May, 4, 12, 0)
	today := time.Date(2026, time.May, 4, 0, 0, 0, 0, time.UTC)
	svc, store, _, studentID := newRaceFixture(t, today)

	qr := svc.generateQRPayload(utils.PgtypeToUUID(store.reservation.CafeteriaID).String(), "2026-05-04", "lunch")
	req := dto.UseReservationRequest{QRPayload: qr}

	_, err := svc.UseReservation(context.Background(), studentID, req)
	require.NoError(t, err)

	_, err = svc.UseReservation(context.Background(), studentID, req)
	assert.ErrorIs(t, err, serviceErrors.ErrReservationAlreadyUsed)
}

func TestCancelReservation_SecondCancel_ReturnsInvalidStatusWithoutRefund(t *testing.T) {
	// Monday; the reservation is next week, well before its Friday cut-off.
	freezeAt(t, 2026, time.May, 4, 10, 0)
	svc, store, payment, studentID := newRaceFixture(t, time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC))
	resID := utils.PgtypeToUUID(store.reservation.ID).String()

	_, err := svc.CancelReservation(context.Background(), studentID, resID)
	require.NoError(t, err)

	_, err = svc.CancelReservation(context.Background(), studentID, resID)
	assert.ErrorIs(t, err, serviceErrors.ErrInvalidStatusForCancel)
	assert.Equal(t, 1, payment.refunds, "a rejected cancel must not request a refund")
}

func TestCancelReservation_AfterUse_ReturnsInvalidStatusWithoutRefund(t *testing.T) {
	freezeAt(t, 2026, time.May, 4, 10, 0)
	svc, store, payment, studentID := newRaceFixture(t, time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC))
	store.used = true

	_, err := svc.CancelReservation(context.Background(), studentID, utils.PgtypeToUUID(store.reservation.ID).String())
	assert.ErrorIs(t, err, serviceErrors.ErrInvalidStatusForCancel)
	assert.Zero(t, payment.refunds)
}

// Payment keys a refund by the reference the reservation was paid under; a
// batch member refunded under its own id would find no payment.
func TestCancelReservation_RefundsUnderPaymentReference(t *testing.T) {
	batchID := uuid.New()
	cases := []struct {
		name    string
		batchID pgtype.UUID
		want    func(resID string) string
	}{
		{"single reservation", pgtype.UUID{}, func(resID string) string { return "res_" + resID }},
		{"batch member", utils.UUIDToPgtype(batchID), func(string) string { return "bat_" + batchID.String() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			freezeAt(t, 2026, time.May, 4, 10, 0)
			svc, store, payment, studentID := newRaceFixture(t, time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC))
			store.reservation.BatchID = tc.batchID
			resID := utils.PgtypeToUUID(store.reservation.ID).String()

			_, err := svc.CancelReservation(context.Background(), studentID, resID)

			require.NoError(t, err)
			assert.Equal(t, tc.want(resID), payment.lastRefund.ReferenceID)
		})
	}
}
