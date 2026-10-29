package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/attendance/internal/dto"
	attErrors "github.com/baaaki/mydreamcampus/attendance/internal/errors"
	"github.com/baaaki/mydreamcampus/attendance/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/platform/clock/clocktest"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// attWorld is an in-memory stand-in for the attendance database and its
// Redis keys. Scans land in the Redis buffer as they do in production; the
// database only sees them through the Redis-down fallback or manual marks.
type attWorld struct {
	courses  map[uuid.UUID]db.CourseView
	students map[uuid.UUID]db.StudentView
	enrolled map[uuid.UUID][]uuid.UUID
	sessions map[uuid.UUID]db.AttendanceSession
	records  []db.AttendanceRecord
	outbox   []attOutboxEvent

	finalizations    map[uuid.UUID]db.CourseFinalization
	failFinalizeSave bool

	periodStart, periodEnd *time.Time
	periods                []sharedRepo.SimplePeriod
	enrolledAt             map[uuid.UUID]time.Time // student -> approved late; absent: from the start
	failSessionTotals      bool
	enrollmentSemester     string
	myEnrollments          []db.GetStudentEnrollmentsBySemesterRow

	redisDown        bool
	failBatchInsert  bool
	failBufferDelete bool
	enrolledSet      map[string]map[string]bool
	scanned          map[string]map[string]bool
	buffer           map[string]map[string]string
	cache            map[string]map[string]any
	invalidated      []uuid.UUID
}

type attOutboxEvent struct {
	eventType string
	payload   []byte
}

var errRedisDown = errors.New("redis: connection refused")

type attCacheView struct{ w *attWorld }

func (v attCacheView) GetCourseCacheByID(_ context.Context, id uuid.UUID) (db.CourseView, error) {
	c, ok := v.w.courses[id]
	if !ok {
		return c, pgx.ErrNoRows
	}
	return c, nil
}

func (v attCacheView) GetStudentCacheByID(_ context.Context, id uuid.UUID) (db.StudentView, error) {
	s, ok := v.w.students[id]
	if !ok {
		return s, pgx.ErrNoRows
	}
	return s, nil
}

func (v attCacheView) GetEnrolledStudentsByCourse(_ context.Context, courseID uuid.UUID, _ string) ([]db.GetEnrolledStudentsByCourseRow, error) {
	var out []db.GetEnrolledStudentsByCourseRow
	for _, id := range v.w.enrolled[courseID] {
		s := v.w.students[id]
		out = append(out, db.GetEnrolledStudentsByCourseRow{ID: s.ID, StudentNumber: s.StudentNumber, FirstName: s.FirstName, LastName: s.LastName, Email: s.Email})
	}
	return out, nil
}

func (v attCacheView) GetStudentEnrollmentsBySemester(_ context.Context, _ uuid.UUID, semester string) ([]db.GetStudentEnrollmentsBySemesterRow, error) {
	v.w.enrollmentSemester = semester
	return v.w.myEnrollments, nil
}

func (v attCacheView) CheckEnrollment(_ context.Context, studentID, courseID uuid.UUID, _ string) (bool, error) {
	for _, id := range v.w.enrolled[courseID] {
		if id == studentID {
			return true, nil
		}
	}
	return false, nil
}

type attSessionView struct{ w *attWorld }

func (v attSessionView) CheckSessionExists(_ context.Context, courseID uuid.UUID, week int16, st db.SessionTypeEnum) (bool, error) {
	for _, s := range v.w.sessions {
		if utils.PgUUIDToUUID(s.CourseID) == courseID && s.WeekNumber == week && s.SessionType == st {
			return true, nil
		}
	}
	return false, nil
}

func (v attSessionView) CreateAttendanceSession(_ context.Context, p db.CreateAttendanceSessionParams) (db.AttendanceSession, error) {
	id := uuid.New()
	s := db.AttendanceSession{
		ID: utils.UUIDToPgUUID(id), CourseID: p.CourseID, InstructorID: p.InstructorID, Semester: p.Semester,
		WeekNumber: p.WeekNumber, SessionDate: p.SessionDate, SessionType: p.SessionType, QrSecret: p.QrSecret,
		QrRotationInterval: p.QrRotationInterval, StartedAt: p.StartedAt, ExpiresAt: p.ExpiresAt,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}
	v.w.sessions[id] = s
	return s, nil
}

func (v attSessionView) GetActiveSessionByID(_ context.Context, id uuid.UUID) (db.AttendanceSession, error) {
	s, ok := v.w.sessions[id]
	if !ok || !s.IsActive.Bool {
		return db.AttendanceSession{}, pgx.ErrNoRows
	}
	return s, nil
}

func (v attSessionView) GetSessionByID(_ context.Context, id uuid.UUID) (db.AttendanceSession, error) {
	s, ok := v.w.sessions[id]
	if !ok {
		return db.AttendanceSession{}, pgx.ErrNoRows
	}
	return s, nil
}

func (v attSessionView) GetSessionsByCourse(_ context.Context, courseID uuid.UUID, _ string) ([]db.GetSessionsByCourseRow, error) {
	var out []db.GetSessionsByCourseRow
	for _, s := range v.w.sessions {
		if utils.PgUUIDToUUID(s.CourseID) == courseID {
			out = append(out, db.GetSessionsByCourseRow{ID: s.ID, WeekNumber: s.WeekNumber, IsActive: s.IsActive, SessionType: s.SessionType, StartedAt: s.StartedAt})
		}
	}
	return out, nil
}

func (v attSessionView) GetSessionsByDateRange(context.Context, db.GetSessionsByDateRangeParams) ([]db.GetSessionsByDateRangeRow, error) {
	return nil, nil
}

func (v attSessionView) DeactivateSession(_ context.Context, id uuid.UUID) error {
	s := v.w.sessions[id]
	s.IsActive = pgtype.Bool{Bool: false, Valid: true}
	v.w.sessions[id] = s
	return nil
}

type attRecordView struct{ w *attWorld }

func (v attRecordView) find(sessionID, studentID pgtype.UUID) int {
	for i, r := range v.w.records {
		if r.SessionID == sessionID && r.StudentID == studentID {
			return i
		}
	}
	return -1
}

func (v attRecordView) CreateAttendanceRecordQR(_ context.Context, p db.CreateAttendanceRecordQRParams) (bool, error) {
	if v.find(p.SessionID, p.StudentID) >= 0 {
		return false, nil // ON CONFLICT DO NOTHING
	}
	v.w.records = append(v.w.records, db.AttendanceRecord{
		ID: utils.UUIDToPgUUID(uuid.New()), SessionID: p.SessionID, StudentID: p.StudentID, CourseID: p.CourseID,
		Semester: p.Semester, WeekNumber: p.WeekNumber, SessionType: p.SessionType, MarkedVia: "qr", ScannedAt: p.ScannedAt,
	})
	return true, nil
}

