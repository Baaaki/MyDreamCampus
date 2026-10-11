package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/grades/internal/dto"
	gradeErrors "github.com/baaaki/mydreamcampus/grades/internal/errors"
	"github.com/baaaki/mydreamcampus/grades/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gradesWorld is an in-memory stand-in for the grades database. Writes made
// through a store bound to a transaction are held until Commit and dropped
// on Rollback, so a workflow that writes outside its transaction, or leaves
// half its writes behind on an error, shows up here as it would in Postgres.
type gradesWorld struct {
	courses   map[uuid.UUID]db.GradesCoursesView
	students  map[uuid.UUID]db.GradesStudentsView
	prereqs   map[string]bool
	regs      []db.GetRegistrationByIDRow
	scores    []db.GradesStudentAssessmentScore
	completed []db.GradesStudentCompletedCourse
	outbox    []db.CreateOutboxEventParams
	periodEnd *time.Time
	audits    []audit.AuditEvent

	failScoresFor       uuid.UUID
	failCreateCompleted bool
	txs                 []*fakeTx
	courseLocks         []courseLock
}

// courseLock is one LockCourseGrading call and the transaction it ran in.
type courseLock struct {
	course uuid.UUID
	tx     *fakeTx
}

type fakeTx struct {
	pgx.Tx
	ops       []func()
	committed bool
}

func (t *fakeTx) Commit(context.Context) error {
	for _, op := range t.ops {
		op()
	}
	t.committed = true
	return nil
}

func (t *fakeTx) Rollback(context.Context) error { return nil }

func (w *gradesWorld) Begin(context.Context) (pgx.Tx, error) {
	tx := &fakeTx{}
	w.txs = append(w.txs, tx)
	return tx, nil
}

// write applies op now, or at commit when tx is set.
func write(tx *fakeTx, op func()) {
	if tx != nil {
		tx.ops = append(tx.ops, op)
		return
	}
	op()
}

func asFakeTx(tx pgx.Tx) *fakeTx { return tx.(*fakeTx) }

type cacheView struct{ w *gradesWorld }

func (v cacheView) GetCourseCacheByID(_ context.Context, id uuid.UUID) (db.GradesCoursesView, error) {
	c, ok := v.w.courses[id]
	if !ok {
		return c, pgx.ErrNoRows
	}
	return c, nil
}

func (v cacheView) GetStudentCacheByID(_ context.Context, id uuid.UUID) (db.GradesStudentsView, error) {
	s, ok := v.w.students[id]
	if !ok {
		return s, pgx.ErrNoRows
	}
	return s, nil
}

func (v cacheView) IsPrerequisiteCourse(_ context.Context, code string) (bool, error) {
	return v.w.prereqs[code], nil
}

type regView struct {
	w  *gradesWorld
	tx *fakeTx
}

func (v regView) WithTx(tx pgx.Tx) registrationStore { return regView{v.w, asFakeTx(tx)} }

func (v regView) GetRegistrationByID(_ context.Context, id uuid.UUID) (db.GetRegistrationByIDRow, error) {
	for _, r := range v.w.regs {
		if r.ID == id {
			return r, nil
		}
	}
	return db.GetRegistrationByIDRow{}, pgx.ErrNoRows
}

func (v regView) GetRegistrationsByIDs(_ context.Context, ids []uuid.UUID) ([]db.GetRegistrationsByIDsRow, error) {
	var out []db.GetRegistrationsByIDsRow
	for _, r := range v.w.regs {
		if slices.Contains(ids, r.ID) {
			out = append(out, db.GetRegistrationsByIDsRow{
				ID: r.ID, StudentID: r.StudentID, CourseID: r.CourseID, Semester: r.Semester,
				IsAttendanceFailed: r.IsAttendanceFailed, StudentNumber: r.StudentNumber,
				CourseCode: r.CourseCode, InstructorID: r.InstructorID, AssessmentSchema: r.AssessmentSchema,
			})
		}
	}
	return out, nil
}

func (v regView) GetRegistrationsByCourse(_ context.Context, courseID uuid.UUID) ([]db.GetRegistrationsByCourseRow, error) {
	var out []db.GetRegistrationsByCourseRow
	for _, r := range v.w.regs {
		if r.CourseID == courseID {
			out = append(out, db.GetRegistrationsByCourseRow{
				ID: r.ID, StudentID: r.StudentID, CourseID: r.CourseID, Semester: r.Semester,
				IsAttendanceFailed: r.IsAttendanceFailed, StudentNumber: r.StudentNumber,
				StudentFirstName: r.StudentFirstName, StudentLastName: r.StudentLastName,
			})
		}
	}
	return out, nil
}

func (v regView) GetActiveRegistrationsByStudent(context.Context, uuid.UUID) ([]db.GetActiveRegistrationsByStudentRow, error) {
	return nil, nil
}

func (v regView) CountRegistrationsByCourse(_ context.Context, courseID uuid.UUID) (int64, error) {
	var n int64
	for _, r := range v.w.regs {
		if r.CourseID == courseID {
			n++
		}
	}
	return n, nil
}

func (v regView) CountEligibleRegistrationsByCourse(_ context.Context, courseID uuid.UUID) (int64, error) {
	var n int64
	for _, r := range v.w.regs {
		if r.CourseID == courseID && !r.IsAttendanceFailed.Bool {
			n++
		}
	}
	return n, nil
}

func (v regView) DeleteRegistrationsByCourse(_ context.Context, courseID uuid.UUID) error {
	write(v.tx, func() {
		v.w.regs = slices.DeleteFunc(v.w.regs, func(r db.GetRegistrationByIDRow) bool { return r.CourseID == courseID })
	})
	return nil
}

type scoreView struct {
	w  *gradesWorld
	tx *fakeTx
}

func (v scoreView) WithTx(tx pgx.Tx) scoreStore { return scoreView{v.w, asFakeTx(tx)} }

func (v scoreView) courseOf(regID uuid.UUID) uuid.UUID {
	for _, r := range v.w.regs {
		if r.ID == regID {
			return r.CourseID
		}
	}
	return uuid.Nil
}

