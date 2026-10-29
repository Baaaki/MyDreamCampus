//go:build integration

package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fall = "2026-2027-Fall"

func newEnrollmentDB(t *testing.T) (*pgxpool.Pool, *EnrollmentRepository) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	pool := pgtest.New(t, "enrollment", "../sql/migrations")
	return pool, NewEnrollmentRepository(pool)
}

func submit(repo *EnrollmentRepository, student uuid.UUID, courses ...ProgramCourseSnapshot) (db.EnrollmentProgram, error) {
	return resubmit(repo, student, nil, courses...)
}

// resubmit is submit for a student whose pending program the new one replaces.
func resubmit(repo *EnrollmentRepository, student uuid.UUID, replace *ProgramReplacement, courses ...ProgramCourseSnapshot) (db.EnrollmentProgram, error) {
	return repo.CreateProgramWithCoursesAndEvent(context.Background(), db.CreateEnrollmentProgramParams{
		StudentID: utils.UUIDToPgtype(student), Semester: fall,
		Status: db.NullEnrollmentStatusEnum{EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending, Valid: true},
	}, courses, map[string]any{}, replace)
}

func replacing(program db.EnrollmentProgram) *ProgramReplacement {
	return &ProgramReplacement{ProgramID: utils.PgtypeToUUID(program.ID), Status: db.EnrollmentEnrollmentStatusEnumPending, EventPayload: map[string]any{}}
}

func course(capacity int16) ProgramCourseSnapshot {
	return ProgramCourseSnapshot{CourseID: uuid.New(), CourseCode: "BLM101", CourseName: "Programlamaya Giriş", Credits: 4, MaxCapacity: capacity}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

// Every submission counts the seats and then inserts; without the advisory
// lock two students who both see one seat left would both get it. Only
// concurrent transactions on a real database can show the lock holding.
func TestCreateProgram_LastSeat_OneOfManyConcurrentStudentsGetsIt(t *testing.T) {
	pool, repo := newEnrollmentDB(t)
	c := course(1)
	const students = 8

	var wg sync.WaitGroup
	errs := make([]error, students)
	start := make(chan struct{})
	for i := range students {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = submit(repo, uuid.New(), c)
		}()
	}
	close(start)
	wg.Wait()

	won := 0
	for _, err := range errs {
		if err == nil {
			won++
			continue
		}
		assert.ErrorIs(t, err, sharedErrors.ErrConflict)
	}
	assert.Equal(t, 1, won)
	n, err := repo.CountEnrollmentForCourse(context.Background(), c.CourseID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM enrollment.outbox_events"), "one submitted event, for the winner")
}

// Two students who pick the same courses in opposite orders must both get
// in. Locked in request order, each transaction held one course and waited
// for the other's, and Postgres broke the deadlock by failing one of them.
func TestCreateProgram_SameCoursesInOppositeOrder_NoDeadlock(t *testing.T) {
	_, repo := newEnrollmentDB(t)
	a, b, c := course(100), course(100), course(100)
	const pairs = 20

	var wg sync.WaitGroup
	errs := make([]error, 2*pairs)
	start := make(chan struct{})
	for i := range pairs {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, errs[2*i] = submit(repo, uuid.New(), a, b, c)
		}()
		go func() {
			defer wg.Done()
			<-start
			_, errs[2*i+1] = submit(repo, uuid.New(), c, b, a)
		}()
	}
	close(start)
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err)
	}
	n, err := repo.CountEnrollmentForCourse(context.Background(), b.CourseID)
	require.NoError(t, err)
	assert.Equal(t, 2*pairs, n)
}

// The pending program used to be deleted in its own transaction before the
// new one was even checked, so a new selection that failed left the student
// with no program at all.
func TestCreateProgram_ReplacementHitsAFullCourse_PendingProgramStays(t *testing.T) {
	pool, repo := newEnrollmentDB(t)
	student := uuid.New()
	kept := course(30)
	full := course(1)
	pending, err := submit(repo, student, kept)
	require.NoError(t, err)
	_, err = submit(repo, uuid.New(), full)
	require.NoError(t, err)

	_, err = resubmit(repo, student, replacing(pending), full)

	require.ErrorIs(t, err, sharedErrors.ErrConflict)
	still, err := repo.GetEnrollmentProgramByStudentAndSemester(context.Background(), student, fall)
	require.NoError(t, err)
	assert.Equal(t, pending.ID, still.ID)
	assert.Equal(t, 0, count(t, pool, "SELECT count(*) FROM enrollment.outbox_events WHERE event_type = 'enrollment.program.cancelled'"),
		"nothing may announce a cancellation that rolled back")
}