func (v attRecordView) CreateAttendanceRecordManual(_ context.Context, p db.CreateAttendanceRecordManualParams) (db.AttendanceRecord, error) {
	r := db.AttendanceRecord{
		ID: utils.UUIDToPgUUID(uuid.New()), SessionID: p.SessionID, StudentID: p.StudentID, CourseID: p.CourseID,
		Semester: p.Semester, WeekNumber: p.WeekNumber, SessionType: p.SessionType, MarkedVia: "manual",
		ManuallyMarkedBy: p.ManuallyMarkedBy, ManualNote: p.ManualNote, ManuallyMarkedAt: utils.TimeToPgTimestamp(time.Now()),
	}
	if i := v.find(p.SessionID, p.StudentID); i >= 0 {
		r.ID = v.w.records[i].ID // ON CONFLICT updates the existing row
		v.w.records[i] = r
		return r, nil
	}
	v.w.records = append(v.w.records, r)
	return r, nil
}

func (v attRecordView) GetAttendanceRecordsBySession(context.Context, uuid.UUID) ([]db.GetAttendanceRecordsBySessionRow, error) {
	return nil, nil
}

func (v attRecordView) BatchCreateAttendanceRecordsQR(ctx context.Context, records []db.CreateAttendanceRecordQRParams) error {
	if v.w.failBatchInsert {
		return errors.New("batch insert: connection reset")
	}
	for _, r := range records {
		if _, err := v.CreateAttendanceRecordQR(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (v attRecordView) GetMarkedStudentsBySession(_ context.Context, sessionID uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for _, r := range v.w.records {
		if utils.PgUUIDToUUID(r.SessionID) == sessionID {
			out = append(out, utils.PgUUIDToUUID(r.StudentID))
		}
	}
	return out, nil
}

func (v attRecordView) GetSessionAttendanceCount(_ context.Context, sessionID uuid.UUID) (int64, error) {
	var n int64
	for _, r := range v.w.records {
		if utils.PgUUIDToUUID(r.SessionID) == sessionID {
			n++
		}
	}
	return n, nil
}

func (v attRecordView) GetStudentAttendanceByCourse(_ context.Context, studentID, courseID uuid.UUID, semester string) ([]db.GetStudentAttendanceByCourseRow, error) {
	var out []db.GetStudentAttendanceByCourseRow
	for _, r := range v.w.records {
		if utils.PgUUIDToUUID(r.StudentID) == studentID && utils.PgUUIDToUUID(r.CourseID) == courseID && r.Semester == semester {
			out = append(out, db.GetStudentAttendanceByCourseRow{WeekNumber: r.WeekNumber, MarkedVia: r.MarkedVia, SessionType: r.SessionType})
		}
	}
	return out, nil
}

func (v attRecordView) GetTotalSessionsByCourseAndType(_ context.Context, courseID uuid.UUID, semester string, st db.SessionTypeEnum) (int64, error) {
	if v.w.failSessionTotals {
		return 0, errors.New("connection reset")
	}
	var n int64
	for _, s := range v.w.sessions {
		if utils.PgUUIDToUUID(s.CourseID) == courseID && s.Semester == semester && s.SessionType == st {
			n++
		}
	}
	return n, nil
}

func (v attRecordView) GetCourseAttendanceByStudentAndType(_ context.Context, courseID uuid.UUID, semester string, st db.SessionTypeEnum) ([]db.GetCourseAttendanceByStudentAndTypeRow, error) {
	var out []db.GetCourseAttendanceByStudentAndTypeRow
	for _, studentID := range v.w.enrolled[courseID] {
		var held, present int64
		since, late := v.w.enrolledAt[studentID]
		for _, session := range v.w.sessions {
			if utils.PgUUIDToUUID(session.CourseID) == courseID && session.Semester == semester && session.SessionType == st &&
				(!late || !session.StartedAt.Time.Before(since)) {
				held++
			}
		}
		for _, r := range v.w.records {
			if utils.PgUUIDToUUID(r.StudentID) == studentID && utils.PgUUIDToUUID(r.CourseID) == courseID && r.Semester == semester && r.SessionType == st {
				present++
			}
		}
		student := v.w.students[studentID]
		out = append(out, db.GetCourseAttendanceByStudentAndTypeRow{
			StudentID: student.ID, StudentNumber: student.StudentNumber, FirstName: student.FirstName, LastName: student.LastName,
			Email: student.Email, HeldCount: held, PresentCount: present,
		})
	}
	return out, nil
}

// attFinalizationView keeps the record and its events together like the
// repository's transaction: a refused or failed save queues nothing.
type attFinalizationView struct{ w *attWorld }

func (v attFinalizationView) GetCourseFinalization(_ context.Context, courseID uuid.UUID) (db.CourseFinalization, bool, error) {
	row, ok := v.w.finalizations[courseID]
	return row, ok, nil
}

func (v attFinalizationView) SaveCourseFinalization(_ context.Context, p db.InsertCourseFinalizationParams, events []repository.OutboxEvent) (bool, error) {
	if v.w.failFinalizeSave {
		return false, errors.New("insert course finalization: connection reset")
	}
	courseID := utils.PgUUIDToUUID(p.CourseID)
	if _, ok := v.w.finalizations[courseID]; ok {
		return false, nil
	}
	v.w.finalizations[courseID] = db.CourseFinalization(p)
	for _, e := range events {
		v.w.outbox = append(v.w.outbox, attOutboxEvent{e.EventType, e.Payload})
	}
	return true, nil
}

type attRedisView struct{ w *attWorld }

func set(m map[string]map[string]bool, key string) map[string]bool {
	if m[key] == nil {
		m[key] = map[string]bool{}
	}
	return m[key]
}

func (v attRedisView) SetSessionCache(_ context.Context, id string, data map[string]any, _ time.Duration) error {
	if v.w.redisDown {
		return errRedisDown
	}
	v.w.cache[id] = data
	return nil
}

func (v attRedisView) AddEnrolledStudents(_ context.Context, id string, ids []uuid.UUID, _ time.Duration) error {
	if v.w.redisDown {
		return errRedisDown
	}
	for _, s := range ids {
		set(v.w.enrolledSet, id)[s.String()] = true
	}
	return nil
}

func (v attRedisView) IsStudentEnrolled(_ context.Context, id, studentID string) (bool, error) {
	if v.w.redisDown {
		return false, errRedisDown
	}
	return v.w.enrolledSet[id][studentID], nil
}

func (v attRedisView) AddToBuffer(_ context.Context, id, studentID, data string, _ time.Duration) (bool, error) {
	if v.w.redisDown {
		return false, errRedisDown
	}
	if set(v.w.scanned, id)[studentID] {
		return false, nil
	}
	set(v.w.scanned, id)[studentID] = true
	if v.w.buffer[id] == nil {
		v.w.buffer[id] = map[string]string{}
	}
	v.w.buffer[id][studentID] = data
	return true, nil
}

func (v attRedisView) GetBuffer(_ context.Context, id string) (map[string]string, error) {
	if v.w.redisDown {
		return nil, errRedisDown
	}
	return v.w.buffer[id], nil
}

func (v attRedisView) HDelBufferFields(_ context.Context, id string, studentIDs []string) error {
	if v.w.redisDown || v.w.failBufferDelete {
		return errRedisDown
	}
	for _, studentID := range studentIDs {
		delete(v.w.buffer[id], studentID)
	}
	return nil
}

func (v attRedisView) ClearSessionKeys(_ context.Context, id string) error {
	delete(v.w.cache, id)
	delete(v.w.enrolledSet, id)
	delete(v.w.scanned, id)
	return nil
}

func (v attRedisView) InvalidateStudentSummary(_ context.Context, studentID uuid.UUID, _ string) error {
	v.w.invalidated = append(v.w.invalidated, studentID)
	return nil
}

type attPeriodView struct{ w *attWorld }

func (v attPeriodView) GetActivePeriodBySemester(_ context.Context, semester string) (*sharedRepo.SimplePeriod, error) {
	if v.w.periodStart == nil {
		return nil, pgx.ErrNoRows
	}
	return &sharedRepo.SimplePeriod{Semester: semester, PeriodStart: *v.w.periodStart, PeriodEnd: *v.w.periodEnd, IsActive: true}, nil
}

func (v attPeriodView) GetAllPeriods(context.Context) ([]sharedRepo.SimplePeriod, error) {
	return v.w.periods, nil
}

type attSemesterClient struct {
	hardDeadline time.Time
	err          error
}

func (c attSemesterClient) GetSemesterInfo(_ context.Context, semester string) (*SemesterInfo, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &SemesterInfo{Name: semester, HardDeadline: c.hardDeadline}, nil
}

var (
	teacherID = uuid.MustParse("0190a000-0000-7000-8000-0000000000aa")
	semester  = "2026-2027-Fall"
)

// newAttWorld: one theory+lab course taught by teacherID, three enrolled
// students, an attendance period open since yesterday.
func newAttWorld(t *testing.T) (*attWorld, uuid.UUID, []uuid.UUID) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	start, end := time.Now().Add(-24*time.Hour), time.Now().Add(30*24*time.Hour)
	courseID := uuid.New()
	w := &attWorld{
		courses: map[uuid.UUID]db.CourseView{courseID: {
			ID: utils.UUIDToPgUUID(courseID), CourseCode: "FIZ101", CourseName: "Fizik I", Semester: semester,
			InstructorID: utils.UUIDToPgUUID(teacherID), TotalWeeks: pgtype.Int2{Int16: 14, Valid: true}, HasLab: true,
		}},
		students: map[uuid.UUID]db.StudentView{}, enrolled: map[uuid.UUID][]uuid.UUID{},
		sessions: map[uuid.UUID]db.AttendanceSession{}, periodStart: &start, periodEnd: &end,
		finalizations: map[uuid.UUID]db.CourseFinalization{},
		enrolledSet:   map[string]map[string]bool{}, scanned: map[string]map[string]bool{},
		buffer: map[string]map[string]string{}, cache: map[string]map[string]any{},
	}
	var ids []uuid.UUID
	for i, name := range []string{"Ali", "Ayşe", "Can"} {
		id := uuid.New()
		w.students[id] = db.StudentView{
			ID: utils.UUIDToPgUUID(id), StudentNumber: "2024" + string(rune('1'+i)),
			FirstName: pgtype.Text{String: name, Valid: true}, LastName: pgtype.Text{String: "Yılmaz", Valid: true},
			Email: pgtype.Text{String: name + "@uni.edu.tr", Valid: true}, IsActive: pgtype.Bool{Bool: true, Valid: true},
		}
		ids = append(ids, id)
	}
	w.enrolled[courseID] = ids
	return w, courseID, ids
}

func (w *attWorld) service(semesters SemesterClient) *AttendanceService {
	if semesters == nil {
		semesters = attSemesterClient{hardDeadline: time.Now().Add(60 * 24 * time.Hour)}
	}
	return &AttendanceService{
		cacheRepo: attCacheView{w}, sessionRepo: attSessionView{w}, attendanceRepo: attRecordView{w},
		finalizations: attFinalizationView{w}, qrService: NewQRService(), redisService: attRedisView{w},
		semesterClient: semesters, periodRepo: attPeriodView{w},
	}
}

func openTheory(t *testing.T, w *attWorld, courseID uuid.UUID, week int16) uuid.UUID {
	t.Helper()
	resp, err := w.service(nil).CreateSession(context.Background(), teacherID, dto.CreateSessionRequest{
		CourseID: courseID, WeekNumber: week, SessionType: "theory", DurationMinutes: 15,
	})
	require.NoError(t, err)
	return resp.SessionID
}

// closeTheory opens a theory session dated to startedAt, marks the given
// students present by hand and closes it, the way the weeks before a late
// approval look.
func closeTheory(t *testing.T, w *attWorld, courseID uuid.UUID, week int16, startedAt time.Time, present ...uuid.UUID) uuid.UUID {
	t.Helper()
	sessionID := openTheory(t, w, courseID, week)
	session := w.sessions[sessionID]
	session.StartedAt = utils.TimeToPgTimestamp(startedAt)
	w.sessions[sessionID] = session
	svc := w.service(nil)
	for _, student := range present {
		_, err := svc.CreateManualAttendance(context.Background(), sessionID, teacherID, false, dto.ManualAttendanceRequest{StudentID: student})
		require.NoError(t, err)
	}
	_, err := svc.CloseSession(context.Background(), sessionID, teacherID, false)
	require.NoError(t, err)
	return sessionID
}

func scanRequest(w *attWorld, sessionID uuid.UUID, at time.Time) dto.ScanQRRequest {
	s := w.sessions[sessionID]
	return dto.ScanQRRequest{QRPayload: NewQRService().GenerateQRPayload(sessionID.String(), s.QrSecret, at, qrRotation(s))}
}

// -------------------------------------------------------------- CreateSession

func TestCreateSession_Owner_OpensSessionAndWarmsRedis(t *testing.T) {
	w, courseID, students := newAttWorld(t)

	resp, err := w.service(nil).CreateSession(context.Background(), teacherID, dto.CreateSessionRequest{
		CourseID: courseID, WeekNumber: 3, SessionType: "lab", DurationMinutes: 20,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, resp.EnrolledStudentCount)
	assert.Equal(t, "FIZ101", resp.CourseCode)
	assert.WithinDuration(t, time.Now().Add(20*time.Minute), resp.ExpiresAt, 5*time.Second)
	s := w.sessions[resp.SessionID]
	assert.NotEmpty(t, s.QrSecret)
	assert.Equal(t, int16(DefaultQRRotation/time.Second), s.QrRotationInterval.Int16, "rotation is stored per session")
	assert.Len(t, w.enrolledSet[resp.SessionID.String()], len(students))
	assert.Equal(t, s.QrSecret, w.cache[resp.SessionID.String()]["qr_secret"])
}

func TestCreateSession_RedisDown_StillOpens(t *testing.T) {
	w, courseID, _ := newAttWorld(t)
	w.redisDown = true

	_, err := w.service(nil).CreateSession(context.Background(), teacherID, dto.CreateSessionRequest{
		CourseID: courseID, WeekNumber: 1, SessionType: "theory", DurationMinutes: 15,
	})

	assert.NoError(t, err, "the cache is a warm-up; scans fall back to the database")
}

func TestCreateSession_Refused(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(48 * time.Hour)
	cases := []struct {
		name      string
		setup     func(w *attWorld, courseID uuid.UUID)
		caller    uuid.UUID
		course    func(uuid.UUID) uuid.UUID
		kind      string
		semesters SemesterClient
		want      error
	}{
		{name: "another teacher's course", caller: uuid.New(), kind: "theory", want: attErrors.ErrForbidden},
		{name: "unknown course", caller: teacherID, kind: "theory", course: func(uuid.UUID) uuid.UUID { return uuid.New() }, want: attErrors.ErrCourseNotFound},
		{name: "lab on a course without lab hours", caller: teacherID, kind: "lab", want: attErrors.ErrLabNotAvailable,
			setup: func(w *attWorld, c uuid.UUID) { cv := w.courses[c]; cv.HasLab = false; w.courses[c] = cv }},
		{name: "second session for the week", caller: teacherID, kind: "theory", want: attErrors.ErrSessionAlreadyExists,
			setup: func(w *attWorld, c uuid.UUID) {
				_, _ = attSessionView{w}.CreateAttendanceSession(context.Background(), db.CreateAttendanceSessionParams{CourseID: utils.UUIDToPgUUID(c), WeekNumber: 5, SessionType: "theory"})
			}},
		{name: "semester over", caller: teacherID, kind: "theory", semesters: attSemesterClient{hardDeadline: past}, want: attErrors.ErrSemesterEnded},
		{name: "period not started", caller: teacherID, kind: "theory", want: attErrors.ErrPeriodNotStarted,
			setup: func(w *attWorld, _ uuid.UUID) { w.periodStart = &future }},
		{name: "period over", caller: teacherID, kind: "theory", want: attErrors.ErrPeriodEnded,
			setup: func(w *attWorld, _ uuid.UUID) { w.periodEnd = &past }},
		{name: "catalog unreachable", caller: teacherID, kind: "theory", semesters: attSemesterClient{err: client.ErrUnavailable}, want: attErrors.ErrSemesterInfoUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, courseID, _ := newAttWorld(t)
			if tc.setup != nil {
				tc.setup(w, courseID)
			}
			if tc.course != nil {
				courseID = tc.course(courseID)
			}
			before := len(w.sessions)

			_, err := w.service(tc.semesters).CreateSession(context.Background(), tc.caller, dto.CreateSessionRequest{
				CourseID: courseID, WeekNumber: 5, SessionType: tc.kind, DurationMinutes: 15,
			})

			assert.ErrorIs(t, err, tc.want)
			assert.Len(t, w.sessions, before)
		})
	}
}

// --------------------------------------------------------------------- ScanQR

func TestScanQR_EnrolledStudent_BufferedOnceInRedis(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)

	resp, err := svc.ScanQR(context.Background(), students[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	assert.Equal(t, "FIZ101", resp.CourseCode)
	assert.Len(t, w.buffer[sessionID.String()], 1)
	assert.Empty(t, w.records, "the flusher, not the scan, writes to the database")

	var buffered db.CreateAttendanceRecordQRParams
	require.NoError(t, json.Unmarshal([]byte(w.buffer[sessionID.String()][students[0].String()]), &buffered))
	assert.Equal(t, utils.UUIDToPgUUID(courseID), buffered.CourseID)

	_, err = svc.ScanQR(context.Background(), students[0], scanRequest(w, sessionID, time.Now()))
	assert.ErrorIs(t, err, attErrors.ErrAlreadyMarked)
	assert.Len(t, w.buffer[sessionID.String()], 1)
}

func TestScanQR_RedisDown_WritesToDatabaseOnce(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	w.redisDown = true
	svc := w.service(nil)

	_, err := svc.ScanQR(context.Background(), students[1], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err, "enrollment is checked in the database when Redis is down")
	require.Len(t, w.records, 1)
	assert.Equal(t, "qr", w.records[0].MarkedVia)

	_, err = svc.ScanQR(context.Background(), students[1], scanRequest(w, sessionID, time.Now()))
	assert.ErrorIs(t, err, attErrors.ErrAlreadyMarked)
	assert.Len(t, w.records, 1)
}

func TestScanQR_LateEnrollment_AcceptedFromDatabase(t *testing.T) {
	w, courseID, _ := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	late := uuid.New()
	w.students[late] = db.StudentView{ID: utils.UUIDToPgUUID(late)}
	w.enrolled[courseID] = append(w.enrolled[courseID], late) // after the set was warmed

	_, err := w.service(nil).ScanQR(context.Background(), late, scanRequest(w, sessionID, time.Now()))

	assert.NoError(t, err, "a miss in the warmed set is checked against the database")
}

func TestScanQR_Refused(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)
	ctx := context.Background()

	_, err := svc.ScanQR(ctx, uuid.New(), scanRequest(w, sessionID, time.Now()))
	assert.ErrorIs(t, err, attErrors.ErrNotEnrolled, "a student of another course")

	forwarded := scanRequest(w, sessionID, time.Now().Add(-3*DefaultQRRotation))
	_, err = svc.ScanQR(ctx, students[0], forwarded)
	assert.ErrorIs(t, err, attErrors.ErrInvalidQRCode, "a photo of an old code")

	forged := scanRequest(w, sessionID, time.Now())
	forged.QRPayload.Signature = NewQRService().GenerateQRPayload(sessionID.String(), "guessed-secret", time.Now(), DefaultQRRotation).Signature
	_, err = svc.ScanQR(ctx, students[0], forged)
	assert.ErrorIs(t, err, attErrors.ErrInvalidQRCode, "a code signed with the wrong secret")

	_, err = svc.ScanQR(ctx, students[0], dto.ScanQRRequest{QRPayload: dto.QRPayload{SessionID: "not-a-uuid"}})
	assert.ErrorIs(t, err, attErrors.ErrSessionNotFound)

	s := w.sessions[sessionID]
	s.ExpiresAt = utils.TimeToPgTimestamp(time.Now().Add(-time.Minute))
	w.sessions[sessionID] = s
	_, err = svc.ScanQR(ctx, students[0], scanRequest(w, sessionID, time.Now()))
	assert.ErrorIs(t, err, attErrors.ErrSessionExpired)

	assert.Empty(t, w.buffer[sessionID.String()])
	assert.Empty(t, w.records)
}

func TestScanQR_ClosedSession_NotActive(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	req := scanRequest(w, sessionID, time.Now())
	_, err := w.service(nil).CloseSession(context.Background(), sessionID, teacherID, false)
	require.NoError(t, err)

	_, err = w.service(nil).ScanQR(context.Background(), students[0], req)

	assert.ErrorIs(t, err, attErrors.ErrSessionNotActive, "a late scan hears the session ended, not that it does not exist")
}

// ---------------------------------------------------------- QR and manual mark

func TestGetQRCode_OwnerGetsAValidCode(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)

	qr, err := svc.GetQRCode(context.Background(), sessionID, teacherID)
	require.NoError(t, err)
	_, err = svc.ScanQR(context.Background(), students[2], dto.ScanQRRequest{QRPayload: qr.QRPayload})
	assert.NoError(t, err, "the displayed code is accepted")

	_, err = svc.GetQRCode(context.Background(), sessionID, uuid.New())
	assert.ErrorIs(t, err, attErrors.ErrForbidden)
}

func TestCreateManualAttendance_OwnerOrAdmin(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 2)
	svc := w.service(nil)
	ctx := context.Background()

	resp, err := svc.CreateManualAttendance(ctx, sessionID, teacherID, false, dto.ManualAttendanceRequest{StudentID: students[0], Note: "Rapor getirdi"})
	require.NoError(t, err)
	assert.Equal(t, "manual", resp.MarkedVia)
	assert.Equal(t, "Ali Yılmaz", resp.StudentName)
	assert.Equal(t, []uuid.UUID{students[0]}, w.invalidated, "the student's cached summary is dropped")

	_, err = svc.CreateManualAttendance(ctx, sessionID, uuid.New(), true, dto.ManualAttendanceRequest{StudentID: students[1]})
	assert.NoError(t, err, "an admin may mark in any session")

	_, err = svc.CreateManualAttendance(ctx, sessionID, uuid.New(), false, dto.ManualAttendanceRequest{StudentID: students[2]})
	assert.ErrorIs(t, err, attErrors.ErrForbidden)
	_, err = svc.CreateManualAttendance(ctx, sessionID, teacherID, false, dto.ManualAttendanceRequest{StudentID: uuid.New()})
	assert.ErrorIs(t, err, attErrors.ErrNotEnrolled)

	assert.Len(t, w.records, 2)
}