func (v scoreView) GetScoreByRegistrationAndSlug(_ context.Context, regID uuid.UUID, slug string) (db.GradesStudentAssessmentScore, error) {
	for _, s := range v.w.scores {
		if s.RegistrationID == regID && s.Slug == slug {
			return s, nil
		}
	}
	return db.GradesStudentAssessmentScore{}, pgx.ErrNoRows
}

func (v scoreView) GetScoresByRegistration(_ context.Context, regID uuid.UUID) ([]db.GradesStudentAssessmentScore, error) {
	if regID == v.w.failScoresFor {
		return nil, errors.New("connection reset")
	}
	var out []db.GradesStudentAssessmentScore
	for _, s := range v.w.scores {
		if s.RegistrationID == regID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (v scoreView) GetLockedRegistrationsBySlug(_ context.Context, slug string, ids []uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for _, s := range v.w.scores {
		if s.Slug == slug && s.IsLocked.Bool && slices.Contains(ids, s.RegistrationID) {
			out = append(out, s.RegistrationID)
		}
	}
	return out, nil
}

func (v scoreView) CountScoresBySlugAndCourse(_ context.Context, arg db.CountScoresBySlugAndCourseParams) (int64, error) {
	var n int64
	for _, s := range v.w.scores {
		if s.Slug == arg.Slug && v.courseOf(s.RegistrationID) == arg.CourseID {
			n++
		}
	}
	return n, nil
}

func (v scoreView) CountLockedScoresBySlugAndCourse(_ context.Context, courseID uuid.UUID, slug string) (int64, error) {
	var n int64
	for _, s := range v.w.scores {
		if s.Slug == slug && s.IsLocked.Bool && v.courseOf(s.RegistrationID) == courseID {
			n++
		}
	}
	return n, nil
}

func (v scoreView) upsert(p db.UpsertAssessmentScoreParams) db.GradesStudentAssessmentScore {
	row := db.GradesStudentAssessmentScore{
		ID: uuid.New(), RegistrationID: p.RegistrationID, Slug: p.Slug, Score: p.Score,
		IsAbsent: p.IsAbsent, GradedBy: p.GradedBy, GradedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
	for i, s := range v.w.scores {
		if s.RegistrationID == p.RegistrationID && s.Slug == p.Slug {
			row.ID, row.IsLocked = s.ID, s.IsLocked
			v.w.scores[i] = row
			return row
		}
	}
	v.w.scores = append(v.w.scores, row)
	return row
}

func (v scoreView) UpsertAssessmentScoreWithEvent(_ context.Context, p db.UpsertAssessmentScoreParams, ev *db.CreateOutboxEventParams) (db.GradesStudentAssessmentScore, error) {
	if ev != nil {
		v.w.outbox = append(v.w.outbox, *ev)
	}
	return v.upsert(p), nil
}

func (v scoreView) BulkUpsertAssessmentScoresWithEvents(_ context.Context, entries []repository.BulkUpsertEntry) (int, error) {
	for _, e := range entries {
		v.upsert(e.ScoreParams)
		if e.EventParams != nil {
			v.w.outbox = append(v.w.outbox, *e.EventParams)
		}
	}
	return len(entries), nil
}

func (v scoreView) setLocked(match func(db.GradesStudentAssessmentScore) bool, locked bool) {
	for i, s := range v.w.scores {
		if match(s) {
			v.w.scores[i].IsLocked = pgtype.Bool{Bool: locked, Valid: true}
		}
	}
}

func (v scoreView) LockScore(_ context.Context, regID uuid.UUID, slug string) error {
	v.setLocked(func(s db.GradesStudentAssessmentScore) bool { return s.RegistrationID == regID && s.Slug == slug }, true)
	return nil
}

func (v scoreView) UnlockScore(_ context.Context, regID uuid.UUID, slug string) error {
	v.setLocked(func(s db.GradesStudentAssessmentScore) bool { return s.RegistrationID == regID && s.Slug == slug }, false)
	return nil
}

func (v scoreView) LockScoresByCourseAndSlug(_ context.Context, courseID uuid.UUID, slug string) error {
	v.setLocked(func(s db.GradesStudentAssessmentScore) bool {
		return s.Slug == slug && v.courseOf(s.RegistrationID) == courseID
	}, true)
	return nil
}

func (v scoreView) LockCourseGrading(_ context.Context, courseID uuid.UUID) error {
	v.w.courseLocks = append(v.w.courseLocks, courseLock{course: courseID, tx: v.tx})
	return nil
}

func (v scoreView) DeleteScoresByCourse(_ context.Context, courseID uuid.UUID) error {
	write(v.tx, func() {
		v.w.scores = slices.DeleteFunc(v.w.scores, func(s db.GradesStudentAssessmentScore) bool {
			return v.courseOf(s.RegistrationID) == courseID
		})
	})
	return nil
}

type completedView struct {
	w  *gradesWorld
	tx *fakeTx
}

func (v completedView) WithTx(tx pgx.Tx) completedStore { return completedView{v.w, asFakeTx(tx)} }

func (v completedView) GetCompletedCourseByStudentAndCourse(_ context.Context, studentID, courseID uuid.UUID) (db.GradesStudentCompletedCourse, error) {
	for _, c := range v.w.completed {
		if c.StudentID == studentID && c.CourseID == courseID {
			return c, nil
		}
	}
	return db.GradesStudentCompletedCourse{}, pgx.ErrNoRows
}

func (v completedView) GetCompletedCoursesByCourse(_ context.Context, courseID uuid.UUID) ([]db.GradesStudentCompletedCourse, error) {
	var out []db.GradesStudentCompletedCourse
	for _, c := range v.w.completed {
		if c.CourseID == courseID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (v completedView) GetCompletedCoursesByStudent(context.Context, uuid.UUID) ([]db.GradesStudentCompletedCourse, error) {
	return nil, nil
}

func (v completedView) GetTranscriptData(_ context.Context, studentID uuid.UUID) ([]db.GetTranscriptDataRow, error) {
	var out []db.GetTranscriptDataRow
	for _, c := range v.w.completed {
		if c.StudentID == studentID {
			out = append(out, db.GetTranscriptDataRow{Semester: c.Semester, CourseCode: c.CourseCode, CourseName: c.CourseName, Credits: c.Credits, GradePoint: c.GradePoint})
		}
	}
	return out, nil
}

func (v completedView) CalculateStudentGPA(context.Context, uuid.UUID) (db.CalculateStudentGPARow, error) {
	return db.CalculateStudentGPARow{}, nil
}

func (v completedView) CreateCompletedCourse(_ context.Context, p db.CreateCompletedCourseParams) (db.GradesStudentCompletedCourse, error) {
	if v.w.failCreateCompleted {
		return db.GradesStudentCompletedCourse{}, errors.New("unique violation")
	}
	row := db.GradesStudentCompletedCourse{
		ID: uuid.New(), StudentID: p.StudentID, StudentNumber: p.StudentNumber, CourseID: p.CourseID,
		CourseCode: p.CourseCode, CourseName: p.CourseName, Credits: p.Credits, Semester: p.Semester,
		InstructorID: p.InstructorID, AssessmentScores: p.AssessmentScores, WeightedAverage: p.WeightedAverage,
		GradePoint: p.GradePoint, GradingType: p.GradingType, GradingConfig: p.GradingConfig,
		ClassStatistics: p.ClassStatistics, IsAttendanceFailed: p.IsAttendanceFailed, FinalizedBy: p.FinalizedBy,
	}
	write(v.tx, func() { v.w.completed = append(v.w.completed, row) })
	return row, nil
}

func (v completedView) DeleteCompletedCourse(_ context.Context, p db.DeleteCompletedCourseParams) error {
	write(v.tx, func() {
		v.w.completed = slices.DeleteFunc(v.w.completed, func(c db.GradesStudentCompletedCourse) bool {
			return c.StudentID == p.StudentID && c.CourseCode == p.CourseCode
		})
	})
	return nil
}

func (v completedView) UpdateCompletedCourseAfterAppeal(_ context.Context, p db.UpdateCompletedCourseAfterAppealParams) error {
	write(v.tx, func() {
		for i, c := range v.w.completed {
			if c.StudentID == p.StudentID && c.CourseID == p.CourseID {
				c.AssessmentScores, c.WeightedAverage, c.GradePoint, c.GradingConfig =
					p.AssessmentScores, p.WeightedAverage, p.GradePoint, p.GradingConfig
				v.w.completed[i] = c
			}
		}
	})
	return nil
}

type outboxView struct {
	w  *gradesWorld
	tx *fakeTx
}

func (v outboxView) WithTx(tx pgx.Tx) outboxStore { return outboxView{v.w, asFakeTx(tx)} }

func (v outboxView) CreateOutboxEvent(_ context.Context, p db.CreateOutboxEventParams) (db.GradesOutboxEvent, error) {
	write(v.tx, func() { v.w.outbox = append(v.w.outbox, p) })
	return db.GradesOutboxEvent{}, nil
}

type periodView struct{ w *gradesWorld }

func (v periodView) GetActivePeriodBySemester(_ context.Context, semester string) (*sharedRepo.SimplePeriod, error) {
	if v.w.periodEnd == nil {
		return nil, pgx.ErrNoRows
	}
	return &sharedRepo.SimplePeriod{Semester: semester, PeriodEnd: *v.w.periodEnd, IsActive: true}, nil
}

type fakeSemesterClient struct {
	hardDeadline time.Time
	err          error
}

func (c fakeSemesterClient) GetSemesterInfo(_ context.Context, semester string) (*SemesterInfo, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &SemesterInfo{Name: semester, HardDeadline: c.hardDeadline}, nil
}

type recordingAudit struct{ w *gradesWorld }

func (a recordingAudit) Log(_ context.Context, e audit.AuditEvent) error {
	a.w.audits = append(a.w.audits, e)
	return nil
}

// Fixture: one course, "vize" 40% + "final" 60%, taught by instructorID in
// an open grading period that ends tomorrow.
var (
	instructorID = uuid.MustParse("0190a000-0000-7000-8000-00000000000a")
	gradeSchema  = []AssessmentSchemaItem{{Slug: "vize", Name: "Vize", Weight: 40}, {Slug: "final", Name: "Final", Weight: 60}}
)

func newGradesWorld(t *testing.T) (*gradesWorld, uuid.UUID) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	schema, err := json.Marshal(gradeSchema)
	require.NoError(t, err)
	end := time.Now().Add(24 * time.Hour)
	courseID := uuid.New()
	w := &gradesWorld{
		courses: map[uuid.UUID]db.GradesCoursesView{courseID: {
			ID: courseID, CourseCode: "BLM101", CourseName: "Programlamaya Giriş", Credits: 4,
			Semester: "2026-2027-Fall", InstructorID: instructorID, AssessmentSchema: schema,
		}},
		students:  map[uuid.UUID]db.GradesStudentsView{},
		prereqs:   map[string]bool{},
		periodEnd: &end,
	}
	return w, courseID
}

// enroll adds a student with the given vize/final scores: -1 leaves the
// assessment ungraded, absent records the student as absent.
func (w *gradesWorld) enroll(courseID uuid.UUID, number string, vize, final float64, attendanceFailed bool) db.GetRegistrationByIDRow {
	c := w.courses[courseID]
	reg := db.GetRegistrationByIDRow{
		ID: uuid.New(), StudentID: uuid.New(), CourseID: courseID, Semester: c.Semester,
		IsAttendanceFailed: pgtype.Bool{Bool: attendanceFailed, Valid: true}, StudentNumber: number,
		CourseCode: c.CourseCode, InstructorID: c.InstructorID, AssessmentSchema: c.AssessmentSchema,
	}
	w.regs = append(w.regs, reg)
	w.students[reg.StudentID] = db.GradesStudentsView{ID: reg.StudentID, StudentNumber: number, IsActive: pgtype.Bool{Bool: true, Valid: true}}
	for slug, v := range map[string]float64{"vize": vize, "final": final} {
		switch {
		case v == absent:
			w.scores = append(w.scores, db.GradesStudentAssessmentScore{ID: uuid.New(), RegistrationID: reg.ID, Slug: slug, IsAbsent: pgtype.Bool{Bool: true, Valid: true}})
		case v >= 0:
			n, _ := scoreToNumeric(v)
			w.scores = append(w.scores, db.GradesStudentAssessmentScore{ID: uuid.New(), RegistrationID: reg.ID, Slug: slug, Score: n})
		}
	}
	return reg
}

func (w *gradesWorld) service(semesters SemesterClient) *GradeService {
	return &GradeService{
		pool:             w,
		cacheRepo:        cacheView{w},
		registrationRepo: regView{w: w},
		scoreRepo:        scoreView{w: w},
		completedRepo:    completedView{w: w},
		outboxRepo:       outboxView{w: w},
		periodRepo:       periodView{w},
		auditLogger:      recordingAudit{w},
		semesterClient:   semesters,
	}
}

func (w *gradesWorld) eventTypes() []string {
	var out []string
	for _, e := range w.outbox {
		out = append(out, e.EventType)
	}
	return out
}

const absent = -2.0

func ptr(f float64) *float64 { return &f }

func openSemester() SemesterClient {
	return fakeSemesterClient{hardDeadline: time.Now().Add(30 * 24 * time.Hour)}
}

// ---------------------------------------------------------------- SubmitScore

func TestSubmitScore_ValidScore_StoredWithGradeSubmittedEvent(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "2024001", -1, -1, false)

	resp, err := w.service(openSemester()).SubmitScore(context.Background(), instructorID, courseID,
		dto.SubmitScoreRequest{RegistrationID: reg.ID, Slug: "vize", Score: ptr(72.5)})

	require.NoError(t, err)
	require.NotNil(t, resp.Score)
	assert.InDelta(t, 72.5, *resp.Score, 0.001)
	assert.Equal(t, "2024001", resp.StudentNumber)
	require.Len(t, w.scores, 1)
	assert.Equal(t, []string{"grade.submitted"}, w.eventTypes())
}

func TestSubmitScore_Absent_StoredWithoutEvent(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "2024001", -1, -1, false)

	resp, err := w.service(openSemester()).SubmitScore(context.Background(), instructorID, courseID,
		dto.SubmitScoreRequest{RegistrationID: reg.ID, Slug: "final", IsAbsent: true})

	require.NoError(t, err)
	assert.True(t, resp.IsAbsent)
	assert.Nil(t, resp.Score)
	assert.Empty(t, w.outbox, "an absence carries no score to publish")
}

func TestSubmitScore_Refused(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	cases := []struct {
		name      string
		setup     func(w *gradesWorld, reg db.GetRegistrationByIDRow)
		caller    uuid.UUID
		slug      string
		score     *float64
		semesters SemesterClient
		want      error
	}{
		{name: "another teacher", caller: uuid.New(), slug: "vize", score: ptr(50), want: gradeErrors.ErrNotCourseInstructor},
		{name: "unknown assessment", caller: instructorID, slug: "quiz", score: ptr(50), want: gradeErrors.ErrInvalidSlug},
		{name: "score above 100", caller: instructorID, slug: "vize", score: ptr(100.5), want: gradeErrors.ErrInvalidScore},
		{name: "negative score", caller: instructorID, slug: "vize", score: ptr(-1), want: gradeErrors.ErrInvalidScore},
		{
			name: "student failed on attendance", caller: instructorID, slug: "vize", score: ptr(50), want: gradeErrors.ErrAttendanceFailed,
			setup: func(w *gradesWorld, reg db.GetRegistrationByIDRow) {
				w.regs[0].IsAttendanceFailed = pgtype.Bool{Bool: true, Valid: true}
			},
		},
		{
			name: "score locked", caller: instructorID, slug: "vize", score: ptr(50), want: gradeErrors.ErrScoreLocked,
			setup: func(w *gradesWorld, reg db.GetRegistrationByIDRow) {
				n, _ := scoreToNumeric(40)
				w.scores = append(w.scores, db.GradesStudentAssessmentScore{RegistrationID: reg.ID, Slug: "vize", Score: n, IsLocked: pgtype.Bool{Bool: true, Valid: true}})
			},
		},
		{
			name: "grading period over", caller: instructorID, slug: "vize", score: ptr(50), want: gradeErrors.ErrGradingPeriodEnded,
			setup: func(w *gradesWorld, _ db.GetRegistrationByIDRow) { w.periodEnd = &past },
		},
		{name: "semester hard deadline passed", caller: instructorID, slug: "vize", score: ptr(50), want: gradeErrors.ErrGradingPeriodEnded, semesters: fakeSemesterClient{hardDeadline: past}},
		{name: "catalog unreachable", caller: instructorID, slug: "vize", score: ptr(50), want: gradeErrors.ErrSemesterInfoUnavailable, semesters: fakeSemesterClient{err: client.ErrUnavailable}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, courseID := newGradesWorld(t)
			reg := w.enroll(courseID, "2024001", -1, -1, false)
			if tc.setup != nil {
				tc.setup(w, reg)
			}
			semesters := tc.semesters
			if semesters == nil {
				semesters = openSemester()
			}
			scoresBefore := len(w.scores)

			_, err := w.service(semesters).SubmitScore(context.Background(), tc.caller, courseID,
				dto.SubmitScoreRequest{RegistrationID: reg.ID, Slug: tc.slug, Score: tc.score})

			assert.ErrorIs(t, err, tc.want)
			assert.Len(t, w.scores, scoresBefore, "nothing is written")
			assert.Empty(t, w.outbox)
		})
	}
}

func TestSubmitScore_NoGradingPeriodDefined_Allowed(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.periodEnd = nil
	reg := w.enroll(courseID, "2024001", -1, -1, false)

	_, err := w.service(openSemester()).SubmitScore(context.Background(), instructorID, courseID,
		dto.SubmitScoreRequest{RegistrationID: reg.ID, Slug: "vize", Score: ptr(80)})

	assert.NoError(t, err)
}

// ---------------------------------------------------------- BulkSubmitScores

func TestBulkSubmitScores_SkipsRowsItMayNotWrite(t *testing.T) {
	w, courseID := newGradesWorld(t)
	ok1 := w.enroll(courseID, "1", -1, -1, false)
	ok2 := w.enroll(courseID, "2", -1, -1, false)
	failed := w.enroll(courseID, "3", -1, -1, true)
	locked := w.enroll(courseID, "4", -1, -1, false)
	n, _ := scoreToNumeric(10)
	w.scores = append(w.scores, db.GradesStudentAssessmentScore{RegistrationID: locked.ID, Slug: "vize", Score: n, IsLocked: pgtype.Bool{Bool: true, Valid: true}})
	otherCourse := uuid.New()
	w.courses[otherCourse] = w.courses[courseID]
	foreign := w.enroll(otherCourse, "5", -1, -1, false)

	resp, err := w.service(openSemester()).BulkSubmitScores(context.Background(), instructorID, courseID, dto.BulkSubmitScoresRequest{
		Slug: "vize",
		Scores: []dto.BulkScoreEntry{
			{RegistrationID: ok1.ID, Score: ptr(70)},
			{RegistrationID: ok2.ID, IsAbsent: true},
			{RegistrationID: failed.ID, Score: ptr(70)},
			{RegistrationID: locked.ID, Score: ptr(70)},
			{RegistrationID: foreign.ID, Score: ptr(70)},
			{RegistrationID: ok1.ID, Score: ptr(170)},
			{RegistrationID: uuid.New(), Score: ptr(70)},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, resp.SuccessCount, "only the two valid rows are written")
	lockedScore, _ := scoreView{w: w}.GetScoreByRegistrationAndSlug(context.Background(), locked.ID, "vize")
	got, _ := utils.PgNumericToFloat64(lockedScore.Score)
	assert.InDelta(t, 10, got, 0.001, "a locked score keeps its value")
	assert.Equal(t, []string{"grade.submitted"}, w.eventTypes(), "the absence publishes nothing")
}

func TestBulkSubmitScores_UnknownSlug_Rejected(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", -1, -1, false)

	_, err := w.service(openSemester()).BulkSubmitScores(context.Background(), instructorID, courseID,
		dto.BulkSubmitScoresRequest{Slug: "quiz", Scores: []dto.BulkScoreEntry{{RegistrationID: reg.ID, Score: ptr(50)}}})

	assert.ErrorIs(t, err, gradeErrors.ErrInvalidSlug)
	assert.Empty(t, w.scores)
}

// -------------------------------------------------------------- AutoFinalize

func completedFor(t *testing.T, w *gradesWorld, studentID uuid.UUID) db.GradesStudentCompletedCourse {
	t.Helper()
	for _, c := range w.completed {
		if c.StudentID == studentID {
			return c
		}
	}
	t.Fatalf("no completed course for %s", studentID)
	return db.GradesStudentCompletedCourse{}
}

func TestAutoFinalize_HighMean_GradesAbsolutelyInOneTransaction(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.prereqs["BLM101"] = true
	top := w.enroll(courseID, "1", 90, 95, false)     // 93 → 4.00
	mid := w.enroll(courseID, "2", 70, 75, false)     // 73 → 2.25
	low := w.enroll(courseID, "3", 60, 40, false)     // 48 → 0.00
	abs := w.enroll(courseID, "4", 80, absent, false) // absent counts as 0 → 32 → 0.00
	ff := w.enroll(courseID, "5", 100, 100, true)     // attendance → FF regardless
	// Class mean of the four regular students is 61.5: absolute grading.

	res, err := w.service(openSemester()).AutoFinalize(context.Background(), courseID, instructorID)

	require.NoError(t, err)
	assert.Equal(t, string(db.GradesGradingTypeEnumAbsolute), res.GradingType)
	assert.Equal(t, 5, res.TotalStudents)
	assert.Equal(t, 2, res.PassingCount)
	assert.Equal(t, 3, res.FailingCount)
	assert.Equal(t, 1, res.AttendanceFailedCount)
	require.Len(t, w.txs, 1)
	assert.True(t, w.txs[0].committed)

	assert.Equal(t, db.GradesGradePointEnum400, completedFor(t, w, top.StudentID).GradePoint)
	assert.Equal(t, db.GradesGradePointEnum225, completedFor(t, w, mid.StudentID).GradePoint)
	assert.Equal(t, db.GradesGradePointEnum000, completedFor(t, w, low.StudentID).GradePoint)
	assert.Equal(t, db.GradesGradePointEnum000, completedFor(t, w, abs.StudentID).GradePoint)
	ffRow := completedFor(t, w, ff.StudentID)
	assert.Equal(t, db.GradesGradePointEnum000, ffRow.GradePoint)
	assert.True(t, ffRow.IsAttendanceFailed.Bool)

	assert.Empty(t, w.regs, "operational registrations are cleared")
	assert.Empty(t, w.scores, "operational scores are cleared")
	assert.Equal(t, []string{
		"grade.student.prerequisite.passed", "grade.student.prerequisite.passed", "grade.finalized",
	}, w.eventTypes(), "one prerequisite event per passing student, then the course event")
}

func TestAutoFinalize_LowMean_GradesOnTheCurve(t *testing.T) {
	w, courseID := newGradesWorld(t)
	best := w.enroll(courseID, "1", 60, 60, false)
	w.enroll(courseID, "2", 40, 40, false)
	worst := w.enroll(courseID, "3", 20, 20, false)

	res, err := w.service(openSemester()).AutoFinalize(context.Background(), courseID, instructorID)

	require.NoError(t, err)
	assert.Equal(t, string(db.GradesGradingTypeEnumRelative), res.GradingType, "a class mean under 60 switches to relative grading")
	bestRow := completedFor(t, w, best.StudentID)
	assert.True(t, isPassing(bestRow.GradePoint), "the top of a weak class still passes on the curve")
	assert.False(t, isPassing(completedFor(t, w, worst.StudentID).GradePoint))

	var cfg map[string]float64
	require.NoError(t, json.Unmarshal(bestRow.GradingConfig, &cfg))
	assert.InDelta(t, 40, cfg["class_mean"], 0.001, "the curve's statistics are frozen for later appeals")
	assert.Positive(t, cfg["student_z_score"])
}

func TestAutoFinalize_RetakeReplacesEarlierRecord(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 90, 90, false)
	w.completed = append(w.completed, db.GradesStudentCompletedCourse{StudentID: reg.StudentID, CourseCode: "BLM101", GradePoint: db.GradesGradePointEnum000})

	_, err := w.service(openSemester()).AutoFinalize(context.Background(), courseID, instructorID)

	require.NoError(t, err)
	require.Len(t, w.completed, 1)
	assert.Equal(t, db.GradesGradePointEnum400, w.completed[0].GradePoint)
}

func TestAutoFinalize_WriteFails_NothingIsKept(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.enroll(courseID, "1", 90, 90, false)
	w.enroll(courseID, "2", 50, 50, false)
	w.failCreateCompleted = true

	_, err := w.service(openSemester()).AutoFinalize(context.Background(), courseID, instructorID)

	require.Error(t, err)
	assert.Empty(t, w.completed)
	assert.Len(t, w.regs, 2, "registrations survive a failed finalize")
	assert.Len(t, w.scores, 4, "scores survive a failed finalize")
	assert.Empty(t, w.outbox)
}

// A student whose scores cannot be read must not be finalized out of the
// course: the same transaction deletes every registration and score, so
// skipping the student would erase their grade without a completed record.
func TestAutoFinalize_ScoresUnreadable_AbortsWithoutDeletingAnything(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.enroll(courseID, "1", 90, 90, false)
	unreadable := w.enroll(courseID, "2", 80, 80, false)
	w.failScoresFor = unreadable.ID

	_, err := w.service(openSemester()).AutoFinalize(context.Background(), courseID, instructorID)

	require.Error(t, err)
	assert.Empty(t, w.completed)
	assert.Len(t, w.regs, 2)
	assert.Len(t, w.scores, 4)
	assert.Empty(t, w.outbox)
}

func TestAutoFinalize_NoRegistrationsOrCourse_Fails(t *testing.T) {
	w, courseID := newGradesWorld(t)

	_, err := w.service(openSemester()).AutoFinalize(context.Background(), courseID, instructorID)
	assert.Error(t, err)

	_, err = w.service(openSemester()).AutoFinalize(context.Background(), uuid.New(), instructorID)
	assert.ErrorIs(t, err, gradeErrors.ErrCourseNotFound)
	for _, tx := range w.txs {
		assert.False(t, tx.committed, "nothing is committed for a course that cannot be finalized")
	}
}

// ------------------------------------------------------ Locking and finalize

func TestLockAssessmentBySlug_LastAssessment_RequestsFinalize(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.enroll(courseID, "1", 70, 80, false)
	w.enroll(courseID, "2", 60, 90, false)
	svc := w.service(openSemester())

	first, err := svc.LockAssessmentBySlug(context.Background(), instructorID, courseID, "vize")
	require.NoError(t, err)
	assert.Equal(t, 2, first.LockedCount)
	assert.False(t, first.FinalizeTriggered, "final is still open")
	assert.Empty(t, w.outbox)

	last, err := svc.LockAssessmentBySlug(context.Background(), instructorID, courseID, "final")
	require.NoError(t, err)
	assert.True(t, last.FinalizeTriggered)
	assert.Equal(t, []string{"grade.finalize.requested"}, w.eventTypes())
}

func TestLockAssessmentBySlug_Refused(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.enroll(courseID, "1", 70, 80, false)
	w.enroll(courseID, "2", -1, 90, false)
	svc := w.service(openSemester())

	_, err := svc.LockAssessmentBySlug(context.Background(), uuid.New(), courseID, "final")
	assert.ErrorIs(t, err, gradeErrors.ErrNotCourseInstructor)
	_, err = svc.LockAssessmentBySlug(context.Background(), instructorID, courseID, "quiz")
	assert.ErrorIs(t, err, gradeErrors.ErrInvalidSlug)
	_, err = svc.LockAssessmentBySlug(context.Background(), instructorID, courseID, "vize")
	assert.ErrorIs(t, err, gradeErrors.ErrIncompleteAssessment, "one student has no vize yet")

	for _, s := range w.scores {
		assert.False(t, s.IsLocked.Bool)
	}
}

func TestLockScore_AdminLocksTheLastScore_RequestsFinalize(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 70, 80, false)
	for i := range w.scores {
		w.scores[i].IsLocked = pgtype.Bool{Bool: w.scores[i].Slug == "vize", Valid: true}
	}

	require.NoError(t, w.service(openSemester()).LockScore(context.Background(), reg.ID, "final"))

	assert.Equal(t, []string{"grade.finalize.requested"}, w.eventTypes())
}

func TestLockScore_HardDeadlinePassed_Refused(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 70, 80, false)

	err := w.service(fakeSemesterClient{hardDeadline: time.Now().Add(-time.Hour)}).LockScore(context.Background(), reg.ID, "final")
	assert.ErrorIs(t, err, gradeErrors.ErrGradingPeriodEnded)
	err = w.service(openSemester()).UnlockScore(context.Background(), reg.ID, "quiz")
	assert.ErrorIs(t, err, gradeErrors.ErrScoreNotFound)
}

