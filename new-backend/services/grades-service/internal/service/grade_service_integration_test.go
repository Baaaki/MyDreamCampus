//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/grades/internal/dto"
	gradeErrors "github.com/baaaki/mydreamcampus/grades/internal/errors"
	"github.com/baaaki/mydreamcampus/grades/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gradingDB is one course with two students on a real database, graded
// through the real GradeService. Catalog is left out (no semester client),
// so the grading period is the only edit rule besides the score locks.
type gradingDB struct {
	pool       *pgxpool.Pool
	svc        *GradeService
	scores     *repository.ScoreRepository
	course     uuid.UUID
	instructor uuid.UUID
	ali, ayse  uuid.UUID // registration ids
}

func newGradingDB(t *testing.T) *gradingDB {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	ctx := context.Background()
	pool := pgtest.New(t, "grades", "../sql/migrations")
	cache := repository.NewCacheRepository(pool)
	regs := repository.NewRegistrationRepository(pool)
	outbox := repository.NewOutboxRepository(pool)
	g := &gradingDB{pool: pool, scores: repository.NewScoreRepository(pool), course: uuid.New(), instructor: uuid.New()}
	g.svc = NewGradeService(pool, cache, regs, g.scores, repository.NewCompletedRepository(pool), outbox,
		sharedRepo.NewSimplePeriodRepository(pool, "grades"),
		audit.NewEventAuditLogger(repository.NewAuditOutbox(outbox), "grades"), nil)

	_, err := cache.UpsertCourseCache(ctx, db.UpsertCourseCacheParams{
		ID: g.course, CourseCode: "BLM101", CourseName: "Programlamaya Giriş", Credits: 4, Semester: "2026-2027-Fall",
		InstructorID: g.instructor, AssessmentSchema: []byte(`[{"slug":"vize","weight":40},{"slug":"final","weight":60}]`),
	})
	require.NoError(t, err)
	enroll := func(number string) uuid.UUID {
		student := uuid.New()
		_, err := cache.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{ID: student, StudentNumber: number, IsActive: pgtype.Bool{Bool: true, Valid: true}})
		require.NoError(t, err)
		reg, err := regs.CreateRegistration(ctx, db.CreateRegistrationParams{
			StudentID: student, CourseID: g.course, Semester: "2026-2027-Fall", IsAttendanceFailed: pgtype.Bool{Valid: true},
		})
		require.NoError(t, err)
		return reg.ID
	}
	g.ali, g.ayse = enroll("2026001"), enroll("2026002")
	return g
}

func (g *gradingDB) grade(t *testing.T, reg uuid.UUID, slug string, v float64) {
	t.Helper()
	_, err := g.scores.UpsertAssessmentScoreWithEvent(context.Background(), db.UpsertAssessmentScoreParams{
		RegistrationID: reg, Slug: slug, Score: utils.Float64ToPgNumeric(v), GradedBy: g.instructor,
	}, nil)
	require.NoError(t, err)
}

func (g *gradingDB) scoreOf(t *testing.T, reg uuid.UUID, slug string) float64 {
	t.Helper()
	s, err := g.scores.GetScoreByRegistrationAndSlug(context.Background(), reg, slug)
	require.NoError(t, err)
	v, err := utils.PgNumericToFloat64(s.Score)
	require.NoError(t, err)
	return v
}

// holdCourse stands in for another grading job on the course: it takes the
// course lock and keeps it until the test commits the returned transaction.
func (g *gradingDB) holdCourse(t *testing.T) (pgx.Tx, *repository.ScoreRepository) {
	t.Helper()
	tx, err := g.pool.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	scores := g.scores.WithTx(tx)
	require.NoError(t, scores.LockCourseGrading(context.Background(), g.course))
	return tx, scores
}

// inBackground runs job and hands its error back once it returns.
func inBackground(job func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- job() }()
	return done
}

func requireWaiting(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("ran while another grading job held the course (err: %v)", err)
	case <-time.After(300 * time.Millisecond):
	}
}

func requireFinished(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("still waiting after the other job ended")
		return nil
	}
}

