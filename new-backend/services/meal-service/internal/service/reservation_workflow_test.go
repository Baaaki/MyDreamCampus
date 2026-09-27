package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/meal/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mealWorld is an in-memory stand-in for the meal database and the payment
// service. Inserts honour the partial unique index on (student, date, meal
// time) over pending and confirmed rows, as Postgres does.
type mealWorld struct {
	ReservationStore
	students     map[uuid.UUID]db.StudentView
	cafeterias   map[uuid.UUID]db.Cafeteria
	closed       map[string]bool
	reservations []db.Reservation
	payments     []dto.InitiatePaymentRequest
	paymentErr   error
	// hideActive makes the pre-insert conflict check miss, as when two
	// requests race past it and only the unique index is left to decide.
	hideActive bool
}

func (w *mealWorld) GetStudentCacheByID(_ context.Context, id uuid.UUID) (db.StudentView, error) {
	s, ok := w.students[id]
	if !ok {
		return s, fmt.Errorf("%w: student %s", sharedErrors.ErrNotFoundRepo, id)
	}
	return s, nil
}

func (w *mealWorld) GetCafeteriaByID(_ context.Context, id uuid.UUID) (db.Cafeteria, error) {
	c, ok := w.cafeterias[id]
	if !ok {
		return c, fmt.Errorf("%w: cafeteria %s", serviceErrors.ErrCafeteriaNotFoundRepo, id)
	}
	return c, nil
}

func (w *mealWorld) IsDateClosed(_ context.Context, d pgtype.Date) (bool, error) {
	return w.closed[d.Time.Format("2006-01-02")], nil
}

func (w *mealWorld) InitiatePayment(_ context.Context, req dto.InitiatePaymentRequest) (*dto.InitiatePaymentResponse, error) {
	if w.paymentErr != nil {
		return nil, w.paymentErr
	}
	w.payments = append(w.payments, req)
	return &dto.InitiatePaymentResponse{PaymentID: fmt.Sprintf("pay_%d", len(w.payments)), Amount: req.Amount, Currency: req.Currency}, nil
}

func (w *mealWorld) RequestRefund(context.Context, dto.RefundRequest) (*dto.RefundResponse, error) {
	return &dto.RefundResponse{}, nil
}

func isActive(r db.Reservation) bool {
	return r.Status == db.MealReservationStatusEnumPending || r.Status == db.MealReservationStatusEnumConfirmed
}

func (w *mealWorld) activeFor(student pgtype.UUID, date pgtype.Date, meal db.MealTimeEnum) *db.Reservation {
	for i, r := range w.reservations {
		if isActive(r) && r.StudentID == student && r.ReservationDate.Time.Equal(date.Time) && r.MealTime == meal {
			return &w.reservations[i]
		}
	}
	return nil
}

func (w *mealWorld) CheckActiveReservation(_ context.Context, p db.CheckActiveReservationParams) (*db.CheckActiveReservationRow, error) {
	if r := w.activeFor(p.StudentID, p.ReservationDate, p.MealTime); r != nil && !w.hideActive {
		return &db.CheckActiveReservationRow{ID: r.ID, Status: r.Status}, nil
	}
	return nil, nil
}

func (w *mealWorld) CheckActiveReservationsForSlots(_ context.Context, p db.CheckActiveReservationsForSlotsParams) ([]db.CheckActiveReservationsForSlotsRow, error) {
	var out []db.CheckActiveReservationsForSlotsRow
	for i := range p.Dates {
		if r := w.activeFor(p.StudentID, p.Dates[i], db.MealTimeEnum(p.MealTimes[i])); r != nil && !w.hideActive {
			out = append(out, db.CheckActiveReservationsForSlotsRow{ID: r.ID, ReservationDate: r.ReservationDate, MealTime: r.MealTime, Status: r.Status})
		}
	}
	return out, nil
}

