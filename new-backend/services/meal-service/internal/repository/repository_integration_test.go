//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/meal/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mealDB struct {
	pool         *pgxpool.Pool
	reservations *ReservationRepository
	student      uuid.UUID
	cafeteria    uuid.UUID
	day          time.Time
}

func newMealDB(t *testing.T) *mealDB {
	t.Helper()
	ctx := context.Background()
	pool := pgtest.New(t, "meal", "../sql/migrations")
	m := &mealDB{pool: pool, reservations: NewReservationRepository(pool), student: uuid.New(),
		day: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)}

	_, err := NewStudentCacheRepository(pool).UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID: utils.UUIDToPgtype(m.student), StudentNumber: "2026001", FirstName: "Zeynep", LastName: "Şahin", IsActive: true,
	})
	require.NoError(t, err)
	caf, err := NewCafeteriaRepository(pool).CreateCafeteria(ctx, db.CreateCafeteriaParams{
		Name: "Merkez Yemekhane", Location: "Kampüs", ServesDinner: true, IsActive: true,
	})
	require.NoError(t, err)
	m.cafeteria = utils.PgtypeToUUID(caf.ID)
	return m
}

func (m *mealDB) slot(day time.Time, status db.ReservationStatusEnum, expiresAt time.Time) db.CreateReservationParams {
	return db.CreateReservationParams{
		StudentID: utils.UUIDToPgtype(m.student), CafeteriaID: utils.UUIDToPgtype(m.cafeteria),
		ReservationDate: pgtype.Date{Time: day, Valid: true}, MealTime: db.MealMealTimeEnumLunch,
		MenuType: db.MealMenuTypeEnumNormal, Status: status,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: !expiresAt.IsZero()},
	}
}

func (m *mealDB) status(t *testing.T, id pgtype.UUID) db.ReservationStatusEnum {
	t.Helper()
	r, err := m.reservations.GetReservationByID(context.Background(), utils.PgtypeToUUID(id))
	require.NoError(t, err)
	return r.Status
}

func (m *mealDB) outboxCount(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, m.pool.QueryRow(context.Background(), "SELECT count(*) FROM meal.outbox_events").Scan(&n))
	return n
}

// requireUniqueViolation is the contract the reservation service relies on:
// it tells a lost booking race (409) from a real failure (500) by finding
// the Postgres error inside the repository's error.
func requireUniqueViolation(t *testing.T, err error) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code)
	assert.ErrorIs(t, err, sharedErrors.ErrQueryFailed)
}

func TestReservationRepository_SecondActiveBooking_SurfacesUniqueViolation(t *testing.T) {
	m := newMealDB(t)
	ctx := context.Background()
	first, err := m.reservations.CreateReservation(ctx, m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))
	require.NoError(t, err)

	_, err = m.reservations.CreateReservation(ctx, m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))

	requireUniqueViolation(t, err)

	_, err = m.reservations.UpdateReservationByID(ctx, db.UpdateReservationByIDParams{ID: first.ID, Status: db.MealReservationStatusEnumCancelled})
	require.NoError(t, err)
	_, err = m.reservations.CreateReservation(ctx, m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))
	assert.NoError(t, err, "a cancelled reservation no longer holds the slot")
}

func TestReservationRepository_BatchWithTakenSlot_BooksNothing(t *testing.T) {
	m := newMealDB(t)
	ctx := context.Background()
	taken := m.day.AddDate(0, 0, 2)
	_, err := m.reservations.CreateReservation(ctx, m.slot(taken, db.MealReservationStatusEnumConfirmed, time.Time{}))
	require.NoError(t, err)

	_, err = m.reservations.CreateBatchReservations(ctx, []db.CreateReservationParams{
		m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)),
		m.slot(m.day.AddDate(0, 0, 1), db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)),
		m.slot(taken, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)),
	})

	requireUniqueViolation(t, err)
	rows, err := m.reservations.GetStudentReservations(ctx, m.student)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "the days before the taken one were rolled back")
}