// -------------------------------------------------------------- ProcessAppeal

// finalizedCourse leaves one completed record as AutoFinalize would.
func finalizedCourse(w *gradesWorld, courseID uuid.UUID, gradingType db.GradesGradingTypeEnum, vize, final float64, gp db.GradesGradePointEnum) db.GradesStudentCompletedCourse {
	scores, _ := json.Marshal(map[string]float64{"vize": vize, "final": final})
	stats, _ := json.Marshal(map[string]any{"mean": 45.0, "stddev": 10.0, "total_students": 30})
	c := db.GradesStudentCompletedCourse{
		StudentID: uuid.New(), CourseID: courseID, CourseCode: "BLM101", Semester: "2026-2027-Fall",
		AssessmentScores: scores, ClassStatistics: stats, GradingType: gradingType, GradePoint: gp,
		WeightedAverage: utils.Float64ToPgNumeric(0.4*vize + 0.6*final),
	}
	w.completed = append(w.completed, c)
	return c
}

func TestProcessAppeal_FailToPass_UpdatesRecordAndPublishesPrerequisite(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.prereqs["BLM101"] = true
	c := finalizedCourse(w, courseID, db.GradesGradingTypeEnumAbsolute, 60, 45, db.GradesGradePointEnum050) // 51
	ctx := context.WithValue(context.Background(), CtxKeyUserID, "admin-1")

	resp, err := w.service(openSemester()).ProcessAppeal(ctx, dto.AppealScoreRequest{
		StudentID: c.StudentID, CourseID: courseID, Slug: "final", NewScore: ptr(65), Reason: "Okuma hatası düzeltildi",
	})

	require.NoError(t, err)
	assert.Equal(t, string(db.GradesGradePointEnum050), resp.OldGradePoint)
	assert.Equal(t, string(db.GradesGradePointEnum125), resp.NewGradePoint, "63 is a pass")
	assert.Equal(t, db.GradesGradePointEnum125, w.completed[0].GradePoint)
	assert.Equal(t, []string{"grade.student.prerequisite.passed"}, w.eventTypes())
	require.Len(t, w.audits, 1)
	assert.Equal(t, "admin-1", w.audits[0].ActorID)
	assert.Equal(t, "grade.appeal_processed", w.audits[0].Action)
}