func (w *mealWorld) newRow(p db.CreateReservationParams) db.Reservation {
	return db.Reservation{
		ID: utils.UUIDToPgtype(uuid.New()), BatchID: p.BatchID, StudentID: p.StudentID, CafeteriaID: p.CafeteriaID,
		ReservationDate: p.ReservationDate, MealTime: p.MealTime, MenuType: p.MenuType, Status: p.Status,
		ExpiresAt: p.ExpiresAt, CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

var uniqueViolation = &pgconn.PgError{Code: "23505"}

func (w *mealWorld) CreateReservation(_ context.Context, p db.CreateReservationParams) (db.Reservation, error) {
	if w.activeFor(p.StudentID, p.ReservationDate, p.MealTime) != nil {
		return db.Reservation{}, uniqueViolation
	}
	r := w.newRow(p)
	w.reservations = append(w.reservations, r)
	return r, nil
}

// CreateBatchReservations is one transaction: every row or none.
func (w *mealWorld) CreateBatchReservations(_ context.Context, ps []db.CreateReservationParams) ([]db.Reservation, error) {
	staged := append([]db.Reservation(nil), w.reservations...)
	var created []db.Reservation
	for _, p := range ps {
		for _, r := range staged {
			if isActive(r) && r.StudentID == p.StudentID && r.ReservationDate.Time.Equal(p.ReservationDate.Time) && r.MealTime == p.MealTime {
				return nil, uniqueViolation
			}
		}
		r := w.newRow(p)
		staged = append(staged, r)
		created = append(created, r)
	}
	w.reservations = staged
	return created, nil
}

func (w *mealWorld) UpdateReservationByID(_ context.Context, p db.UpdateReservationByIDParams) (db.Reservation, error) {
	for i, r := range w.reservations {
		if r.ID == p.ID {
			w.reservations[i].Status, w.reservations[i].ExpiresAt = p.Status, p.ExpiresAt
			return w.reservations[i], nil
		}
	}
	return db.Reservation{}, serviceErrors.ErrReservationNotFoundRepo
}

func (w *mealWorld) UpdateReservationsByBatchID(_ context.Context, p db.UpdateReservationsByBatchIDParams) error {
	for i, r := range w.reservations {
		if r.BatchID == p.BatchID {
			w.reservations[i].Status, w.reservations[i].ExpiresAt = p.Status, p.ExpiresAt
		}
	}
	return nil
}

func (w *mealWorld) filtered(student pgtype.UUID, status db.NullReservationStatusEnum) []db.GetStudentReservationsFilteredRow {
	var out []db.GetStudentReservationsFilteredRow
	for _, r := range w.reservations {
		if r.StudentID != student || (status.Valid && r.Status != status.ReservationStatusEnum) {
			continue
		}
		c := w.cafeterias[utils.PgtypeToUUID(r.CafeteriaID)]
		out = append(out, db.GetStudentReservationsFilteredRow{
			ID: r.ID, CafeteriaID: r.CafeteriaID, ReservationDate: r.ReservationDate, MealTime: r.MealTime,
			MenuType: r.MenuType, Status: r.Status, IsUsed: r.IsUsed, CafeteriaName: c.Name, CafeteriaLocation: c.Location,
		})
	}
	return out
}

func (w *mealWorld) GetStudentReservationsFiltered(_ context.Context, p db.GetStudentReservationsFilteredParams) ([]db.GetStudentReservationsFilteredRow, error) {
	rows := w.filtered(p.StudentID, p.Status)
	if limit, _ := p.Column4.(int32); limit > 0 {
		end := min(int(p.Offset+limit), len(rows))
		rows = rows[min(int(p.Offset), len(rows)):end]
	}
	return rows, nil
}

func (w *mealWorld) CountStudentReservationsFiltered(_ context.Context, p db.CountStudentReservationsFilteredParams) (int64, error) {
	return int64(len(w.filtered(p.StudentID, p.Status))), nil
}

func (w *mealWorld) service() *ReservationService {
	cfg := defaultCfg()
	cfg.Reservation.TimeoutMinutes = 15
	cfg.Reservation.MealPriceTRY = 40
	return NewReservationService(w, w, w, w, w, cfg, zap.NewNop())
}

// newMealWorld freezes the clock on Monday 5 October 2026, 09:00 UTC+3,
// with one active student, a lunch-only cafeteria and a full one.
func newMealWorld(t *testing.T) (w *mealWorld, student, lunchOnly, full uuid.UUID) {
	t.Helper()
	freezeAt(t, 2026, time.October, 5, 9, 0)
	student, lunchOnly, full = uuid.New(), uuid.New(), uuid.New()
	w = &mealWorld{
		students: map[uuid.UUID]db.StudentView{student: {ID: utils.UUIDToPgtype(student), StudentNumber: "20240001", IsActive: true}},
		cafeterias: map[uuid.UUID]db.Cafeteria{
			lunchOnly: {ID: utils.UUIDToPgtype(lunchOnly), Name: "Merkez", IsActive: true},
			full:      {ID: utils.UUIDToPgtype(full), Name: "Mühendislik", HasVeganMenu: true, ServesDinner: true, IsActive: true},
		},
		closed: map[string]bool{"2026-10-29": true},
	}
	return w, student, lunchOnly, full
}

func booking(cafeteria uuid.UUID, date, meal, menu string) dto.CreateReservationRequest {
	return dto.CreateReservationRequest{CafeteriaID: cafeteria.String(), Date: date, MealTime: meal, MenuType: menu}
}

// ---------------------------------------------------------- CreateReservation

func TestCreateReservation_Valid_PendingRowAndPaymentStarted(t *testing.T) {
	w, student, _, full := newMealWorld(t)

	resp, err := w.service().CreateReservation(context.Background(), student, booking(full, "2026-10-06", "dinner", "vegan"))

	require.NoError(t, err)
	require.Len(t, w.reservations, 1)
	r := w.reservations[0]
	assert.Equal(t, db.MealReservationStatusEnumPending, r.Status, "confirmed only by payment.completed")
	assert.Equal(t, time.Date(2026, 10, 5, 9, 15, 0, 0, utcPlus3).Unix(), r.ExpiresAt.Time.Unix(), "held for the payment timeout")
	require.Len(t, w.payments, 1)
	assert.Equal(t, "res_"+r.ID.String(), w.payments[0].ReferenceID, "refunds find the payment by this reference")
	assert.InDelta(t, 40, w.payments[0].Amount, 0.001)
	assert.Equal(t, "pay_1", resp.PaymentID)
	assert.Equal(t, "Mühendislik", resp.Reservation.CafeteriaName)
}

func TestCreateReservation_DateBoundaries(t *testing.T) {
	cases := []struct {
		date string
		want error
	}{
		{"2026-10-05", nil},
		{"2026-11-04", nil},
		{"2026-10-04", serviceErrors.ErrInvalidDateRange},
		{"2026-11-05", serviceErrors.ErrInvalidDateRange},
		{"2026-10-29", serviceErrors.ErrCafeteriaClosedOnDate},
		{"05.10.2026", sharedErrors.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.date, func(t *testing.T) {
			w, student, lunchOnly, _ := newMealWorld(t)
			_, err := w.service().CreateReservation(context.Background(), student, booking(lunchOnly, tc.date, "lunch", "normal"))
			if tc.want == nil {
				assert.NoError(t, err, "today and 30 days ahead are bookable")
				return
			}
			assert.ErrorIs(t, err, tc.want)
			assert.Empty(t, w.reservations)
		})
	}
}

func TestCreateReservation_Refused(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *mealWorld, student, lunchOnly, full uuid.UUID) (uuid.UUID, dto.CreateReservationRequest)
		want  error
	}{
		{"deactivated student", func(w *mealWorld, s, l, _ uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			st := w.students[s]
			st.IsActive = false
			w.students[s] = st
			return s, booking(l, "2026-10-06", "lunch", "normal")
		}, serviceErrors.ErrStudentDeactivated},
		{"unknown student", func(_ *mealWorld, _, l, _ uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			return uuid.New(), booking(l, "2026-10-06", "lunch", "normal")
		}, sharedErrors.ErrNotFound},
		{"unknown cafeteria", func(_ *mealWorld, s, _, _ uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			return s, booking(uuid.New(), "2026-10-06", "lunch", "normal")
		}, serviceErrors.ErrCafeteriaNotFound},
		{"closed cafeteria", func(w *mealWorld, s, l, _ uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			c := w.cafeterias[l]
			c.IsActive = false
			w.cafeterias[l] = c
			return s, booking(l, "2026-10-06", "lunch", "normal")
		}, serviceErrors.ErrCafeteriaNotActive},
		{"dinner where only lunch is served", func(_ *mealWorld, s, l, _ uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			return s, booking(l, "2026-10-06", "dinner", "normal")
		}, serviceErrors.ErrCafeteriaNoDinner},
		{"vegan where there is no vegan menu", func(_ *mealWorld, s, l, _ uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			return s, booking(l, "2026-10-06", "lunch", "vegan")
		}, serviceErrors.ErrCafeteriaNoVegan},
		{"second lunch the same day", func(w *mealWorld, s, l, f uuid.UUID) (uuid.UUID, dto.CreateReservationRequest) {
			_, err := w.service().CreateReservation(context.Background(), s, booking(f, "2026-10-06", "lunch", "normal"))
			require.NoError(t, err)
			return s, booking(l, "2026-10-06", "lunch", "normal")
		}, serviceErrors.ErrActiveReservationExists},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, student, lunchOnly, full := newMealWorld(t)
			caller, req := tc.setup(w, student, lunchOnly, full)
			rows, payments := len(w.reservations), len(w.payments)

			_, err := w.service().CreateReservation(context.Background(), caller, req)

			assert.ErrorIs(t, err, tc.want)
			assert.Len(t, w.reservations, rows, "nothing is booked")
			assert.Len(t, w.payments, payments, "nothing is charged")
		})
	}
}