// --------------------------------------------------------------- CloseSession

func TestCloseSession_WritesBufferedScansToDatabase(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)
	ctx := context.Background()
	_, err := svc.ScanQR(ctx, students[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	_, err = svc.CreateManualAttendance(ctx, sessionID, teacherID, false, dto.ManualAttendanceRequest{StudentID: students[1]})
	require.NoError(t, err)
	require.Len(t, w.buffer[sessionID.String()], 1)

	resp, err := svc.CloseSession(ctx, sessionID, teacherID, false)

	require.NoError(t, err)
	assert.Equal(t, 3, resp.Summary.TotalEnrolled)
	assert.Equal(t, 2, resp.Summary.PresentCount, "the scan and the manual mark, each once")
	assert.Equal(t, 1, resp.Summary.AbsentCount)
	assert.False(t, w.sessions[sessionID].IsActive.Bool)
	assert.Empty(t, w.buffer[sessionID.String()], "the close drains the buffer instead of waiting for the flusher")
	assert.Len(t, w.records, 2, "the scan is in the database the moment the session is closed")
	assert.Nil(t, w.enrolledSet[sessionID.String()])

	_, err = svc.CloseSession(ctx, sessionID, teacherID, false)
	assert.ErrorIs(t, err, attErrors.ErrSessionNotActive)
	_, err = svc.CloseSession(ctx, openTheory(t, w, courseID, 2), uuid.New(), false)
	assert.ErrorIs(t, err, attErrors.ErrForbidden)
}

func TestCloseSession_FlushFails_ClosesAndKeepsTheBuffer(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)
	ctx := context.Background()
	_, err := svc.ScanQR(ctx, students[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	_, err = svc.CreateManualAttendance(ctx, sessionID, teacherID, false, dto.ManualAttendanceRequest{StudentID: students[1]})
	require.NoError(t, err)
	w.failBatchInsert = true

	resp, err := svc.CloseSession(ctx, sessionID, teacherID, false)

	require.NoError(t, err, "the flusher retries; the close must not fail over it")
	assert.False(t, w.sessions[sessionID].IsActive.Bool)
	assert.Len(t, w.buffer[sessionID.String()], 1, "an unwritten scan must survive the close")
	assert.Equal(t, 2, resp.Summary.PresentCount, "one in the database, one still in the buffer")
}

// The rows went in but the HDEL failed: the same scan is in the database
// and in the buffer, and must still count once.
func TestCloseSession_ScanInDatabaseAndBuffer_CountedOnce(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)
	ctx := context.Background()
	_, err := svc.ScanQR(ctx, students[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	w.failBufferDelete = true

	resp, err := svc.CloseSession(ctx, sessionID, teacherID, false)

	require.NoError(t, err)
	require.Len(t, w.records, 1)
	require.Len(t, w.buffer[sessionID.String()], 1)
	assert.Equal(t, 1, resp.Summary.PresentCount)
}

// --------------------------------------------------------- FinalizeAttendance

// holdSessions holds n closed sessions of a kind and marks each student
// present in the first present[i] of them.
func holdSessions(w *attWorld, courseID uuid.UUID, kind db.SessionTypeEnum, n int, students []uuid.UUID, present []int) {
	for week := 0; week < n; week++ {
		s, _ := attSessionView{w}.CreateAttendanceSession(context.Background(), db.CreateAttendanceSessionParams{
			CourseID: utils.UUIDToPgUUID(courseID), Semester: semester, WeekNumber: int16(week + 1), SessionType: kind,
		})
		_ = attSessionView{w}.DeactivateSession(context.Background(), utils.PgUUIDToUUID(s.ID))
		for i, id := range students {
			if week < present[i] {
				_, _ = attRecordView{w}.CreateAttendanceRecordQR(context.Background(), db.CreateAttendanceRecordQRParams{
					SessionID: s.ID, StudentID: utils.UUIDToPgUUID(id), CourseID: utils.UUIDToPgUUID(courseID), Semester: semester, SessionType: kind,
				})
			}
		}
	}
}

func TestFinalizeAttendance_FailsStudentsBelowThreshold(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	// 14 theory (needs 10) and 14 lab (needs 11) sessions.
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{10, 9, 14})
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumLab, 14, s, []int{11, 5, 10})

	resp, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	require.NoError(t, err)
	assert.Equal(t, 10, resp.Thresholds.TheoryMinRequired)
	assert.Equal(t, 11, resp.Thresholds.LabMinRequired)
	assert.Equal(t, 1, resp.FinalizationSummary.PassingCount)
	assert.Equal(t, 2, resp.FinalizationSummary.FailingCount)
	byStudent := map[uuid.UUID]dto.FailedStudent{}
	for _, f := range resp.FailedStudents {
		byStudent[f.StudentID] = f
	}
	assert.NotContains(t, byStudent, s[0], "exactly at both thresholds passes")
	assert.Equal(t, "both", byStudent[s[1]].FailedType)
	assert.Equal(t, "lab", byStudent[s[2]].FailedType)
	assert.Equal(t, 4, byStudent[s[2]].Lab.AbsentCount)

	require.Len(t, w.outbox, 2)
	assert.Equal(t, 2, resp.EventsPublished)
	var ev struct {
		EventType string                                `json:"event_type"`
		Data      dto.AttendanceSemesterFailedEventData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.outbox[0].payload, &ev))
	assert.Equal(t, "FIZ101", ev.Data.CourseCode)
	assert.NotEmpty(t, ev.Data.StudentEmail, "notification e-mails the student")
}