func TestProcessAppeal_RelativeCourse_UsesFrozenStatistics(t *testing.T) {
	w, courseID := newGradesWorld(t)
	c := finalizedCourse(w, courseID, db.GradesGradingTypeEnumRelative, 40, 40, db.GradesGradePointEnum150)

	resp, err := w.service(openSemester()).ProcessAppeal(context.Background(), dto.AppealScoreRequest{
		StudentID: c.StudentID, CourseID: courseID, Slug: "final", NewScore: ptr(70), Reason: "Kağıt yeniden okundu",
	})

	require.NoError(t, err)
	wantGP, wantZ := calculateZScoreGradePoint(0.4*40+0.6*70, 45, 10)
	assert.Equal(t, string(wantGP), resp.NewGradePoint)
	assert.InDelta(t, 45, resp.FrozenClassMean, 0.001)
	var cfg map[string]float64
	require.NoError(t, json.Unmarshal(w.completed[0].GradingConfig, &cfg))
	assert.InDelta(t, wantZ, cfg["student_z_score"], 0.0001)
	assert.Empty(t, w.outbox, "not a prerequisite course")
}

func TestProcessAppeal_Refused(t *testing.T) {
	w, courseID := newGradesWorld(t)
	c := finalizedCourse(w, courseID, db.GradesGradingTypeEnumAbsolute, 60, 45, db.GradesGradePointEnum050)
	before := fmt.Sprint(w.completed)

	svc := w.service(openSemester())
	_, err := svc.ProcessAppeal(context.Background(), dto.AppealScoreRequest{StudentID: c.StudentID, CourseID: courseID, Slug: "quiz", NewScore: ptr(70)})
	assert.ErrorIs(t, err, gradeErrors.ErrInvalidSlug)
	_, err = svc.ProcessAppeal(context.Background(), dto.AppealScoreRequest{StudentID: c.StudentID, CourseID: courseID, Slug: "final"})
	assert.ErrorIs(t, err, gradeErrors.ErrInvalidScore, "a missing score is not a zero")
	_, err = svc.ProcessAppeal(context.Background(), dto.AppealScoreRequest{StudentID: uuid.New(), CourseID: courseID, Slug: "final", NewScore: ptr(70)})
	assert.ErrorIs(t, err, gradeErrors.ErrCourseNotFound)
	_, err = w.service(fakeSemesterClient{hardDeadline: time.Now().Add(-time.Hour)}).ProcessAppeal(context.Background(),
		dto.AppealScoreRequest{StudentID: c.StudentID, CourseID: courseID, Slug: "final", NewScore: ptr(70)})
	assert.ErrorIs(t, err, gradeErrors.ErrGradingPeriodEnded, "even an admin appeal stops at the hard deadline")

	assert.Equal(t, before, fmt.Sprint(w.completed))
	assert.Empty(t, w.outbox)
}