func TestCreateReservation_LostRaceOnUniqueIndex_ReportsConflict(t *testing.T) {
	w, student, _, full := newMealWorld(t)
	_, err := w.service().CreateReservation(context.Background(), student, booking(full, "2026-10-06", "lunch", "normal"))
	require.NoError(t, err)
	w.hideActive = true

	_, err = w.service().CreateReservation(context.Background(), student, booking(full, "2026-10-06", "lunch", "vegan"))

	assert.ErrorIs(t, err, serviceErrors.ErrActiveReservationExists)
	assert.Len(t, w.payments, 1, "the losing request starts no payment")
}

func TestCreateReservation_PaymentUnavailable_CancelsThePendingRow(t *testing.T) {
	w, student, _, full := newMealWorld(t)
	w.paymentErr = errors.New("payment-service: 503")

	_, err := w.service().CreateReservation(context.Background(), student, booking(full, "2026-10-06", "lunch", "normal"))

	require.Error(t, err)
	require.Len(t, w.reservations, 1)
	assert.Equal(t, db.MealReservationStatusEnumCancelled, w.reservations[0].Status, "the slot is freed for a retry")

	w.paymentErr = nil
	_, err = w.service().CreateReservation(context.Background(), student, booking(full, "2026-10-06", "lunch", "normal"))
	assert.NoError(t, err)
}