func TestFinalizeAttendance_Refused(t *testing.T) {
	w, courseID, _ := newAttWorld(t)

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, uuid.New(), semester)
	assert.ErrorIs(t, err, attErrors.ErrForbidden)
	_, err = w.service(nil).FinalizeAttendance(context.Background(), uuid.New(), teacherID, semester)
	assert.ErrorIs(t, err, attErrors.ErrCourseNotFound)
	assert.Empty(t, w.outbox)
}

// With the session totals unreadable every threshold reads as zero: nobody
// would fail on attendance and the finalize would report success.
func TestFinalizeAttendance_SessionTotalsUnreadable_Fails(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{0, 0, 0})
	w.failSessionTotals = true

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	assert.Error(t, err)
	assert.Empty(t, w.outbox)
}

// A second finalize must not queue every failed student's event again.
func TestFinalizeAttendance_SecondCall_RefusedWithoutNewEvents(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 3, 14})
	svc := w.service(nil)

	first, err := svc.FinalizeAttendance(context.Background(), courseID, teacherID, semester)
	require.NoError(t, err)
	require.Len(t, w.outbox, 1)

	_, err = svc.FinalizeAttendance(context.Background(), courseID, teacherID, semester)
	assert.ErrorIs(t, err, attErrors.ErrAttendanceFinalized)
	assert.Len(t, w.outbox, 1, "no event is queued twice")

	status, err := svc.GetFinalization(context.Background(), courseID, teacherID, false)
	require.NoError(t, err)
	require.True(t, status.Finalized)
	assert.Equal(t, first.FinalizationSummary, status.Result.FinalizationSummary)
	assert.Equal(t, first.FailedStudents, status.Result.FailedStudents)
}

