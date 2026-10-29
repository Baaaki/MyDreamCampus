//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fall = "2026-2027-Fall"

type attendanceDB struct {
	pool       *pgxpool.Pool
	cache      *CacheRepository
	sessions   *SessionRepository
	attendance *AttendanceRepository
	course     uuid.UUID
}

func newAttendanceDB(t *testing.T) *attendanceDB {
	t.Helper()
	pool := pgtest.New(t, "attendance", "../sql/migrations")
	a := &attendanceDB{pool: pool, cache: NewCacheRepository(pool), sessions: NewSessionRepository(pool),
		attendance: NewAttendanceRepository(pool), course: uuid.New()}
	require.NoError(t, a.cache.UpsertCourseCache(context.Background(), db.UpsertCourseCacheParams{
		ID: utils.UUIDToPgtype(a.course), CourseCode: "BLM101", CourseName: "Programlamaya Giriş", Credits: 4,
		Semester: fall, InstructorID: utils.UUIDToPgtype(uuid.New()), HasLab: true,
	}))
	return a
}

func (a *attendanceDB) enroll(t *testing.T, number string) uuid.UUID {
	return a.enrollAt(t, number, time.Time{})
}

// enrollAt enrolls a student whose program was approved at enrolledAt; zero
// means from the start.
func (a *attendanceDB) enrollAt(t *testing.T, number string, enrolledAt time.Time) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	require.NoError(t, a.cache.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
		ID: utils.UUIDToPgtype(id), StudentNumber: number, IsActive: pgtype.Bool{Bool: true, Valid: true},
	}))
	require.NoError(t, a.cache.CreateEnrollmentCache(ctx, uuid.New(), id, a.course, fall, enrolledAt))
	return id
}

func (a *attendanceDB) session(t *testing.T, week int16, kind db.SessionTypeEnum) pgtype.UUID {
	return a.sessionAt(t, week, kind, time.Now())
}

func (a *attendanceDB) sessionAt(t *testing.T, week int16, kind db.SessionTypeEnum, now time.Time) pgtype.UUID {
	t.Helper()
	s, err := a.sessions.CreateAttendanceSession(context.Background(), db.CreateAttendanceSessionParams{
		CourseID: utils.UUIDToPgtype(a.course), InstructorID: utils.UUIDToPgtype(uuid.New()), Semester: fall, WeekNumber: week,
		SessionDate: pgtype.Date{Time: now, Valid: true}, QrSecret: "secret",
		StartedAt: pgtype.Timestamp{Time: now, Valid: true}, ExpiresAt: pgtype.Timestamp{Time: now.Add(time.Hour), Valid: true},
		SessionType: kind,
	})
	require.NoError(t, err)
	return s.ID
}

func (a *attendanceDB) scan(session pgtype.UUID, student uuid.UUID, week int16, kind db.SessionTypeEnum, qrTimestamp int64) db.CreateAttendanceRecordQRParams {
	return db.CreateAttendanceRecordQRParams{
		SessionID: session, StudentID: utils.UUIDToPgtype(student), CourseID: utils.UUIDToPgtype(a.course), Semester: fall,
		WeekNumber: week, ScannedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		QrTimestamp: pgtype.Int8{Int64: qrTimestamp, Valid: qrTimestamp != 0}, SessionType: kind,
	}
}