// Finalization deletes the registrations, so an appeal reads the student's
// frozen record instead.
func TestGetCourseResults_ListsFrozenRecords(t *testing.T) {
	w, courseID := newGradesWorld(t)
	c := finalizedCourse(w, courseID, db.GradesGradingTypeEnumAbsolute, 60, 45, db.GradesGradePointEnum050)
	w.completed[0].StudentNumber, w.completed[0].StudentFirstName, w.completed[0].StudentLastName = "2024001", "Ali", "Veli"
	w.completed[0].IsAttendanceFailed = pgtype.Bool{Bool: true, Valid: true}
	finalizedCourse(w, uuid.New(), db.GradesGradingTypeEnumAbsolute, 90, 90, db.GradesGradePointEnum400)

	resp, err := w.service(openSemester()).GetCourseResults(context.Background(), courseID)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1, "only this course's records")
	got := resp.Results[0]
	assert.Equal(t, c.StudentID, got.StudentID)
	assert.Equal(t, "2024001", got.StudentNumber)
	assert.Equal(t, "Ali", got.FirstName)
	assert.Equal(t, map[string]float64{"vize": 60, "final": 45}, got.Scores)
	assert.InDelta(t, 51, got.WeightedAverage, 0.001)
	assert.Equal(t, string(db.GradesGradePointEnum050), got.GradePoint)
	assert.True(t, got.IsAttendanceFailed)
}