// ----------------------------------------------------- CreateBatchReservation

func TestCreateBatchReservation_Valid_OnePaymentForTheWholeBatch(t *testing.T) {
	w, student, lunchOnly, full := newMealWorld(t)

	resp, err := w.service().CreateBatchReservation(context.Background(), student, dto.BatchReservationRequest{Reservations: []dto.CreateReservationRequest{
		booking(lunchOnly, "2026-10-06", "lunch", "normal"),
		booking(full, "2026-10-06", "dinner", "vegan"),
		booking(lunchOnly, "2026-10-07", "lunch", "normal"),
	}})

	require.NoError(t, err)
	require.Len(t, w.reservations, 3)
	for _, r := range w.reservations {
		assert.Equal(t, resp.BatchID, r.BatchID.String(), "every row carries the batch id")
		assert.Equal(t, db.MealReservationStatusEnumPending, r.Status)
	}
	require.Len(t, w.payments, 1)
	assert.Equal(t, "bat_"+resp.BatchID, w.payments[0].ReferenceID)
	assert.InDelta(t, 120, w.payments[0].Amount, 0.001)
	assert.InDelta(t, 120, resp.TotalAmount, 0.001)
	require.Len(t, resp.Reservations, 3)
	assert.Equal(t, "Mühendislik", resp.Reservations[1].CafeteriaName)
}

func TestCreateBatchReservation_Refused_WritesNothing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *mealWorld, student uuid.UUID)
		items func(lunchOnly, full uuid.UUID) []dto.CreateReservationRequest
		want  error
	}{
		{"one invalid entry", nil, func(l, f uuid.UUID) []dto.CreateReservationRequest {
			return []dto.CreateReservationRequest{booking(l, "2026-10-06", "lunch", "normal"), booking(l, "2026-10-06", "dinner", "normal")}
		}, serviceErrors.ErrValidationErrors},
		{"a closed day", nil, func(l, _ uuid.UUID) []dto.CreateReservationRequest {
			return []dto.CreateReservationRequest{booking(l, "2026-10-28", "lunch", "normal"), booking(l, "2026-10-29", "lunch", "normal")}
		}, serviceErrors.ErrValidationErrors},
		{"clash with an existing reservation", func(w *mealWorld, s uuid.UUID) {
			w.reservations = append(w.reservations, db.Reservation{
				ID: utils.UUIDToPgtype(uuid.New()), StudentID: utils.UUIDToPgtype(s), MealTime: db.MealMealTimeEnumLunch,
				ReservationDate: pgtype.Date{Time: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC), Valid: true},
				Status:          db.MealReservationStatusEnumConfirmed,
			})
		}, func(l, _ uuid.UUID) []dto.CreateReservationRequest {
			return []dto.CreateReservationRequest{booking(l, "2026-10-06", "lunch", "normal"), booking(l, "2026-10-07", "lunch", "normal")}
		}, serviceErrors.ErrReservationConflicts},
		{"the same slot twice in one batch", nil, func(l, f uuid.UUID) []dto.CreateReservationRequest {
			return []dto.CreateReservationRequest{booking(l, "2026-10-06", "lunch", "normal"), booking(f, "2026-10-06", "lunch", "vegan")}
		}, serviceErrors.ErrReservationConflicts},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, student, lunchOnly, full := newMealWorld(t)
			if tc.setup != nil {
				tc.setup(w, student)
			}
			before := len(w.reservations)

			_, err := w.service().CreateBatchReservation(context.Background(), student, dto.BatchReservationRequest{Reservations: tc.items(lunchOnly, full)})

			assert.ErrorIs(t, err, tc.want)
			assert.Len(t, w.reservations, before)
			assert.Empty(t, w.payments)
		})
	}
}