// A concurrent call that wins the insert leaves this one with a refusal,
// not a second set of events.
func TestFinalizeAttendance_LostRace_Refused(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 3, 14})
	svc := w.service(nil)
	svc.finalizations = racingFinalizations{attFinalizationView{w}}

	_, err := svc.FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	assert.ErrorIs(t, err, attErrors.ErrAttendanceFinalized)
	assert.Empty(t, w.outbox)
}

// racingFinalizations reports "not finalized" on the read and lets another
// caller finalize just before the save.
type racingFinalizations struct{ attFinalizationView }

func (r racingFinalizations) GetCourseFinalization(context.Context, uuid.UUID) (db.CourseFinalization, bool, error) {
	return db.CourseFinalization{}, false, nil
}

func (r racingFinalizations) SaveCourseFinalization(ctx context.Context, p db.InsertCourseFinalizationParams, events []repository.OutboxEvent) (bool, error) {
	r.w.finalizations[utils.PgUUIDToUUID(p.CourseID)] = db.CourseFinalization{CourseID: p.CourseID}
	return r.attFinalizationView.SaveCourseFinalization(ctx, p, events)
}

func TestFinalizeAttendance_SaveFails_NothingQueuedAndRetryable(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 3, 14})
	w.failFinalizeSave = true

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)
	require.Error(t, err)
	assert.Empty(t, w.outbox)

	w.failFinalizeSave = false
	_, err = w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)
	require.NoError(t, err)
	assert.Len(t, w.outbox, 1)
}