func TestGetCourseResults_NotFinalized_EmptyList(t *testing.T) {
	w, courseID := newGradesWorld(t)

	resp, err := w.service(openSemester()).GetCourseResults(context.Background(), courseID)

	require.NoError(t, err)
	assert.NotNil(t, resp.Results, "an empty list, not null")
	assert.Empty(t, resp.Results)
}

// ------------------------------------------------------------------- Queries

func TestGetCourseStatus_ReportsPendingAssessments(t *testing.T) {
	w, courseID := newGradesWorld(t)
	w.enroll(courseID, "1", 70, -1, false)
	w.enroll(courseID, "2", 60, 90, false)
	svc := w.service(openSemester())

	st, err := svc.GetCourseStatus(context.Background(), instructorID, courseID, false)
	require.NoError(t, err)
	assert.Equal(t, 2, st.TotalStudents)
	require.Len(t, st.Assessments, 2)
	assert.True(t, st.Assessments[0].IsComplete)
	assert.Equal(t, 1, st.Assessments[1].PendingCount)
	assert.Contains(t, st.PendingMessage, "final")

	_, err = svc.GetCourseStatus(context.Background(), uuid.New(), courseID, false)
	assert.ErrorIs(t, err, gradeErrors.ErrNotCourseInstructor)
	_, err = svc.GetCourseStatus(context.Background(), uuid.New(), courseID, true)
	assert.NoError(t, err, "an admin reads any course")
}

