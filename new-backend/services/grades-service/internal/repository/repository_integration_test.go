//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gradesDB struct {
	pool      *pgxpool.Pool
	cache     *CacheRepository
	regs      *RegistrationRepository
	scores    *ScoreRepository
	completed *CompletedRepository
	course    uuid.UUID
}

func newGradesDB(t *testing.T) *gradesDB {
	t.Helper()
	pool := pgtest.New(t, "grades", "../sql/migrations")
	g := &gradesDB{
		pool: pool, cache: NewCacheRepository(pool), regs: NewRegistrationRepository(pool),
		scores: NewScoreRepository(pool), completed: NewCompletedRepository(pool), course: uuid.New(),
	}
	_, err := g.cache.UpsertCourseCache(context.Background(), db.UpsertCourseCacheParams{
		ID: g.course, CourseCode: "BLM101", CourseName: "Programlamaya Giriş", Credits: 4, Semester: "2026-2027-Fall",
		InstructorID: uuid.New(), AssessmentSchema: []byte(`[{"slug":"vize","weight":40},{"slug":"final","weight":60}]`),
	})
	require.NoError(t, err)
	return g
}

// register adds a student to the course and returns the registration id.
func (g *gradesDB) register(t *testing.T, number string, attendanceFailed bool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	student := uuid.New()
	_, err := g.cache.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{ID: student, StudentNumber: number, IsActive: pgtype.Bool{Bool: true, Valid: true}})
	require.NoError(t, err)
	reg, err := g.regs.CreateRegistration(ctx, db.CreateRegistrationParams{
		StudentID: student, CourseID: g.course, Semester: "2026-2027-Fall",
		IsAttendanceFailed: pgtype.Bool{Bool: attendanceFailed, Valid: true},
	})
	require.NoError(t, err)
	return reg.ID
}

func (g *gradesDB) outboxCount(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, g.pool.QueryRow(context.Background(), "SELECT count(*) FROM grades.outbox_events").Scan(&n))
	return n
}

func score(t *testing.T, reg uuid.UUID, slug string, v float64) db.UpsertAssessmentScoreParams {
	t.Helper()
	n := utils.Float64ToPgNumeric(v)
	require.True(t, n.Valid)
	return db.UpsertAssessmentScoreParams{RegistrationID: reg, Slug: slug, Score: n, GradedBy: uuid.New()}
}

// gpaValue reads the untyped GPA column the way the service does, and fails
// if pgx stops handing it over as a Numeric, which the service's parser
// would otherwise quietly read as zero.
func gpaValue(t *testing.T, v any) float64 {
	t.Helper()
	n, ok := v.(pgtype.Numeric)
	require.True(t, ok, "gpa scanned as %T", v)
	f, err := utils.PgNumericToFloat64(n)
	require.NoError(t, err)
	return f
}

func event() *db.CreateOutboxEventParams {
	return &db.CreateOutboxEventParams{EventType: "grade.submitted", RoutingKey: "grade.submitted", Payload: []byte(`{}`)}
}

func TestScoreRepository_UpsertWithEvent_WritesScoreAndEventTogether(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	reg := g.register(t, "1", false)

	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 70), event())
	require.NoError(t, err)
	_, err = g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 85), event())
	require.NoError(t, err)

	rows, err := g.scores.GetScoresByRegistration(ctx, reg)
	require.NoError(t, err)
	require.Len(t, rows, 1, "a second grade for the same assessment replaces the first")
	got, _ := utils.PgNumericToFloat64(rows[0].Score)
	assert.InDelta(t, 85, got, 0.001)
	assert.False(t, rows[0].IsLocked.Bool, "a submitted score is a draft until locked")
	assert.Equal(t, 2, g.outboxCount(t))
}

// A locked score is final: the upsert must neither change it nor leave an
// event behind that would tell the student it changed.
func TestScoreRepository_LockedScore_NeitherChangedNorAnnounced(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	reg := g.register(t, "1", false)
	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 70), event())
	require.NoError(t, err)
	require.NoError(t, g.scores.LockScore(ctx, reg, "vize"))

	_, err = g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 100), event())

	require.Error(t, err)
	s, err := g.scores.GetScoreByRegistrationAndSlug(ctx, reg, "vize")
	require.NoError(t, err)
	got, _ := utils.PgNumericToFloat64(s.Score)
	assert.InDelta(t, 70, got, 0.001)
	assert.Equal(t, 1, g.outboxCount(t))
}