func TestFinalizeAttendance_OpenSession_Refused(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 13, s, []int{13, 13, 13})
	openTheory(t, w, courseID, 14)

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	assert.ErrorIs(t, err, attErrors.ErrOpenSessionExists)
	assert.Empty(t, w.finalizations)
}

// bufferScan leaves a QR scan of studentID in the Redis buffer of the
// course's closed session for week, as if the flusher had not reached it.
func bufferScan(t *testing.T, w *attWorld, courseID uuid.UUID, week int16, studentID uuid.UUID) {
	t.Helper()
	for id, session := range w.sessions {
		if utils.PgUUIDToUUID(session.CourseID) != courseID || session.WeekNumber != week {
			continue
		}
		data, err := json.Marshal(db.CreateAttendanceRecordQRParams{
			SessionID: session.ID, StudentID: utils.UUIDToPgUUID(studentID), CourseID: session.CourseID,
			Semester: session.Semester, WeekNumber: week, SessionType: session.SessionType,
		})
		require.NoError(t, err)
		if w.buffer[id.String()] == nil {
			w.buffer[id.String()] = map[string]string{}
		}
		w.buffer[id.String()][studentID.String()] = string(data)
		return
	}
	t.Fatalf("no session for week %d", week)
}

// The last scan of a student one short of the threshold is still in Redis.
// Finalizing without it would fail them for good.
func TestFinalizeAttendance_BufferedScan_FlushedAndCounted(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 9, 14})
	bufferScan(t, w, courseID, 14, s[1])

	resp, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	require.NoError(t, err)
	assert.Equal(t, 0, resp.FinalizationSummary.FailingCount, "10 of 14 with the buffered scan")
	assert.Empty(t, resp.FailedStudents)
	assert.Empty(t, w.outbox)
	for id := range w.sessions {
		assert.Empty(t, w.buffer[id.String()])
	}
}

