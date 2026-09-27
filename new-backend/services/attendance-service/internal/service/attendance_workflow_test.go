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
	"github.com/baaaki/mydreamcampus/shared/client"
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

	periodStart, periodEnd *time.Time
	failSessionTotals      bool

	redisDown   bool
	enrolledSet map[string]map[string]bool
	scanned     map[string]map[string]bool
	buffer      map[string]map[string]string
	cache       map[string]map[string]any
	invalidated []uuid.UUID
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

func (v attCacheView) GetStudentEnrollmentsBySemester(context.Context, uuid.UUID, string) ([]db.GetStudentEnrollmentsBySemesterRow, error) {
	return nil, nil
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

func (v attSessionView) GetSessionsByCourse(context.Context, uuid.UUID, string) ([]db.GetSessionsByCourseRow, error) {
	return nil, nil
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

func (v attRecordView) GetMarkedStudentsBySession(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
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

func (v attRecordView) GetStudentAttendanceByCourse(context.Context, uuid.UUID, uuid.UUID, string) ([]db.GetStudentAttendanceByCourseRow, error) {
	return nil, nil
}

func (v attRecordView) GetStudentPresentCountByType(context.Context, uuid.UUID, uuid.UUID, string, db.SessionTypeEnum) (int64, error) {
	return 0, nil
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

func (v attRecordView) GetFailingStudentsByCourseByType(_ context.Context, courseID uuid.UUID, semester string, st db.SessionTypeEnum, total, min int64) ([]db.GetFailingStudentsByCourseByTypeRow, error) {
	var out []db.GetFailingStudentsByCourseByTypeRow
	for _, studentID := range v.w.enrolled[courseID] {
		var present int64
		for _, r := range v.w.records {
			if utils.PgUUIDToUUID(r.StudentID) == studentID && utils.PgUUIDToUUID(r.CourseID) == courseID && r.Semester == semester && r.SessionType == st {
				present++
			}
		}
		if present < min {
			s := v.w.students[studentID]
			out = append(out, db.GetFailingStudentsByCourseByTypeRow{
				StudentID: s.ID, StudentNumber: s.StudentNumber, FirstName: s.FirstName, LastName: s.LastName,
				Email: s.Email, PresentCount: present, AbsentCount: total - present,
			})
		}
	}
	return out, nil
}

type attOutboxView struct{ w *attWorld }

func (v attOutboxView) CreateOutboxEvent(_ context.Context, eventType, _ string, payload []byte) error {
	v.w.outbox = append(v.w.outbox, attOutboxEvent{eventType, payload})
	return nil
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
		enrolledSet: map[string]map[string]bool{}, scanned: map[string]map[string]bool{},
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
		outboxRepo: attOutboxView{w}, qrService: NewQRService(), redisService: attRedisView{w},
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

func TestScanQR_ClosedSession_NotFound(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	req := scanRequest(w, sessionID, time.Now())
	_, err := w.service(nil).CloseSession(context.Background(), sessionID, teacherID, false)
	require.NoError(t, err)

	_, err = w.service(nil).ScanQR(context.Background(), students[0], req)

	assert.ErrorIs(t, err, attErrors.ErrSessionNotFound)
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

func TestCloseSession_CountsBufferedScansAndKeepsTheBuffer(t *testing.T) {
	w, courseID, students := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	svc := w.service(nil)
	ctx := context.Background()
	_, err := svc.ScanQR(ctx, students[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	_, err = svc.CreateManualAttendance(ctx, sessionID, teacherID, false, dto.ManualAttendanceRequest{StudentID: students[1]})
	require.NoError(t, err)

	resp, err := svc.CloseSession(ctx, sessionID, teacherID, false)

	require.NoError(t, err)
	assert.Equal(t, 3, resp.Summary.TotalEnrolled)
	assert.Equal(t, 2, resp.Summary.PresentCount, "one in the database, one still in the buffer")
	assert.Equal(t, 1, resp.Summary.AbsentCount)
	assert.False(t, w.sessions[sessionID].IsActive.Bool)
	assert.Len(t, w.buffer[sessionID.String()], 1, "an unflushed scan must survive the close")
	assert.Nil(t, w.enrolledSet[sessionID.String()])

	_, err = svc.CloseSession(ctx, sessionID, teacherID, false)
	assert.ErrorIs(t, err, attErrors.ErrSessionNotActive)
	_, err = svc.CloseSession(ctx, openTheory(t, w, courseID, 2), uuid.New(), false)
	assert.ErrorIs(t, err, attErrors.ErrForbidden)
}

// holdSessions opens n sessions of a kind and marks each student present in
// the first present[i] of them.
func holdSessions(w *attWorld, courseID uuid.UUID, kind db.SessionTypeEnum, n int, students []uuid.UUID, present []int) {
	for week := 0; week < n; week++ {
		s, _ := attSessionView{w}.CreateAttendanceSession(context.Background(), db.CreateAttendanceSessionParams{
			CourseID: utils.UUIDToPgUUID(courseID), Semester: semester, WeekNumber: int16(week + 1), SessionType: kind,
		})
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
