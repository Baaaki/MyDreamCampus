//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fallSemester   = "2026-2027-Fall"
	springSemester = "2026-2027-Spring"
)

var windowBase = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func day(n int) time.Time { return windowBase.AddDate(0, 0, n) }

func createWindow(t *testing.T, pool *pgxpool.Pool, repo *sharedRepo.SimplePeriodRepository, semester, periodType string, start, end time.Time) (*sharedRepo.SimplePeriod, error) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	p, err := repo.CreatePeriodOfTypeTx(ctx, tx, sharedRepo.SimplePeriod{
		Semester: semester, PeriodStart: start, PeriodEnd: end, IsActive: true,
	}, periodType)
	if err != nil {
		return nil, err
	}
	require.NoError(t, tx.Commit(ctx))
	return p, nil
}

func newWindowRepo(t *testing.T) (*pgxpool.Pool, *sharedRepo.SimplePeriodRepository) {
	t.Helper()
	pool := pgtest.New(t, "course_catalog", "../sql/migrations")
	repo := sharedRepo.NewSimplePeriodRepository(pool, "course_catalog")
	_, err := createWindow(t, pool, repo, fallSemester, sharedRepo.PeriodTypeEnrollment, day(0), day(14))
	require.NoError(t, err)
	return pool, repo
}

func TestEnrollmentWindow_OverlappingSecondSemester_Refused(t *testing.T) {
	pool, repo := newWindowRepo(t)

	_, err := createWindow(t, pool, repo, springSemester, sharedRepo.PeriodTypeEnrollment, day(10), day(20))

	require.ErrorIs(t, err, sharedRepo.ErrEnrollmentWindowOverlap)
	other, err := repo.OverlappingEnrollmentSemester(context.Background(), springSemester, day(10), day(20))
	require.NoError(t, err)
	assert.Equal(t, fallSemester, other)
}

func TestEnrollmentWindow_AfterTheOpenOne_Accepted(t *testing.T) {
	pool, repo := newWindowRepo(t)

	_, err := createWindow(t, pool, repo, springSemester, sharedRepo.PeriodTypeEnrollment, day(15), day(29))

	require.NoError(t, err)
	other, err := repo.OverlappingEnrollmentSemester(context.Background(), springSemester, day(15), day(29))
	require.NoError(t, err)
	assert.Empty(t, other)
}

// The wizard moves a planned semester's window with this update; dragging it
// onto the open one has to be refused just like creating it there.
func TestEnrollmentWindow_MovedOntoTheOpenOne_Refused(t *testing.T) {
	pool, repo := newWindowRepo(t)
	_, err := createWindow(t, pool, repo, springSemester, sharedRepo.PeriodTypeEnrollment, day(15), day(29))
	require.NoError(t, err)

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = repo.UpdatePeriodBySemesterAndTypeTx(ctx, tx, springSemester, sharedRepo.PeriodTypeEnrollment, day(5), day(29))

	require.ErrorIs(t, err, sharedRepo.ErrEnrollmentWindowOverlap)
}

// Closing the open window frees its dates for another semester.
func TestEnrollmentWindow_ClosedWindow_DoesNotBlock(t *testing.T) {
	pool, repo := newWindowRepo(t)
	fall, err := repo.GetPeriods(context.Background(), fallSemester, sharedRepo.PeriodTypeEnrollment)
	require.NoError(t, err)
	require.Len(t, fall, 1)
	inactive := false
	_, err = repo.UpdatePeriod(context.Background(), fall[0].ID, nil, &inactive)
	require.NoError(t, err)

	_, err = createWindow(t, pool, repo, springSemester, sharedRepo.PeriodTypeEnrollment, day(10), day(20))

	require.NoError(t, err)
}

// Only enrollment is exclusive: grading or attendance of two semesters may
// overlap, and a semester's own rows never conflict with each other.
func TestEnrollmentWindow_OtherPeriodTypes_Unaffected(t *testing.T) {
	pool, repo := newWindowRepo(t)

	_, err := createWindow(t, pool, repo, springSemester, sharedRepo.PeriodTypeGrading, day(10), day(20))
	require.NoError(t, err)
	_, err = createWindow(t, pool, repo, fallSemester, sharedRepo.PeriodTypeGrading, day(10), day(20))
	require.NoError(t, err)
}