func TestGetCourseStudents_ComputesRunningAverage(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 50, 100, false)

	res, err := w.service(openSemester()).GetCourseStudents(context.Background(), instructorID, courseID, false)

	require.NoError(t, err)
	require.Len(t, res.Students, 1)
	assert.Equal(t, reg.ID, res.Students[0].RegistrationID)
	require.NotNil(t, res.Students[0].CurrentAverage)
	assert.InDelta(t, 80, *res.Students[0].CurrentAverage, 0.001)
}

func TestGetTranscript_StudentReadsOnlyOwn(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 90, 90, false)
	svc := &StudentGradesService{cacheRepo: cacheView{w}, registrationRepo: regView{w: w}, scoreRepo: scoreView{w: w}, completedRepo: completedView{w: w}}
	w.completed = append(w.completed, db.GradesStudentCompletedCourse{StudentID: reg.StudentID, CourseCode: "BLM101", Credits: 4, GradePoint: db.GradesGradePointEnum400})

	_, err := svc.GetTranscript(context.Background(), uuid.New(), "student", reg.StudentID)
	assert.Error(t, err, "another student's transcript")

	tr, err := svc.GetTranscript(context.Background(), reg.StudentID, "student", reg.StudentID)
	require.NoError(t, err)
	assert.Contains(t, fmt.Sprint(tr), "BLM101")

	w.students[reg.StudentID] = db.GradesStudentsView{ID: reg.StudentID, IsActive: pgtype.Bool{Bool: false, Valid: true}}
	_, err = svc.GetTranscript(context.Background(), reg.StudentID, "student", reg.StudentID)
	assert.ErrorIs(t, err, gradeErrors.ErrStudentDeactivated)
}