func TestFinalizeAttendance_BufferCannotBeFlushed_RefusedAndNothingRecorded(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 9, 14})
	bufferScan(t, w, courseID, 14, s[1])
	w.failBatchInsert = true

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	assert.ErrorIs(t, err, attErrors.ErrScansPending)
	assert.Empty(t, w.finalizations, "a finalization cannot be undone; it must wait for the scans")
	assert.Empty(t, w.outbox)

	w.failBatchInsert = false
	_, err = w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)
	require.NoError(t, err, "once the scans are written the finalize goes through")
	assert.Empty(t, w.outbox)
}

func TestFinalizeAttendance_BufferUnreadable_Refused(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 9, 14})
	w.redisDown = true

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)

	assert.Error(t, err, "with Redis unreadable nobody can tell whether scans are pending")
	assert.Empty(t, w.finalizations)
	assert.Empty(t, w.outbox)
}

// Under another semester name the course has no sessions and nobody would
// fail; an empty semester means the course's own.
func TestFinalizeAttendance_Semester(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 14, s, []int{14, 3, 14})

	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, "2020-2021-Fall")
	assert.ErrorIs(t, err, attErrors.ErrSemesterMismatch)
	assert.Empty(t, w.outbox)

	resp, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, "")
	require.NoError(t, err)
	assert.Equal(t, semester, resp.Semester)
	assert.Equal(t, 1, resp.FinalizationSummary.FailingCount)
}

func TestCreateSession_AfterFinalize_Refused(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	holdSessions(w, courseID, db.AttendanceSessionTypeEnumTheory, 13, s, []int{13, 13, 13})
	_, err := w.service(nil).FinalizeAttendance(context.Background(), courseID, teacherID, semester)
	require.NoError(t, err)

	_, err = w.service(nil).CreateSession(context.Background(), teacherID, dto.CreateSessionRequest{
		CourseID: courseID, WeekNumber: 14, SessionType: "theory", DurationMinutes: 15,
	})

	assert.ErrorIs(t, err, attErrors.ErrAttendanceFinalized)
}

func TestGetFinalization_NotFinalizedYet_AndOwnership(t *testing.T) {
	w, courseID, _ := newAttWorld(t)
	svc := w.service(nil)

	status, err := svc.GetFinalization(context.Background(), courseID, teacherID, false)
	require.NoError(t, err)
	assert.False(t, status.Finalized)
	assert.Nil(t, status.Result)

	_, err = svc.GetFinalization(context.Background(), courseID, uuid.New(), false)
	assert.ErrorIs(t, err, attErrors.ErrForbidden)
	_, err = svc.GetFinalization(context.Background(), courseID, uuid.New(), true)
	assert.NoError(t, err, "an admin reads any course")
}

// ----------------------------------------------------------- GetMyAttendance

func TestCurrentSemester_PicksPeriodHoldingNow(t *testing.T) {
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 12, 0, 0, 0, time.UTC) }
	periods := []sharedRepo.SimplePeriod{
		{Semester: "2025-2026-Spring", PeriodStart: day(2, 1), PeriodEnd: day(6, 1), IsActive: true},
		{Semester: "2026-2027-Fall", PeriodStart: day(9, 15), PeriodEnd: day(12, 31), IsActive: true},
		{Semester: "2026-2027-Draft", PeriodStart: day(9, 1), PeriodEnd: day(12, 31), IsActive: false},
	}

	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"inside the fall period", day(10, 11), "2026-2027-Fall"},
		{"between periods falls back to the latest started", day(7, 1), "2025-2026-Spring"},
		{"an inactive period never counts", day(9, 5), "2025-2026-Spring"},
		{"before the first period", day(1, 1), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, currentSemester(periods, tc.now))
		})
	}
}

// Without a semester the page used to come back empty; the service clock
// (time machine included) now chooses it.
func TestGetMyAttendance_NoSemester_UsesActivePeriod(t *testing.T) {
	w, _, s := newAttWorld(t)
	now := time.Date(2026, time.October, 11, 9, 0, 0, 0, time.UTC)
	clocktest.Freeze(t, now)
	w.periods = []sharedRepo.SimplePeriod{
		{Semester: "2025-2026-Spring", PeriodStart: now.AddDate(0, -8, 0), PeriodEnd: now.AddDate(0, -4, 0), IsActive: true},
		{Semester: semester, PeriodStart: now.AddDate(0, 0, -20), PeriodEnd: now.AddDate(0, 2, 0), IsActive: true},
	}

	resp, err := w.service(nil).GetMyAttendance(context.Background(), s[0], "")
	require.NoError(t, err)
	assert.Equal(t, semester, resp.Semester)
	assert.Equal(t, semester, w.enrollmentSemester)

	resp, err = w.service(nil).GetMyAttendance(context.Background(), s[0], "2025-2026-Spring")
	require.NoError(t, err)
	assert.Equal(t, "2025-2026-Spring", resp.Semester, "an explicit semester is kept")
}

// The student pages iterate these lists; null instead of [] crashed them.
func TestGetMyAttendance_NothingToList_MarshalsEmptyArrays(t *testing.T) {
	w, courseID, s := newAttWorld(t)

	resp, err := w.service(nil).GetMyAttendance(context.Background(), s[0], semester)
	require.NoError(t, err)
	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, jsonField(t, raw, "courses"))

	w.myEnrollments = []db.GetStudentEnrollmentsBySemesterRow{{CourseID: utils.UUIDToPgtype(courseID), CourseCode: "CENG101"}}
	resp, err = w.service(nil).GetMyAttendance(context.Background(), s[0], semester)
	require.NoError(t, err)
	require.Len(t, resp.Courses, 1)
	raw, err = json.Marshal(resp.Courses[0])
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, jsonField(t, raw, "weekly_records"))
}