// The flush writes buffered scans in one UNNEST query whose session types
// travel as text[] and are cast per row, because pgx cannot encode a Go
// slice into the enum array. A retried flush must not double-count.
func TestBatchCreateQR_WritesEveryScanOnceAcrossRetries(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	ali, ayse := a.enroll(t, "1"), a.enroll(t, "2")
	theory, lab := a.session(t, 1, db.AttendanceSessionTypeEnumTheory), a.session(t, 1, db.AttendanceSessionTypeEnumLab)
	batch := []db.CreateAttendanceRecordQRParams{
		a.scan(theory, ali, 1, db.AttendanceSessionTypeEnumTheory, 1_790_000_000),
		a.scan(theory, ayse, 1, db.AttendanceSessionTypeEnumTheory, 0),
		a.scan(lab, ali, 1, db.AttendanceSessionTypeEnumLab, 1_790_000_030),
	}

	require.NoError(t, a.attendance.BatchCreateAttendanceRecordsQR(ctx, batch))
	require.NoError(t, a.attendance.BatchCreateAttendanceRecordsQR(ctx, batch), "a retried flush")

	records, err := a.attendance.GetAttendanceRecordsBySession(ctx, utils.PgtypeToUUID(theory))
	require.NoError(t, err)
	require.Len(t, records, 2)
	for _, r := range records {
		assert.Equal(t, "qr_scan", r.MarkedVia)
		assert.Equal(t, db.AttendanceSessionTypeEnumTheory, r.SessionType)
		if utils.PgtypeToUUID(r.StudentID) == ayse {
			assert.False(t, r.QrTimestamp.Valid, "the zero stand-in is stored as NULL")
		} else {
			assert.Equal(t, int64(1_790_000_000), r.QrTimestamp.Int64)
		}
	}
	labCount, err := a.attendance.GetStudentPresentCountByType(ctx, ali, a.course, fall, db.AttendanceSessionTypeEnumLab)
	require.NoError(t, err)
	assert.Equal(t, int64(1), labCount)
}

func TestManualMark_OverridesAScanAndALaterScanChangesNothing(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	ali := a.enroll(t, "1")
	s := a.session(t, 1, db.AttendanceSessionTypeEnumTheory)
	written, err := a.attendance.CreateAttendanceRecordQR(ctx, a.scan(s, ali, 1, db.AttendanceSessionTypeEnumTheory, 1))
	require.NoError(t, err)
	require.True(t, written)

	teacher := uuid.New()
	manual, err := a.attendance.CreateAttendanceRecordManual(ctx, db.CreateAttendanceRecordManualParams{
		SessionID: s, StudentID: utils.UUIDToPgtype(ali), CourseID: utils.UUIDToPgtype(a.course), Semester: fall, WeekNumber: 1,
		ManuallyMarkedBy: utils.UUIDToPgtype(teacher), ManualNote: utils.StringToPgText("Kart okuyucu arızalı"), SessionType: db.AttendanceSessionTypeEnumTheory,
	})
	require.NoError(t, err)
	assert.Equal(t, "manual", manual.MarkedVia)
	assert.Equal(t, teacher, utils.PgtypeToUUID(manual.ManuallyMarkedBy))

	written, err = a.attendance.CreateAttendanceRecordQR(ctx, a.scan(s, ali, 1, db.AttendanceSessionTypeEnumTheory, 2))
	require.NoError(t, err)
	assert.False(t, written, "the student is already present")
	n, err := a.attendance.GetSessionAttendanceCount(ctx, utils.PgtypeToUUID(s))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestFailingStudents_AbsencesCountedAgainstTheSessionsHeld(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	always, half, never := a.enroll(t, "1"), a.enroll(t, "2"), a.enroll(t, "3")
	var batch []db.CreateAttendanceRecordQRParams
	for week := int16(1); week <= 4; week++ {
		s := a.session(t, week, db.AttendanceSessionTypeEnumTheory)
		batch = append(batch, a.scan(s, always, week, db.AttendanceSessionTypeEnumTheory, 0))
		if week <= 2 {
			batch = append(batch, a.scan(s, half, week, db.AttendanceSessionTypeEnumTheory, 0))
		}
	}
	require.NoError(t, a.attendance.BatchCreateAttendanceRecordsQR(ctx, batch))
	exists, err := a.sessions.CheckSessionExists(ctx, a.course, 1, db.AttendanceSessionTypeEnumTheory)
	require.NoError(t, err)
	assert.True(t, exists)

	total, err := a.attendance.GetTotalSessionsByCourse(ctx, a.course, fall)
	require.NoError(t, err)
	require.Equal(t, int64(4), total)
	failing, err := a.attendance.GetFailingStudentsByCourse(ctx, a.course, fall, total, 1)
	require.NoError(t, err)

	absences := map[uuid.UUID]int64{}
	for _, f := range failing {
		absences[utils.PgtypeToUUID(f.StudentID)] = f.AbsentCount
	}
	assert.Equal(t, map[uuid.UUID]int64{half: 2, never: 4}, absences, "more than one absence out of four fails")

	stats, err := a.attendance.GetCourseAttendanceStats(ctx, a.course, fall)
	require.NoError(t, err)
	require.Len(t, stats, 3, "a student who never came still appears, with zero")
	assert.Equal(t, int64(0), stats[2].PresentCount)
}

// Finalize holds each student to the sessions run since their program was
// approved: weeks 1-3 ran before the late student's approval.
func TestCourseAttendanceByStudentAndType_HeldSinceEnrollment(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-10 * 24 * time.Hour)
	early := a.enroll(t, "1")
	late := a.enrollAt(t, "2", base.Add(3*24*time.Hour+time.Hour))
	var batch []db.CreateAttendanceRecordQRParams
	for week := int16(1); week <= 4; week++ {
		s := a.sessionAt(t, week, db.AttendanceSessionTypeEnumTheory, base.Add(time.Duration(week)*24*time.Hour))
		batch = append(batch, a.scan(s, early, week, db.AttendanceSessionTypeEnumTheory, 0))
		if week == 4 {
			batch = append(batch, a.scan(s, late, week, db.AttendanceSessionTypeEnumTheory, 0))
		}
	}
	a.sessionAt(t, 1, db.AttendanceSessionTypeEnumLab, base.Add(5*24*time.Hour))
	require.NoError(t, a.attendance.BatchCreateAttendanceRecordsQR(ctx, batch))

	rows, err := a.attendance.GetCourseAttendanceByStudentAndType(ctx, a.course, fall, db.AttendanceSessionTypeEnumTheory)

	require.NoError(t, err)
	got := map[uuid.UUID][2]int64{}
	for _, r := range rows {
		got[utils.PgtypeToUUID(r.StudentID)] = [2]int64{r.HeldCount, r.PresentCount}
	}
	assert.Equal(t, map[uuid.UUID][2]int64{early: {4, 4}, late: {1, 1}}, got, "held, present — the lab session is not counted")
}