func TestScoreRepository_BulkUpsert_AllOrNothing(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	a, b, locked := g.register(t, "1", false), g.register(t, "2", false), g.register(t, "3", false)
	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, locked, "final", 40), nil)
	require.NoError(t, err)
	require.NoError(t, g.scores.LockScore(ctx, locked, "final"))

	_, err = g.scores.BulkUpsertAssessmentScoresWithEvents(ctx, []BulkUpsertEntry{
		{ScoreParams: score(t, a, "final", 80), EventParams: event()},
		{ScoreParams: score(t, b, "final", 90), EventParams: event()},
		{ScoreParams: score(t, locked, "final", 99), EventParams: event()},
	})

	require.Error(t, err, "one locked row fails the batch")
	for _, reg := range []uuid.UUID{a, b} {
		rows, err := g.scores.GetScoresByRegistration(ctx, reg)
		require.NoError(t, err)
		assert.Empty(t, rows, "the rows before the failure were rolled back")
	}
	assert.Equal(t, 0, g.outboxCount(t))

	n, err := g.scores.BulkUpsertAssessmentScoresWithEvents(ctx, []BulkUpsertEntry{
		{ScoreParams: score(t, a, "final", 80), EventParams: event()},
		{ScoreParams: score(t, b, "final", 90)},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, 1, g.outboxCount(t), "an entry without event params publishes nothing")
}

func TestScoreRepository_ScoreOutOfRange_RejectedByTheDatabase(t *testing.T) {
	g := newGradesDB(t)
	reg := g.register(t, "1", false)

	_, err := g.scores.UpsertAssessmentScoreWithEvent(context.Background(), score(t, reg, "vize", 100.5), event())

	assert.Error(t, err, "the CHECK constraint is the last line behind the service's range check")
	assert.Equal(t, 0, g.outboxCount(t))
}

func TestScoreRepository_CountsAndLocksPerCourse(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	a, b := g.register(t, "1", false), g.register(t, "2", false)
	g.register(t, "3", true)
	for _, reg := range []uuid.UUID{a, b} {
		_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 60), nil)
		require.NoError(t, err)
	}
	absent := db.UpsertAssessmentScoreParams{RegistrationID: a, Slug: "final", IsAbsent: pgtype.Bool{Bool: true, Valid: true}, GradedBy: uuid.New()}
	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, absent, nil)
	require.NoError(t, err)

	total, err := g.regs.CountRegistrationsByCourse(ctx, g.course)
	require.NoError(t, err)
	eligible, err := g.regs.CountEligibleRegistrationsByCourse(ctx, g.course)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, int64(2), eligible, "an attendance failure needs no scores")

	graded, err := g.scores.CountScoresBySlugAndCourse(ctx, db.CountScoresBySlugAndCourseParams{CourseID: g.course, Slug: "final"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), graded, "an absence counts as graded")

	require.NoError(t, g.scores.LockScoresByCourseAndSlug(ctx, g.course, "vize"))
	lockedCount, err := g.scores.CountLockedScoresBySlugAndCourse(ctx, g.course, "vize")
	require.NoError(t, err)
	assert.Equal(t, int64(2), lockedCount)
	lockedIDs, err := g.scores.GetLockedRegistrationsBySlug(ctx, "vize", []uuid.UUID{a, b, uuid.New()})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a, b}, lockedIDs)

	require.NoError(t, g.scores.UnlockScore(ctx, a, "vize"))
	lockedCount, err = g.scores.CountLockedScoresBySlugAndCourse(ctx, g.course, "vize")
	require.NoError(t, err)
	assert.Equal(t, int64(1), lockedCount)
}

func completedCourse(student uuid.UUID, code string, credits int16, gp db.GradesGradePointEnum, at time.Time) db.CreateCompletedCourseParams {
	return db.CreateCompletedCourseParams{
		StudentID: student, StudentNumber: "1", CourseID: uuid.New(), CourseCode: code, CourseName: code, Credits: credits,
		Semester: "2026-2027-Fall", InstructorID: uuid.New(), AssessmentScores: []byte(`{}`),
		WeightedAverage: utils.Float64ToPgNumeric(0), GradePoint: gp, GradingType: db.GradesGradingTypeEnumAbsolute,
		GradingConfig: []byte(`{}`), ClassStatistics: []byte(`{}`), FinalizedAt: utils.TimeToPgTimestamp(at), FinalizedBy: uuid.New(),
	}
}