// Weeks 1-4 are closed and the student missed week 2. Week 5 is open and the
// student's scan still waits in the buffer: the page read 3 of 5, below the
// minimum of 4, right after a successful scan.
func TestGetMyAttendance_ScanAwaitingFlush_CountedAsAttended(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	svc := w.service(nil)
	ctx := context.Background()
	for week := int16(1); week <= 4; week++ {
		sessionID := openTheory(t, w, courseID, week)
		if week != 2 {
			_, err := svc.ScanQR(ctx, s[0], scanRequest(w, sessionID, time.Now()))
			require.NoError(t, err)
		}
		_, err := svc.CloseSession(ctx, sessionID, teacherID, false)
		require.NoError(t, err)
	}
	open := openTheory(t, w, courseID, 5)
	_, err := svc.ScanQR(ctx, s[0], scanRequest(w, open, time.Now()))
	require.NoError(t, err)
	w.myEnrollments = []db.GetStudentEnrollmentsBySemesterRow{{CourseID: utils.UUIDToPgtype(courseID), CourseCode: "FIZ101"}}

	attended := &dto.SessionTypeAttendance{PresentCount: 4, AbsentCount: 1, TotalSessions: 5, MinRequired: 4, Passed: true}

	resp, err := svc.GetMyAttendance(ctx, s[0], semester)
	require.NoError(t, err)
	require.Len(t, resp.Courses, 1)
	assert.Equal(t, attended, resp.Courses[0].Theory)
	assert.Len(t, resp.Courses[0].WeeklyRecords, 4, "the buffered scan is listed as attended")
	assert.Len(t, resp.Courses[0].MissedSessions, 1)
	assert.Nil(t, resp.Courses[0].Lab, "no lab session was held")

	other, err := svc.GetMyAttendance(ctx, s[1], semester)
	require.NoError(t, err)
	assert.Equal(t, 4, other.Courses[0].Theory.TotalSessions, "a student who has not scanned yet is not marked absent while it is open")

	w.redisDown = true
	resp, err = svc.GetMyAttendance(ctx, s[0], semester)
	require.NoError(t, err)
	assert.Equal(t, &dto.SessionTypeAttendance{PresentCount: 3, AbsentCount: 1, TotalSessions: 4, MinRequired: 3, Passed: true}, resp.Courses[0].Theory,
		"without the buffer the open session waits for the flush")
	w.redisDown = false

	require.NoError(t, FlushSessionBuffer(ctx, attRecordView{w}, attRedisView{w}, open.String()))
	resp, err = svc.GetMyAttendance(ctx, s[0], semester)
	require.NoError(t, err)
	assert.Equal(t, attended, resp.Courses[0].Theory, "counted once after the flush")
	assert.Len(t, resp.Courses[0].WeeklyRecords, 4)
}

// Approved in week 5, after four closed sessions: the student could not scan
// into those, so they are neither held nor missed for them.
func TestGetMyAttendance_ApprovedLate_HeldOnlyToLaterSessions(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	svc := w.service(nil)
	ctx := context.Background()
	for week := int16(1); week <= 4; week++ {
		closeTheory(t, w, courseID, week, time.Now().Add(-time.Duration(30-week)*24*time.Hour))
	}
	approvedAt := time.Now().Add(-time.Hour)
	held := openTheory(t, w, courseID, 5)
	_, err := svc.ScanQR(ctx, s[0], scanRequest(w, held, time.Now()))
	require.NoError(t, err)
	_, err = svc.CloseSession(ctx, held, teacherID, false)
	require.NoError(t, err)
	w.myEnrollments = []db.GetStudentEnrollmentsBySemesterRow{{
		CourseID: utils.UUIDToPgtype(courseID), CourseCode: "FIZ101", EnrolledAt: utils.TimeToPgTimestamp(approvedAt),
	}}

	resp, err := svc.GetMyAttendance(ctx, s[0], semester)

	require.NoError(t, err)
	assert.Equal(t, &dto.SessionTypeAttendance{PresentCount: 1, AbsentCount: 0, TotalSessions: 1, MinRequired: 1, Passed: true}, resp.Courses[0].Theory)
	assert.Empty(t, resp.Courses[0].MissedSessions, "weeks 1-4 ran before the approval")
}

func TestFinalizeAttendance_ApprovedLate_NotFailedForEarlierSessions(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	svc := w.service(nil)
	ctx := context.Background()
	for week := int16(1); week <= 4; week++ {
		closeTheory(t, w, courseID, week, time.Now().Add(-time.Duration(30-week)*24*time.Hour), s[0], s[1])
	}
	// s[2] is approved now and attends the one session held since.
	w.enrolledAt = map[uuid.UUID]time.Time{s[2]: time.Now().Add(-time.Hour)}
	last := openTheory(t, w, courseID, 5)
	for _, student := range s {
		_, err := svc.ScanQR(ctx, student, scanRequest(w, last, time.Now()))
		require.NoError(t, err)
	}
	_, err := svc.CloseSession(ctx, last, teacherID, false)
	require.NoError(t, err)

	resp, err := svc.FinalizeAttendance(ctx, courseID, teacherID, semester)

	require.NoError(t, err)
	assert.Empty(t, resp.FailedStudents, "1 of the 1 session held since approval is a pass")
	assert.Equal(t, 4, resp.Thresholds.TheoryMinRequired, "the course-wide minimum still reads 4 of 5")
}

func jsonField(t *testing.T, raw []byte, field string) string {
	t.Helper()
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj))
	return string(obj[field])
}

// ---------------------------------------------------------- session date

func TestCreateSession_SessionDate_IsCampusLocalDay(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{"01:30 local belongs to the local day, not the UTC one", time.Date(2026, 10, 11, 22, 30, 0, 0, time.UTC), "2026-10-12"},
		{"afternoon matches the UTC day", time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), "2026-10-12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, courseID, _ := newAttWorld(t)
			clocktest.Freeze(t, tc.at)
			start, end := tc.at.Add(-24*time.Hour), tc.at.Add(30*24*time.Hour)
			w.periodStart, w.periodEnd = &start, &end

			resp, err := w.service(attSemesterClient{hardDeadline: tc.at.Add(60 * 24 * time.Hour)}).CreateSession(context.Background(), teacherID, dto.CreateSessionRequest{
				CourseID: courseID, WeekNumber: 1, SessionType: "theory", DurationMinutes: 15,
			})
			require.NoError(t, err)

			assert.Equal(t, tc.want, resp.SessionDate)
			stored := w.sessions[resp.SessionID].SessionDate.Time
			assert.Equal(t, tc.want, stored.Format("2006-01-02"), "pgx encodes the date from its calendar fields")
		})
	}
}