// A changed approved program is cancelled and its replacement approved;
// the two events can be consumed in either order. Approved first must not
// let the cancel take the course both programs share off the roster.
func TestDeleteProgramEnrollments_OnlyTheCancelledProgramsRows(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	student := a.enroll(t, "1")
	other := uuid.New()
	require.NoError(t, a.cache.UpsertCourseCache(ctx, db.UpsertCourseCacheParams{
		ID: utils.UUIDToPgtype(other), CourseCode: "BLM102", CourseName: "Veri Yapıları", Credits: 4,
		Semester: fall, InstructorID: utils.UUIDToPgtype(uuid.New()),
	}))
	oldProgram, newProgram := uuid.New(), uuid.New()
	// enroll wrote the shared course under a random program: take it over
	// the way the old program's approval would have.
	require.NoError(t, a.cache.CreateEnrollmentCache(ctx, oldProgram, student, a.course, fall, time.Time{}))
	require.NoError(t, a.cache.CreateEnrollmentCache(ctx, oldProgram, student, other, fall, time.Time{}))

	// The new program keeps a.course and drops other; its approval lands first.
	approvedAt := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, a.cache.CreateEnrollmentCache(ctx, newProgram, student, a.course, fall, approvedAt))
	removed, err := a.cache.DeleteProgramEnrollments(ctx, oldProgram, student, []uuid.UUID{a.course, other})

	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)
	rows, err := a.cache.GetStudentEnrollmentsBySemester(ctx, student, fall)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, a.course, utils.PgtypeToUUID(rows[0].CourseID))
	assert.Equal(t, approvedAt, rows[0].EnrolledAt.Time, "the replacement's approval time holds")
}