func TestCreateBatchReservation_PaymentUnavailable_CancelsTheBatch(t *testing.T) {
	w, student, lunchOnly, _ := newMealWorld(t)
	w.paymentErr = errors.New("payment-service: timeout")

	_, err := w.service().CreateBatchReservation(context.Background(), student, dto.BatchReservationRequest{Reservations: []dto.CreateReservationRequest{
		booking(lunchOnly, "2026-10-06", "lunch", "normal"), booking(lunchOnly, "2026-10-07", "lunch", "normal"),
	}})

	require.Error(t, err)
	require.Len(t, w.reservations, 2)
	for _, r := range w.reservations {
		assert.Equal(t, db.MealReservationStatusEnumCancelled, r.Status)
	}
}

// ---------------------------------------------------------- GetMyReservations

func TestGetMyReservations_SummaryAndPages(t *testing.T) {
	w, student, lunchOnly, full := newMealWorld(t)
	svc := w.service()
	for _, d := range []string{"2026-10-06", "2026-10-07", "2026-10-08"} {
		_, err := svc.CreateReservation(context.Background(), student, booking(lunchOnly, d, "lunch", "normal"))
		require.NoError(t, err)
	}
	_, err := svc.CreateReservation(context.Background(), student, booking(full, "2026-10-06", "dinner", "normal"))
	require.NoError(t, err)
	w.reservations[0].Status, w.reservations[0].IsUsed = db.MealReservationStatusEnumConfirmed, true
	w.reservations[1].Status = db.MealReservationStatusEnumConfirmed
	w.reservations[2].Status = db.MealReservationStatusEnumCancelled
	w.reservations = append(w.reservations, db.Reservation{StudentID: utils.UUIDToPgtype(uuid.New()), Status: db.MealReservationStatusEnumConfirmed})

	all, err := svc.GetMyReservations(context.Background(), student, dto.MyReservationsQuery{})
	require.NoError(t, err)
	assert.Equal(t, dto.ReservationSummary{Total: 4, Confirmed: 2, Pending: 1, Used: 1, Cancelled: 1}, all.Summary, "another student's row is not counted")
	assert.Equal(t, "Merkez", all.Reservations[0].Cafeteria.Name)

	page, err := svc.GetMyReservations(context.Background(), student, dto.MyReservationsQuery{Page: 2, Limit: 3})
	require.NoError(t, err)
	assert.Len(t, page.Reservations, 1)
	require.NotNil(t, page.Pagination)
	assert.Equal(t, 2, page.Pagination.TotalPages)
	assert.Equal(t, 4, page.Pagination.TotalItems)

	_, err = svc.GetMyReservations(context.Background(), student, dto.MyReservationsQuery{FromDate: "6 Ekim"})
	assert.ErrorIs(t, err, sharedErrors.ErrBadRequest)
}

// ----------------------------------------------------------------- GenerateQR

func TestGenerateQR_DefaultsToTodayAndTheMealWindow(t *testing.T) {
	w, _, _, full := newMealWorld(t)
	svc := w.service()

	qr, err := svc.GenerateQR(context.Background(), full.String(), "", "lunch")
	require.NoError(t, err)
	assert.Equal(t, "2026-10-05", qr.Date)
	assert.Equal(t, dto.ValidTimeWindow{Start: "11:00", End: "13:00"}, qr.ValidTimeWindow)
	assert.Equal(t, "Mühendislik", qr.CafeteriaName)
	assert.NotEmpty(t, qr.QRPayload)

	dinner, err := svc.GenerateQR(context.Background(), full.String(), "2026-10-06", "dinner")
	require.NoError(t, err)
	assert.Equal(t, dto.ValidTimeWindow{Start: "16:00", End: "19:00"}, dinner.ValidTimeWindow)

	_, err = svc.GenerateQR(context.Background(), uuid.NewString(), "", "lunch")
	assert.ErrorIs(t, err, serviceErrors.ErrCafeteriaNotFound)
	_, err = svc.GenerateQR(context.Background(), "kantin", "", "lunch")
	assert.ErrorIs(t, err, sharedErrors.ErrBadRequest)
}