// ------------------------------------------------------------ course lock

// Every write that grades, locks or finalizes a course must take the
// course's lock inside the transaction it writes in; a lock taken outside
// it, or in a transaction that is never committed, serializes nothing. The
// waiting itself is shown against Postgres in the integration tests.
func TestGradingWrites_EachTakesTheCourseLockInItsTransaction(t *testing.T) {
	ctx := context.Background()
	for name, run := range map[string]func(w *gradesWorld, svc *GradeService, courseID uuid.UUID, reg db.GetRegistrationByIDRow) error{
		"submit": func(_ *gradesWorld, svc *GradeService, courseID uuid.UUID, reg db.GetRegistrationByIDRow) error {
			_, err := svc.SubmitScore(ctx, instructorID, courseID, dto.SubmitScoreRequest{RegistrationID: reg.ID, Slug: "vize", Score: ptr(60)})
			return err
		},
		"bulk submit": func(_ *gradesWorld, svc *GradeService, courseID uuid.UUID, reg db.GetRegistrationByIDRow) error {
			_, err := svc.BulkSubmitScores(ctx, instructorID, courseID, dto.BulkSubmitScoresRequest{
				Slug: "vize", Scores: []dto.BulkScoreEntry{{RegistrationID: reg.ID, Score: ptr(60)}},
			})
			return err
		},
		"lock assessment": func(_ *gradesWorld, svc *GradeService, courseID uuid.UUID, _ db.GetRegistrationByIDRow) error {
			_, err := svc.LockAssessmentBySlug(ctx, instructorID, courseID, "vize")
			return err
		},
		"admin lock": func(_ *gradesWorld, svc *GradeService, _ uuid.UUID, reg db.GetRegistrationByIDRow) error {
			return svc.LockScore(ctx, reg.ID, "vize")
		},
		"admin unlock": func(_ *gradesWorld, svc *GradeService, _ uuid.UUID, reg db.GetRegistrationByIDRow) error {
			return svc.UnlockScore(ctx, reg.ID, "vize")
		},
		"finalize": func(_ *gradesWorld, svc *GradeService, courseID uuid.UUID, _ db.GetRegistrationByIDRow) error {
			_, err := svc.AutoFinalize(ctx, courseID, instructorID)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			w, courseID := newGradesWorld(t)
			reg := w.enroll(courseID, "2024001", 70, 80, false)

			require.NoError(t, run(w, w.service(openSemester()), courseID, reg))

			require.Len(t, w.courseLocks, 1)
			lock := w.courseLocks[0]
			assert.Equal(t, courseID, lock.course)
			require.NotNil(t, lock.tx, "the lock was taken outside a transaction")
			assert.True(t, lock.tx.committed, "the transaction holding the lock was never committed")
		})
	}
}