func TestStudentHasRecordsInSemester_OnlyThatSemester(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	student, idle := a.enroll(t, "1"), a.enroll(t, "2")
	s := a.session(t, 1, db.AttendanceSessionTypeEnumTheory)
	require.NoError(t, a.attendance.BatchCreateAttendanceRecordsQR(ctx, []db.CreateAttendanceRecordQRParams{
		a.scan(s, student, 1, db.AttendanceSessionTypeEnumTheory, 0),
	}))

	for _, tc := range []struct {
		student  uuid.UUID
		semester string
		want     bool
	}{{student, fall, true}, {student, "2026-2027-Spring", false}, {idle, fall, false}} {
		got, err := a.attendance.StudentHasRecordsInSemester(ctx, tc.student, tc.semester)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.semester)
	}
}

// The finalization record and its events commit together, and the primary
// key turns a second finalize into "already done" with nothing queued.
func TestSaveCourseFinalization_OnceWithItsEvents(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	repo := NewFinalizationRepository(a.pool)
	params := db.InsertCourseFinalizationParams{
		CourseID: utils.UUIDToPgtype(a.course), Semester: fall, FinalizedBy: utils.UUIDToPgtype(uuid.New()),
		FinalizedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}, Result: []byte(`{"course_code":"BLM101"}`),
	}
	events := []OutboxEvent{
		{EventType: "attendance.semester.failed", RoutingKey: "attendance.semester.failed", Payload: []byte(`{"n":1}`)},
		{EventType: "attendance.semester.failed", RoutingKey: "attendance.semester.failed", Payload: []byte(`{"n":2}`)},
	}

	_, found, err := repo.GetCourseFinalization(ctx, a.course)
	require.NoError(t, err)
	assert.False(t, found)

	saved, err := repo.SaveCourseFinalization(ctx, params, events)
	require.NoError(t, err)
	assert.True(t, saved)

	saved, err = repo.SaveCourseFinalization(ctx, params, events)
	require.NoError(t, err)
	assert.False(t, saved, "second finalize is refused")

	var queued int
	require.NoError(t, a.pool.QueryRow(ctx, `SELECT count(*) FROM attendance.outbox_events`).Scan(&queued))
	assert.Equal(t, 2, queued, "the refused call queued nothing")

	row, found, err := repo.GetCourseFinalization(ctx, a.course)
	require.NoError(t, err)
	require.True(t, found)
	assert.JSONEq(t, `{"course_code":"BLM101"}`, string(row.Result))
}

// An event that cannot be queued takes the finalization record down with it,
// so the course stays open for a retry.
func TestSaveCourseFinalization_EventFails_RollsBackRecord(t *testing.T) {
	a := newAttendanceDB(t)
	ctx := context.Background()
	repo := NewFinalizationRepository(a.pool)
	params := db.InsertCourseFinalizationParams{
		CourseID: utils.UUIDToPgtype(a.course), Semester: fall, FinalizedBy: utils.UUIDToPgtype(uuid.New()),
		FinalizedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}, Result: []byte(`{}`),
	}
	// outbox_events.payload is JSONB: a non-JSON payload fails the insert.
	events := []OutboxEvent{
		{EventType: "attendance.semester.failed", RoutingKey: "attendance.semester.failed", Payload: []byte(`{"n":1}`)},
		{EventType: "attendance.semester.failed", RoutingKey: "attendance.semester.failed", Payload: []byte(`not json`)},
	}

	_, err := repo.SaveCourseFinalization(ctx, params, events)
	require.Error(t, err)

	_, found, err := repo.GetCourseFinalization(ctx, a.course)
	require.NoError(t, err)
	assert.False(t, found)
	var queued int
	require.NoError(t, a.pool.QueryRow(ctx, `SELECT count(*) FROM attendance.outbox_events`).Scan(&queued))
	assert.Zero(t, queued)
}