func TestReservationRepository_ExpiryFreesTheSlotAndCleanupRemovesOldRows(t *testing.T) {
	m := newMealDB(t)
	ctx := context.Background()
	lapsed, err := m.reservations.CreateReservation(ctx, m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(-time.Minute)))
	require.NoError(t, err)
	waiting, err := m.reservations.CreateReservation(ctx, m.slot(m.day.AddDate(0, 0, 1), db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))
	require.NoError(t, err)
	stale, err := m.reservations.CreateReservation(ctx, m.slot(m.day.AddDate(0, 0, 2), db.MealReservationStatusEnumExpired, time.Now().AddDate(0, 0, -8)))
	require.NoError(t, err)

	require.NoError(t, m.reservations.ExpirePendingReservations(ctx, 100))

	assert.Equal(t, db.MealReservationStatusEnumExpired, m.status(t, lapsed.ID))
	assert.Equal(t, db.MealReservationStatusEnumPending, m.status(t, waiting.ID), "an unpaid booking keeps its slot until it lapses")
	_, err = m.reservations.CreateReservation(ctx, m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))
	assert.NoError(t, err, "an expired reservation no longer holds the slot")

	require.NoError(t, m.reservations.CleanupExpiredReservations(ctx, 100))
	_, err = m.reservations.GetReservationByID(ctx, utils.PgtypeToUUID(stale.ID))
	assert.ErrorIs(t, err, serviceErrors.ErrReservationNotFoundRepo, "expired over a week ago")
	assert.Equal(t, db.MealReservationStatusEnumExpired, m.status(t, lapsed.ID), "expired just now: kept for the student's history")
}

// A confirmed meal is either eaten or refunded, once. The guards live in the
// UPDATE statements, so only a real database shows that the losing call
// finds no row and that a refused cancel leaves no refund event behind.
func TestReservationRepository_UseAndCancel_EachOnlyOnce(t *testing.T) {
	m := newMealDB(t)
	ctx := context.Background()
	eaten, err := m.reservations.CreateReservation(ctx, m.slot(m.day, db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))
	require.NoError(t, err)
	refunded, err := m.reservations.CreateReservation(ctx, m.slot(m.day.AddDate(0, 0, 1), db.MealReservationStatusEnumPending, time.Now().Add(time.Hour)))
	require.NoError(t, err)
	ids := []uuid.UUID{utils.PgtypeToUUID(eaten.ID), utils.PgtypeToUUID(refunded.ID)}
	require.NoError(t, m.reservations.ConfirmReservationsWithEvents(ctx, ids, []map[string]any{{"n": 1}, {"n": 2}}))
	assert.Equal(t, 2, m.outboxCount(t), "one created event per confirmed reservation")

	used, err := m.reservations.MarkReservationUsed(ctx, ids[0])
	require.NoError(t, err)
	assert.True(t, used.IsUsed)
	assert.False(t, used.ExpiresAt.Valid, "confirming cleared the payment deadline")
	_, err = m.reservations.MarkReservationUsed(ctx, ids[0])
	assert.ErrorIs(t, err, serviceErrors.ErrReservationNotFoundRepo, "a QR cannot be scanned twice")
	_, err = m.reservations.CancelReservationWithRefund(ctx, ids[0], map[string]any{})
	assert.ErrorIs(t, err, serviceErrors.ErrInvalidStatusForCancel, "an eaten meal is not refunded")
	assert.Equal(t, 2, m.outboxCount(t))

	cancelled, err := m.reservations.CancelReservationWithRefund(ctx, ids[1], map[string]any{"amount": 40})
	require.NoError(t, err)
	assert.Equal(t, db.MealReservationStatusEnumCancelled, cancelled.Status)
	_, err = m.reservations.CancelReservationWithRefund(ctx, ids[1], map[string]any{"amount": 40})
	assert.ErrorIs(t, err, serviceErrors.ErrInvalidStatusForCancel)
	_, err = m.reservations.MarkReservationUsed(ctx, ids[1])
	assert.ErrorIs(t, err, serviceErrors.ErrReservationNotFoundRepo, "a refunded meal cannot be eaten")
	assert.Equal(t, 3, m.outboxCount(t), "exactly one refund event")
}