func TestCreateProgram_Replacement_SwapsProgramsAtomicallyAndKeepsTheSeat(t *testing.T) {
	pool, repo := newEnrollmentDB(t)
	student := uuid.New()
	last := course(1)
	pending, err := submit(repo, student, last)
	require.NoError(t, err)

	// The course is full with the student's own seat; the replacement
	// frees that seat inside the same transaction and takes it back.
	replaced, err := resubmit(repo, student, replacing(pending), last)

	require.NoError(t, err)
	assert.NotEqual(t, pending.ID, replaced.ID)
	_, err = repo.GetEnrollmentProgramByID(context.Background(), utils.PgtypeToUUID(pending.ID))
	assert.ErrorIs(t, err, sharedErrors.ErrNotFound)
	n, err := repo.CountEnrollmentForCourse(context.Background(), last.CourseID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM enrollment.outbox_events WHERE event_type = 'enrollment.program.cancelled'"))
}

func TestCreateProgram_ReplacedProgramApprovedMeanwhile_Conflict(t *testing.T) {
	_, repo := newEnrollmentDB(t)
	student := uuid.New()
	pending, err := submit(repo, student, course(30))
	require.NoError(t, err)
	_, err = repo.ApproveProgramWithEvent(context.Background(), utils.PgtypeToUUID(pending.ID), map[string]any{})
	require.NoError(t, err)

	_, err = resubmit(repo, student, replacing(pending), course(30))

	require.ErrorIs(t, err, serviceErrors.ErrSubmissionConflict)
	approved, err := repo.GetEnrollmentProgramByID(context.Background(), utils.PgtypeToUUID(pending.ID))
	require.NoError(t, err)
	assert.Equal(t, db.EnrollmentEnrollmentStatusEnumApproved, approved.Status.EnrollmentStatusEnum)
}

// A double click sends two submissions for one student at once. Both pass
// the "no program yet" read; the unique index lets one in, and the other used
// to surface its violation as a 500.
func TestCreateProgram_SameStudentTwiceAtOnce_OneWinsTheRestConflict(t *testing.T) {
	_, repo := newEnrollmentDB(t)
	student := uuid.New()
	const attempts = 6

	var wg sync.WaitGroup
	errs := make([]error, attempts)
	start := make(chan struct{})
	for i := range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = submit(repo, student, course(30))
		}()
	}
	close(start)
	wg.Wait()

	won := 0
	for _, err := range errs {
		if err == nil {
			won++
			continue
		}
		assert.ErrorIs(t, err, serviceErrors.ErrSubmissionConflict)
	}
	assert.Equal(t, 1, won)
}

func TestCreateProgram_RejectedOrCancelledProgram_GivesItsSeatBack(t *testing.T) {
	pool, repo := newEnrollmentDB(t)
	ctx := context.Background()
	c := course(1)
	rejected, err := submit(repo, uuid.New(), c)
	require.NoError(t, err)
	_, err = submit(repo, uuid.New(), c)
	require.ErrorIs(t, err, sharedErrors.ErrConflict)

	require.NoError(t, repo.RejectProgramWithEventAndLog(ctx, utils.PgtypeToUUID(rejected.ID), db.CreateRejectionLogParams{
		OriginalProgramID: rejected.ID, StudentID: rejected.StudentID, AdvisorID: utils.UUIDToPgtype(uuid.New()),
		AdvisorFullname: "Ahmet Yılmaz", Semester: fall, RejectionReason: "Ön koşul eksik", RejectedCourses: []byte(`[]`),
	}, map[string]any{}))

	assert.Equal(t, 0, count(t, pool, "SELECT count(*) FROM enrollment.enrollment_program_courses"), "the program's courses went with it")
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM enrollment.enrollment_rejection_logs WHERE original_program_id = $1", rejected.ID))
	cancelled, err := submit(repo, uuid.New(), c)
	require.NoError(t, err, "the rejected program's seat is free")

	require.NoError(t, repo.CancelProgramWithEvent(ctx, utils.PgtypeToUUID(cancelled.ID), map[string]any{}))
	_, err = submit(repo, uuid.New(), c)
	assert.NoError(t, err, "the cancelled program's seat is free")
}

func TestApproveProgram_ConcurrentApprovals_ApproveOnceAndAnnounceOnce(t *testing.T) {
	pool, repo := newEnrollmentDB(t)
	program, err := submit(repo, uuid.New(), course(30))
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make([]error, 4)
	start := make(chan struct{})
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = repo.ApproveProgramWithEvent(context.Background(), utils.PgtypeToUUID(program.ID), map[string]any{})
		}()
	}
	close(start)
	wg.Wait()

	approved := 0
	for _, err := range errs {
		if err == nil {
			approved++
			continue
		}
		assert.ErrorIs(t, err, serviceErrors.ErrProgramNotPending)
	}
	assert.Equal(t, 1, approved)
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM enrollment.outbox_events WHERE event_type = 'enrollment.program.approved'"))

	err = repo.CancelProgramWithEvent(context.Background(), utils.PgtypeToUUID(program.ID), map[string]any{})
	assert.ErrorIs(t, err, sharedErrors.ErrNotFound, "an approved program cannot be cancelled")
}

func TestPassedPrerequisites_RegradeKeepsOneRowPerCourseCode(t *testing.T) {
	pool := pgtest.New(t, "enrollment", "../sql/migrations")
	repo := NewPassedPrerequisitesRepository(pool)
	ctx := context.Background()
	student := uuid.New()

	require.NoError(t, repo.UpsertPassedPrerequisite(ctx, student, uuid.New(), "BLM101", "2025-2026-Fall", "1.00"))
	require.NoError(t, repo.UpsertPassedPrerequisite(ctx, student, uuid.New(), "BLM101", "2025-2026-Spring", "3.50"))

	passed, err := repo.HasPassedPrerequisite(ctx, student, "BLM101")
	require.NoError(t, err)
	assert.True(t, passed)
	var grade, semester string
	require.NoError(t, pool.QueryRow(ctx, "SELECT grade_point, semester FROM enrollment.student_passed_prerequisites WHERE student_id = $1", student).Scan(&grade, &semester))
	assert.Equal(t, "3.50", grade, "the retake replaces the first pass")
	assert.Equal(t, "2025-2026-Spring", semester)
	passed, err = repo.HasPassedPrerequisite(ctx, uuid.New(), "BLM101")
	require.NoError(t, err)
	assert.False(t, passed)
}