// The teacher saves while the assessment is being locked. The save waits,
// then finds the scores locked and leaves them as they were, instead of
// failing the whole batch with a generic error.
func TestBulkSubmitScores_DuringALock_WaitsThenLeavesLockedScoresAlone(t *testing.T) {
	g := newGradingDB(t)
	ctx := context.Background()
	g.grade(t, g.ali, "vize", 70)
	g.grade(t, g.ayse, "vize", 80)
	lock, lockScores := g.holdCourse(t)

	var resp *dto.BulkSubmitScoresResponse
	done := inBackground(func() (err error) {
		resp, err = g.svc.BulkSubmitScores(ctx, g.instructor, g.course, dto.BulkSubmitScoresRequest{
			Slug: "vize", Scores: []dto.BulkScoreEntry{{RegistrationID: g.ali, Score: ptr(90)}, {RegistrationID: g.ayse, Score: ptr(95)}},
		})
		return err
	})
	requireWaiting(t, done)
	require.NoError(t, lockScores.LockScoresByCourseAndSlug(ctx, g.course, "vize"))
	require.NoError(t, lock.Commit(ctx))

	require.NoError(t, requireFinished(t, done))
	assert.Equal(t, 0, resp.SuccessCount)
	assert.Equal(t, 2, resp.SkippedLocked, "the teacher is told why nothing was saved")
	assert.InDelta(t, 70, g.scoreOf(t, g.ali, "vize"), 0.001)
	assert.InDelta(t, 80, g.scoreOf(t, g.ayse, "vize"), 0.001)
}

// The teacher locks the assessment while a save of the last missing score
// is still running. The lock waits for the save, so it sees every score and
// locks them all, instead of refusing the assessment as incomplete.
func TestLockAssessment_DuringASave_WaitsAndLocksTheSavedScoreToo(t *testing.T) {
	g := newGradingDB(t)
	ctx := context.Background()
	g.grade(t, g.ali, "final", 60)
	save, saveScores := g.holdCourse(t)
	_, err := saveScores.UpsertAssessmentScoreWithEvent(ctx, db.UpsertAssessmentScoreParams{
		RegistrationID: g.ayse, Slug: "final", Score: utils.Float64ToPgNumeric(75), GradedBy: g.instructor,
	}, nil)
	require.NoError(t, err)

	var resp *dto.LockAssessmentResponse
	done := inBackground(func() (err error) {
		resp, err = g.svc.LockAssessmentBySlug(ctx, g.instructor, g.course, "final")
		return err
	})
	requireWaiting(t, done)
	require.NoError(t, save.Commit(ctx))

	require.NoError(t, requireFinished(t, done))
	assert.Equal(t, 2, resp.LockedCount)
	locked, err := g.scores.CountLockedScoresBySlugAndCourse(ctx, g.course, "final")
	require.NoError(t, err)
	assert.Equal(t, int64(2), locked)
}

func TestSubmitScore_LockedWhileItWaited_RefusedAsLocked(t *testing.T) {
	g := newGradingDB(t)
	ctx := context.Background()
	g.grade(t, g.ali, "vize", 70)
	lock, lockScores := g.holdCourse(t)

	done := inBackground(func() error {
		_, err := g.svc.SubmitScore(ctx, g.instructor, g.course, dto.SubmitScoreRequest{RegistrationID: g.ali, Slug: "vize", Score: ptr(99)})
		return err
	})
	requireWaiting(t, done)
	require.NoError(t, lockScores.LockScore(ctx, g.ali, "vize"))
	require.NoError(t, lock.Commit(ctx))

	err := requireFinished(t, done)
	assert.ErrorIs(t, err, gradeErrors.ErrScoreLocked)
	assert.InDelta(t, 70, g.scoreOf(t, g.ali, "vize"), 0.001)
}

func TestAutoFinalize_WaitsForTheCourse(t *testing.T) {
	g := newGradingDB(t)
	ctx := context.Background()
	for _, reg := range []uuid.UUID{g.ali, g.ayse} {
		g.grade(t, reg, "vize", 50)
		g.grade(t, reg, "final", 50)
	}
	other, _ := g.holdCourse(t)

	done := inBackground(func() error {
		_, err := g.svc.AutoFinalize(ctx, g.course, g.instructor)
		return err
	})
	requireWaiting(t, done)
	require.NoError(t, other.Commit(ctx))

	require.NoError(t, requireFinished(t, done))
	n, err := repository.NewRegistrationRepository(g.pool).CountRegistrationsByCourse(ctx, g.course)
	require.NoError(t, err)
	assert.Zero(t, n, "finalized: the operational rows are gone")
}