func TestCompletedRepository_GPA_WeightsGradePointsByCredits(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	student := uuid.New()
	base := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	for i, c := range []db.CreateCompletedCourseParams{
		completedCourse(student, "BLM101", 4, db.GradesGradePointEnum400, base),
		completedCourse(student, "MAT101", 3, db.GradesGradePointEnum225, base.Add(time.Hour)),
		completedCourse(student, "FIZ101", 2, db.GradesGradePointEnum000, base.Add(2*time.Hour)),
		completedCourse(uuid.New(), "BLM101", 4, db.GradesGradePointEnum000, base),
	} {
		_, err := g.completed.CreateCompletedCourse(ctx, c)
		require.NoError(t, err, "course %d", i)
	}

	gpa, err := g.completed.CalculateStudentGPA(ctx, student)
	require.NoError(t, err)
	assert.InDelta(t, 2.53, gpaValue(t, gpa.Gpa), 0.001, "(4×4.00 + 3×2.25 + 2×0.00) / 9, rounded to two places")

	transcript, err := g.completed.GetTranscriptData(ctx, student)
	require.NoError(t, err)
	require.Len(t, transcript, 3, "only the student's own courses")
	assert.Equal(t, []string{"BLM101", "MAT101", "FIZ101"},
		[]string{transcript[0].CourseCode, transcript[1].CourseCode, transcript[2].CourseCode}, "in the order they were finalized")
}

func TestCompletedRepository_NoCourses_GPAIsZero(t *testing.T) {
	g := newGradesDB(t)

	gpa, err := g.completed.CalculateStudentGPA(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Zero(t, gpaValue(t, gpa.Gpa))
}

// AutoFinalize binds every repository to one transaction; a rollback must
// leave neither the completed records nor the cleared registrations.
func TestRepositories_WithTx_RollbackLeavesNothing(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	reg := g.register(t, "1", false)
	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 90), nil)
	require.NoError(t, err)

	tx, err := g.pool.Begin(ctx)
	require.NoError(t, err)
	_, err = g.completed.WithTx(tx).CreateCompletedCourse(ctx, completedCourse(uuid.New(), "BLM101", 4, db.GradesGradePointEnum400, time.Now()))
	require.NoError(t, err)
	require.NoError(t, g.scores.WithTx(tx).DeleteScoresByCourse(ctx, g.course))
	require.NoError(t, g.regs.WithTx(tx).DeleteRegistrationsByCourse(ctx, g.course))
	require.NoError(t, tx.Rollback(ctx))

	rows, err := g.completed.GetCompletedCoursesByCourse(ctx, g.course)
	require.NoError(t, err)
	assert.Empty(t, rows)
	n, err := g.regs.CountRegistrationsByCourse(ctx, g.course)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	scores, err := g.scores.GetScoresByRegistration(ctx, reg)
	require.NoError(t, err)
	assert.Len(t, scores, 1)
}

// A repository bound to a transaction must not open one of its own for the
// *WithEvent writes: they are part of the caller's job and roll back with it.
func TestScoreRepository_BoundToATransaction_WritesRollBackWithIt(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	reg := g.register(t, "1", false)

	tx, err := g.pool.Begin(ctx)
	require.NoError(t, err)
	scores := g.scores.WithTx(tx)
	_, err = scores.UpsertAssessmentScoreWithEvent(ctx, score(t, reg, "vize", 70), event())
	require.NoError(t, err)
	_, err = scores.BulkUpsertAssessmentScoresWithEvents(ctx, []BulkUpsertEntry{{ScoreParams: score(t, reg, "final", 60), EventParams: event()}})
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	rows, err := g.scores.GetScoresByRegistration(ctx, reg)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, g.outboxCount(t))
}

