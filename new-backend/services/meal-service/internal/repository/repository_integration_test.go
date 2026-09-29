//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
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