func TestScoreRepository_LockCourseGrading_TheNextJobWaitsForTheFirst(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	first, err := g.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = first.Rollback(ctx) }()
	require.NoError(t, g.scores.WithTx(first).LockCourseGrading(ctx, g.course))

	lockOn := func(course uuid.UUID) <-chan error {
		done := make(chan error, 1)
		go func() {
			tx, err := g.pool.Begin(ctx)
			if err != nil {
				done <- err
				return
			}
			defer func() { _ = tx.Rollback(ctx) }()
			done <- g.scores.WithTx(tx).LockCourseGrading(ctx, course)
		}()
		return done
	}
	sameCourse := lockOn(g.course)
	otherCourse := lockOn(uuid.New())

	select {
	case err := <-otherCourse:
		require.NoError(t, err, "another course is not held up")
	case <-time.After(5 * time.Second):
		t.Fatal("a job on another course waited for this one")
	}
	select {
	case <-sameCourse:
		t.Fatal("a second job on the course ran while the first held it")
	case <-time.After(300 * time.Millisecond):
	}
	require.NoError(t, first.Commit(ctx))
	select {
	case err := <-sameCourse:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the second job never got the course after the first ended")
	}
}

// registrationOf reads the student's registration on the course, if any.
func (g *gradesDB) registrationOf(t *testing.T, student uuid.UUID) (uuid.UUID, bool) {
	t.Helper()
	var id, program pgtype.UUID
	err := g.pool.QueryRow(context.Background(),
		"SELECT id, program_id FROM grades.student_course_registrations WHERE student_id = $1 AND course_id = $2",
		student, g.course).Scan(&id, &program)
	if err != nil {
		return uuid.Nil, false
	}
	return utils.PgtypeToUUID(program), true
}

func (g *gradesDB) studentOf(t *testing.T, reg uuid.UUID) uuid.UUID {
	t.Helper()
	var student uuid.UUID
	require.NoError(t, g.pool.QueryRow(context.Background(),
		"SELECT student_id FROM grades.student_course_registrations WHERE id = $1", reg).Scan(&student))
	return student
}

// A changed approved program is cancelled and its replacement approved; the
// replacement's approval may be consumed first. The cancel then leaves the
// course it took over, and never takes a registration that holds a score.
func TestDeleteProgramRegistrations_KeepsTakenOverAndScoredRows(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	oldProgram, newProgram := uuid.New(), uuid.New()
	takenOver := g.studentOf(t, g.register(t, "1", false))
	scoredReg := g.register(t, "2", false)
	scored := g.studentOf(t, scoredReg)
	dropped := g.studentOf(t, g.register(t, "3", false))
	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, scoredReg, "vize", 70), nil)
	require.NoError(t, err)

	_, err = g.regs.CreateRegistration(ctx, db.CreateRegistrationParams{
		StudentID: takenOver, CourseID: g.course, Semester: "2026-2027-Fall", ProgramID: utils.UUIDToPgtype(newProgram),
	})
	require.NoError(t, err)
	for _, student := range []uuid.UUID{takenOver, scored, dropped} {
		_, err := g.regs.DeleteProgramRegistrations(ctx, oldProgram, student, []uuid.UUID{g.course})
		require.NoError(t, err)
	}

	program, ok := g.registrationOf(t, takenOver)
	assert.True(t, ok, "the replacement program owns the course now")
	assert.Equal(t, newProgram, program)
	_, ok = g.registrationOf(t, scored)
	assert.True(t, ok, "deleting it would cascade the score away")
	_, ok = g.registrationOf(t, dropped)
	assert.False(t, ok)
}

func TestStudentHasGradesInSemester_ScoreCounts(t *testing.T) {
	g := newGradesDB(t)
	ctx := context.Background()
	scoredReg := g.register(t, "1", false)
	idle := g.studentOf(t, g.register(t, "2", false))
	_, err := g.scores.UpsertAssessmentScoreWithEvent(ctx, score(t, scoredReg, "vize", 70), nil)
	require.NoError(t, err)
	scored := g.studentOf(t, scoredReg)

	for _, tc := range []struct {
		student  uuid.UUID
		semester string
		want     bool
	}{{scored, "2026-2027-Fall", true}, {scored, "2026-2027-Spring", false}, {idle, "2026-2027-Fall", false}} {
		got, err := g.regs.StudentHasGradesInSemester(ctx, tc.student, tc.semester)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.semester)
	}
}
