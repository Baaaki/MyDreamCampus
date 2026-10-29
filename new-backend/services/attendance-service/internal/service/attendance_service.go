package service

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/attendance/internal/dto"
	"github.com/baaaki/mydreamcampus/attendance/internal/errors"
	"github.com/baaaki/mydreamcampus/attendance/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/rules"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	CacheTTLBuffer = 5 * time.Minute
	// The attendance rule is defined as a ratio over a standard 14-week
	// semester: at least 10/14 theory and 11/14 lab sessions. Actual
	// thresholds are scaled to the sessions really held — see
	// minTheoryRequired / minLabRequired in attendance_finalize_helpers.go.
	StandardSemesterWeeks = 14
	MinTheoryAttendance   = 10 // per 14 sessions
	MinLabAttendance      = 11 // per 14 sessions
)

// campusZone dates sessions by the campus's local day, as meal-service does:
// Turkey keeps UTC+3 all year, and on the UTC day a session opened before
// 03:00 would land on the day before.
var campusZone = time.FixedZone("UTC+3", 3*3600)

type AttendanceService struct {
	cacheRepo      attendanceCache
	sessionRepo    sessionStore
	attendanceRepo attendanceStore
	finalizations  finalizationStore
	qrService      *QRService
	redisService   attendanceRedis
	semesterClient SemesterClient
	periodRepo     attendancePeriods
}

func NewAttendanceService(
	cacheRepo *repository.CacheRepository,
	sessionRepo *repository.SessionRepository,
	attendanceRepo *repository.AttendanceRepository,
	finalizations *repository.FinalizationRepository,
	qrService *QRService,
	redisService *RedisService,
	semesterClient SemesterClient,
	periodRepo *sharedRepo.SimplePeriodRepository,
) *AttendanceService {
	return &AttendanceService{
		cacheRepo:      cacheRepo,
		sessionRepo:    sessionRepo,
		attendanceRepo: attendanceRepo,
		finalizations:  finalizations,
		qrService:      qrService,
		redisService:   redisService,
		semesterClient: semesterClient,
		periodRepo:     periodRepo,
	}
}

// CreateSession creates a new attendance session
func (s *AttendanceService) CreateSession(ctx context.Context, instructorID uuid.UUID, req dto.CreateSessionRequest) (dto.CreateSessionResponse, error) {
	logger.Info("Creating attendance session",
		zap.String("course_id", req.CourseID.String()),
		zap.Int("week", int(req.WeekNumber)),
	)

	// 1. Check instructor owns the course
	course, err := s.cacheRepo.GetCourseCacheByID(ctx, req.CourseID)
	if err != nil {
		logger.Error("course not found", zap.Error(err))
		return dto.CreateSessionResponse{}, errors.ErrCourseNotFound
	}

	if utils.PgUUIDToUUID(course.InstructorID) != instructorID {
		logger.Error("instructor does not own course")
		return dto.CreateSessionResponse{}, errors.ErrForbidden
	}

	// A session opened after finalization would count for nothing: grades
	// already has the outcome.
	if _, finalized, err := s.finalizations.GetCourseFinalization(ctx, req.CourseID); err != nil {
		return dto.CreateSessionResponse{}, fmt.Errorf("read course finalization: %w", err)
	} else if finalized {
		return dto.CreateSessionResponse{}, errors.ErrAttendanceFinalized
	}

	// 2. Semester enforcement: checks hard_deadline + period window
	if err := s.checkSemesterEnforcement(ctx, course.Semester, false); err != nil {
		return dto.CreateSessionResponse{}, err
	}

	// 3. Validate lab session is allowed
	if req.SessionType == "lab" && !course.HasLab {
		logger.Error("lab session requested but course has no lab hours")
		return dto.CreateSessionResponse{}, errors.ErrLabNotAvailable
	}

	// 3. Check session doesn't already exist for this week + session_type
	sessionType := db.SessionTypeEnum(req.SessionType)
	exists, err := s.sessionRepo.CheckSessionExists(ctx, req.CourseID, req.WeekNumber, sessionType)
	if err != nil {
		return dto.CreateSessionResponse{}, err
	}
	if exists {
		return dto.CreateSessionResponse{}, errors.ErrSessionAlreadyExists
	}

	// 3. Generate QR secret
	qrSecret, err := s.qrService.GenerateSecret()
	if err != nil {
		return dto.CreateSessionResponse{}, err
	}

	// 4. Create session in DB
	now := clock.Now()
	expiresAt := now.Add(time.Duration(req.DurationMinutes) * time.Minute)
	sessionDay := now.In(campusZone)

	session, err := s.sessionRepo.CreateAttendanceSession(ctx, db.CreateAttendanceSessionParams{
		CourseID:     utils.UUIDToPgUUID(req.CourseID),
		InstructorID: utils.UUIDToPgUUID(instructorID),
		Semester:     course.Semester,
		WeekNumber:   req.WeekNumber,
		SessionDate:  utils.TimeToPgDate(sessionDay),
		QrSecret:     qrSecret,
		// Stored per session so a later change to the default cannot
		// invalidate codes already on screen.
		QrRotationInterval: pgtype.Int2{Int16: int16(DefaultQRRotation / time.Second), Valid: true},
		StartedAt:          utils.TimeToPgTimestamp(now),
		ExpiresAt:          utils.TimeToPgTimestamp(expiresAt),
		SessionType:        sessionType,
	})
	if err != nil {
		return dto.CreateSessionResponse{}, err
	}

	// 5. Warm Redis cache
	sessionID := utils.PgUUIDToUUID(session.ID).String()

	// Get enrolled students
	enrolledStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, req.CourseID, course.Semester)
	if err != nil {
		logger.Error("failed to get enrolled students", zap.Error(err))
	} else {
		studentIDs := make([]uuid.UUID, len(enrolledStudents))
		for i, student := range enrolledStudents {
			studentIDs[i] = utils.PgUUIDToUUID(student.ID)
		}
		if err := s.redisService.AddEnrolledStudents(ctx, sessionID, studentIDs, expiresAt.Sub(now)+CacheTTLBuffer); err != nil {
			// Best-effort warm: scan path falls back to DB when the set is missing.
			logger.Warn("failed to warm enrolled-student set", zap.Error(err))
		}
	}

	// Session cache
	if err := s.redisService.SetSessionCache(ctx, sessionID, map[string]any{
		"course_id":      req.CourseID.String(),
		"instructor_id":  instructorID.String(),
		"semester":       course.Semester,
		"week_number":    fmt.Sprintf("%d", req.WeekNumber),
		"session_type":   req.SessionType,
		"qr_secret":      qrSecret,
		"expires_at":     fmt.Sprintf("%d", expiresAt.Unix()),
		"enrolled_count": fmt.Sprintf("%d", len(enrolledStudents)),
	}, expiresAt.Sub(now)+CacheTTLBuffer); err != nil {
		logger.Warn("failed to warm session cache", zap.Error(err))
	}

	return dto.CreateSessionResponse{
		SessionID:            utils.PgUUIDToUUID(session.ID),
		CourseID:             req.CourseID,
		CourseCode:           course.CourseCode,
		CourseName:           course.CourseName,
		WeekNumber:           req.WeekNumber,
		SessionType:          req.SessionType,
		SessionDate:          sessionDay.Format("2006-01-02"),
		StartedAt:            now,
		ExpiresAt:            expiresAt,
		EnrolledStudentCount: len(enrolledStudents),
	}, nil
}

// ScanQR processes QR code scan for attendance
// Simplified: no student active check (JWT handles it), no Redis marked set check (DB UNIQUE handles duplicates)
func (s *AttendanceService) ScanQR(ctx context.Context, studentID uuid.UUID, req dto.ScanQRRequest) (dto.ScanQRResponse, error) {
	logger.Info("Processing QR scan", zap.String("student_id", studentID.String()))

	// 1. Get session (with fallback)
	sessionID := req.QRPayload.SessionID
	session, err := s.getSessionWithFallback(ctx, sessionID)
	if err != nil {
		return dto.ScanQRResponse{}, errors.ErrSessionNotFound
	}

	// Check if expired
	if clock.Now().After(utils.PgTimestampToTime(session.ExpiresAt)) {
		return dto.ScanQRResponse{}, errors.ErrSessionExpired
	}
	// Closed by the instructor before it ran out.
	if !session.IsActive.Bool {
		return dto.ScanQRResponse{}, errors.ErrSessionNotActive
	}

	// Semester enforcement: checks hard_deadline + period window
	if err := s.checkSemesterEnforcement(ctx, session.Semester, false); err != nil {
		return dto.ScanQRResponse{}, err
	}

	// 2. Validate QR signature — also rejects a code whose window has passed
	if !s.qrService.ValidateQRSignature(req.QRPayload, session.QrSecret, clock.Now(), qrRotation(session)) {
		return dto.ScanQRResponse{}, errors.ErrInvalidQRCode
	}

	// 3. Check enrollment (with fallback)
	courseID := utils.PgUUIDToUUID(session.CourseID)
	enrolled, err := s.checkEnrollmentWithFallback(ctx, sessionID, studentID, courseID, session.Semester)
	if err != nil || !enrolled {
		return dto.ScanQRResponse{}, errors.ErrNotEnrolled
	}

	// 5. Build record params. Dedup is handled atomically by AddToBuffer via SADD.
	// DB UNIQUE(session_id, student_id) + ON CONFLICT DO NOTHING covers the
	// fallback path when Redis is unavailable.
	recordParams := db.CreateAttendanceRecordQRParams{
		SessionID:   session.ID,
		StudentID:   utils.UUIDToPgUUID(studentID),
		CourseID:    session.CourseID,
		Semester:    session.Semester,
		WeekNumber:  session.WeekNumber,
		ScannedAt:   utils.TimeToPgTimestamp(clock.Now()),
		SessionType: session.SessionType,
	}

	bufferJSON, err := json.Marshal(recordParams)
	if err != nil {
		return dto.ScanQRResponse{}, fmt.Errorf("failed to marshal buffer data: %w", err)
	}

	// Scanned SET TTL = session remaining time + 1 hour safety buffer.
	// Remaining on the service clock, the one expires_at was set on; against
	// the real clock a simulated session would get a TTL off by the offset.
	scannedTTL := utils.PgTimestampToTime(session.ExpiresAt).Sub(clock.Now()) + 1*time.Hour

	added, err := s.redisService.AddToBuffer(ctx, sessionID, studentID.String(), string(bufferJSON), scannedTTL)
	if err != nil {
		// Redis down, write directly to DB. ON CONFLICT keeps this safe on
		// duplicates; 0 rows means the scan was already recorded.
		logger.Warn("Redis down, writing directly to DB", zap.Error(err))
		inserted, dbErr := s.attendanceRepo.CreateAttendanceRecordQR(ctx, recordParams)
		if dbErr != nil {
			return dto.ScanQRResponse{}, dbErr
		}
		if !inserted {
			return dto.ScanQRResponse{}, errors.ErrAlreadyMarked
		}
	} else if !added {
		return dto.ScanQRResponse{}, errors.ErrAlreadyMarked
	}

	// Get course info for response
	course, err := s.cacheRepo.GetCourseCacheByID(ctx, courseID)
	if err != nil {
		logger.Error("failed to get course info for scan response", zap.Error(err))
		return dto.ScanQRResponse{}, errors.ErrCourseNotFound
	}

	return dto.ScanQRResponse{
		Message:     "Yoklama başarıyla alındı",
		CourseCode:  course.CourseCode,
		CourseName:  course.CourseName,
		WeekNumber:  session.WeekNumber,
		SessionType: string(session.SessionType),
		MarkedAt:    clock.Now(),
	}, nil
}

// GetQRCode generates QR code data for instructor
func (s *AttendanceService) GetQRCode(ctx context.Context, sessionID, instructorID uuid.UUID) (dto.GetQRResponse, error) {
	session, err := s.sessionRepo.GetActiveSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetQRResponse{}, errors.ErrSessionNotFound
	}

	// Check ownership
	if utils.PgUUIDToUUID(session.InstructorID) != instructorID {
		return dto.GetQRResponse{}, errors.ErrForbidden
	}

	rotation := qrRotation(session)
	payload := s.qrService.GenerateQRPayload(sessionID.String(), session.QrSecret, clock.Now(), rotation)

	return dto.GetQRResponse{
		SessionID:        sessionID,
		QRPayload:        payload,
		ValidUntil:       utils.PgTimestampToTime(session.ExpiresAt),
		RotationInterval: int(rotation / time.Second),
	}, nil
}

// canManageSession: the instructor who opened the session, or an admin —
// the same bypass grades gives admin on course reads. QR display, session
// creation and finalize stay instructor-only.
func canManageSession(owner pgtype.UUID, callerID uuid.UUID, isAdmin bool) bool {
	return isAdmin || utils.PgUUIDToUUID(owner) == callerID
}

// qrRotation reads the session's rotation interval. Sessions created before
// it was stored have NULL there and get the default.
func qrRotation(session db.AttendanceSession) time.Duration {
	if session.QrRotationInterval.Valid && session.QrRotationInterval.Int16 > 0 {
		return time.Duration(session.QrRotationInterval.Int16) * time.Second
	}
	return DefaultQRRotation
}

// CreateManualAttendance creates manual attendance record
func (s *AttendanceService) CreateManualAttendance(ctx context.Context, sessionID, instructorID uuid.UUID, isAdmin bool, req dto.ManualAttendanceRequest) (dto.ManualAttendanceResponse, error) {
	session, err := s.sessionRepo.GetActiveSessionByID(ctx, sessionID)
	if err != nil {
		return dto.ManualAttendanceResponse{}, errors.ErrSessionNotFound
	}

	if !canManageSession(session.InstructorID, instructorID, isAdmin) {
		return dto.ManualAttendanceResponse{}, errors.ErrForbidden
	}

	// Semester enforcement: checks hard_deadline + period window
	if err := s.checkSemesterEnforcement(ctx, session.Semester, false); err != nil {
		return dto.ManualAttendanceResponse{}, err
	}

	// Check enrollment
	enrolled, err := s.cacheRepo.CheckEnrollment(ctx, req.StudentID, utils.PgUUIDToUUID(session.CourseID), session.Semester)
	if err != nil {
		logger.Error("failed to check enrollment", zap.Error(err))
		return dto.ManualAttendanceResponse{}, errors.ErrNotEnrolled
	}
	if !enrolled {
		return dto.ManualAttendanceResponse{}, errors.ErrNotEnrolled
	}

	// Create record (ON CONFLICT updates existing record)
	record, err := s.attendanceRepo.CreateAttendanceRecordManual(ctx, db.CreateAttendanceRecordManualParams{
		SessionID:        utils.UUIDToPgUUID(sessionID),
		StudentID:        utils.UUIDToPgUUID(req.StudentID),
		CourseID:         session.CourseID,
		Semester:         session.Semester,
		WeekNumber:       session.WeekNumber,
		ManuallyMarkedBy: utils.UUIDToPgUUID(instructorID),
		ManualNote:       utils.StringToPgText(req.Note),
		SessionType:      session.SessionType,
	})
	if err != nil {
		return dto.ManualAttendanceResponse{}, err
	}

	// Invalidate student summary cache
	if err := s.redisService.InvalidateStudentSummary(ctx, req.StudentID, session.Semester); err != nil {
		logger.Warn("failed to invalidate student summary cache", zap.Error(err))
	}

	// Get student info
	student, err := s.cacheRepo.GetStudentCacheByID(ctx, req.StudentID)
	if err != nil {
		logger.Error("failed to get student info", zap.Error(err))
		return dto.ManualAttendanceResponse{}, errors.ErrStudentNotFound
	}

	return dto.ManualAttendanceResponse{
		ID:            utils.PgUUIDToUUID(record.ID),
		SessionID:     sessionID,
		StudentID:     req.StudentID,
		StudentNumber: student.StudentNumber,
		StudentName:   fmt.Sprintf("%s %s", utils.PgTextToString(student.FirstName), utils.PgTextToString(student.LastName)),
		MarkedVia:     "manual",
		Note:          &req.Note,
		MarkedAt: func() *time.Time {
			t := utils.PgTimestampToTime(record.ManuallyMarkedAt)
			if t.IsZero() {
				return nil
			}
			return &t
		}(),
	}, nil
}

// CloseSession closes an attendance session
// Simplified: no absent record creation needed. Record exists = present, no record = absent.
func (s *AttendanceService) CloseSession(ctx context.Context, sessionID, instructorID uuid.UUID, isAdmin bool) (dto.CloseSessionResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.CloseSessionResponse{}, errors.ErrSessionNotFound
	}

	if !canManageSession(session.InstructorID, instructorID, isAdmin) {
		return dto.CloseSessionResponse{}, errors.ErrForbidden
	}

	// Check if already closed
	if !utils.PgBoolToBool(session.IsActive) {
		return dto.CloseSessionResponse{}, errors.ErrSessionNotActive
	}

	// Deactivate session
	if err := s.sessionRepo.DeactivateSession(ctx, sessionID); err != nil {
		return dto.CloseSessionResponse{}, err
	}

	// Write the buffered scans now instead of on the flusher's next tick: a
	// finalize right after the last close must find them in the database.
	// A failure leaves them in the buffer for the flusher to retry, so the
	// close itself still succeeds.
	if err := FlushSessionBuffer(ctx, s.attendanceRepo, s.redisService, sessionID.String()); err != nil {
		logger.Warn("failed to flush scan buffer at close, flusher will retry",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}

	// Clear Redis keys
	if err := s.redisService.ClearSessionKeys(ctx, sessionID.String()); err != nil {
		logger.Warn("failed to clear session redis keys", zap.Error(err))
	}

	// Get counts
	courseID := utils.PgUUIDToUUID(session.CourseID)
	enrolledStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, courseID, session.Semester)
	if err != nil {
		logger.Error("failed to get enrolled students", zap.Error(err))
		return dto.CloseSessionResponse{}, err
	}

	marked, err := s.attendanceRepo.GetMarkedStudentsBySession(ctx, sessionID)
	if err != nil {
		logger.Error("failed to get marked students", zap.Error(err))
		return dto.CloseSessionResponse{}, err
	}
	present := make(map[string]bool, len(marked))
	for _, id := range marked {
		present[id.String()] = true
	}

	// Scans the flush above could not write are still in the buffer; count
	// them so the summary the instructor sees at close time is complete. A
	// set rather than a sum: a scan already in the database (its HDEL
	// failed) or a student also marked by hand must count once.
	if pending, bufErr := s.redisService.GetBuffer(ctx, sessionID.String()); bufErr == nil {
		for studentID := range pending {
			present[studentID] = true
		}
	}
	presentCount := len(present)

	totalEnrolled := len(enrolledStudents)

	return dto.CloseSessionResponse{
		SessionID: sessionID,
		ClosedAt:  clock.Now(),
		Summary: dto.SessionSummary{
			TotalEnrolled: totalEnrolled,
			PresentCount:  presentCount,
			AbsentCount:   totalEnrolled - presentCount,
		},
	}, nil
}

// GetMyAttendance returns student's own attendance records
func (s *AttendanceService) GetMyAttendance(ctx context.Context, studentID uuid.UUID, semester string) (dto.GetMyAttendanceResponse, error) {
	// Check student exists
	student, err := s.cacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		return dto.GetMyAttendanceResponse{}, errors.ErrStudentNotFound
	}

	if semester == "" {
		periods, err := s.periodRepo.GetAllPeriods(ctx)
		if err != nil {
			return dto.GetMyAttendanceResponse{}, fmt.Errorf("list academic periods: %w", err)
		}
		semester = currentSemester(periods, clock.Now())
	}

	// Get enrollments
	enrollments, err := s.cacheRepo.GetStudentEnrollmentsBySemester(ctx, studentID, semester)
	if err != nil {
		return dto.GetMyAttendanceResponse{}, err
	}

	courses := make([]dto.CourseAttendanceDetail, 0, len(enrollments))

	for _, enrollment := range enrollments {
		courseID := utils.PgUUIDToUUID(enrollment.CourseID)

		// Get attendance records (only records where student was present)
		records, err := s.attendanceRepo.GetStudentAttendanceByCourse(ctx, studentID, courseID, semester)
		if err != nil {
			continue
		}

		sessions, err := s.sessionRepo.GetSessionsByCourse(ctx, courseID, semester)
		if err != nil {
			return dto.GetMyAttendanceResponse{}, fmt.Errorf("list sessions of course %s: %w", courseID, err)
		}

		weeklyRecords := make([]dto.WeeklyAttendanceRecord, 0, len(records))
		for _, record := range records {
			weeklyRecords = append(weeklyRecords, dto.WeeklyAttendanceRecord{
				Week:        record.WeekNumber,
				SessionType: string(record.SessionType),
				Date:        record.SessionDate.Time.Format("2006-01-02"),
				MarkedVia:   record.MarkedVia,
			})
		}

		theoryTotal, theoryPresent := heldAndAttended(sessions, records, db.AttendanceSessionTypeEnumTheory)
		labTotal, labPresent := heldAndAttended(sessions, records, db.AttendanceSessionTypeEnumLab)

		detail := dto.CourseAttendanceDetail{
			CourseID:       courseID,
			CourseCode:     enrollment.CourseCode,
			CourseName:     enrollment.CourseName,
			Instructor:     utils.PgTextToString(enrollment.InstructorFullname),
			TotalWeeks:     enrollment.TotalWeeks.Int16,
			WeeklyRecords:  weeklyRecords,
			MissedSessions: missedSessions(sessions, records),
		}

		if theoryTotal > 0 {
			minTheory := minTheoryRequired(theoryTotal)
			detail.Theory = &dto.SessionTypeAttendance{
				PresentCount:  theoryPresent,
				AbsentCount:   int(theoryTotal) - theoryPresent,
				TotalSessions: int(theoryTotal),
				MinRequired:   minTheory,
				Passed:        theoryPresent >= minTheory,
			}
		}

		if labTotal > 0 {
			minLab := minLabRequired(labTotal)
			detail.Lab = &dto.SessionTypeAttendance{
				PresentCount:  labPresent,
				AbsentCount:   int(labTotal) - labPresent,
				TotalSessions: int(labTotal),
				MinRequired:   minLab,
				Passed:        labPresent >= minLab,
			}
		}

		courses = append(courses, detail)
	}

	return dto.GetMyAttendanceResponse{
		StudentID:     studentID,
		StudentNumber: student.StudentNumber,
		Semester:      semester,
		Courses:       courses,
	}, nil
}

// missedSessions lists the sessions a course held without the student on
// the roll. A session still open is left out: its QR scans wait in Redis
// until it closes, so the student's record may just not be written yet.
func missedSessions(sessions []db.GetSessionsByCourseRow, attended []db.GetStudentAttendanceByCourseRow) []dto.MissedSession {
	type slot struct {
		week        int16
		sessionType db.SessionTypeEnum
	}
	present := make(map[slot]bool, len(attended))
	for _, r := range attended {
		present[slot{r.WeekNumber, r.SessionType}] = true
	}

	missed := make([]dto.MissedSession, 0, len(sessions))
	for _, session := range sessions {
		if session.IsActive.Bool || present[slot{session.WeekNumber, session.SessionType}] {
			continue
		}
		missed = append(missed, dto.MissedSession{
			Week:        session.WeekNumber,
			SessionType: string(session.SessionType),
			Date:        session.SessionDate.Time.Format("2006-01-02"),
		})
	}
	return missed
}

// heldAndAttended counts a course's sessions of one type and the student's
// records among them. An open session the student has no record for is left
// out, as missedSessions leaves it out: a scan waits in Redis for the flusher,
// and counting the session would show a student who has just scanned one
// absence short — below the minimum, on a close call.
func heldAndAttended(sessions []db.GetSessionsByCourseRow, attended []db.GetStudentAttendanceByCourseRow, sessionType db.SessionTypeEnum) (held int64, present int) {
	attendedWeeks := make(map[int16]bool, len(attended))
	for _, r := range attended {
		if r.SessionType == sessionType {
			attendedWeeks[r.WeekNumber] = true
		}
	}
	for _, session := range sessions {
		if session.SessionType != sessionType {
			continue
		}
		switch {
		case attendedWeeks[session.WeekNumber]:
			held++
			present++
		case !session.IsActive.Bool:
			held++
		}
	}
	return held, present
}

// currentSemester picks the semester whose attendance period holds now, on
// the service clock so the time machine moves it too. After a period closes
// students still look back at it, so between periods the latest one that has
// started stands in; before the first one there is none.
func currentSemester(periods []sharedRepo.SimplePeriod, now time.Time) string {
	var current, latest *sharedRepo.SimplePeriod
	for i := range periods {
		p := &periods[i]
		if !p.IsActive || p.PeriodStart.After(now) {
			continue
		}
		if !now.After(p.PeriodEnd) && (current == nil || p.PeriodStart.After(current.PeriodStart)) {
			current = p
		}
		if latest == nil || p.PeriodStart.After(latest.PeriodStart) {
			latest = p
		}
	}
	switch {
	case current != nil:
		return current.Semester
	case latest != nil:
		return latest.Semester
	default:
		return ""
	}
}

// FinalizeAttendance closes a course's attendance once. Students below the
// theory (10/14) or lab (11/14) threshold fail, and the record of the
// finalization is written with an attendance.semester.failed event for each
// of them, so a second call is refused instead of queuing the events again.
// semester may be empty: a course id belongs to one semester and that one
// is used.
func (s *AttendanceService) FinalizeAttendance(ctx context.Context, courseID, instructorID uuid.UUID, semester string) (dto.FinalizeAttendanceResponse, error) {
	logger.Info("Finalizing attendance",
		zap.String("course_id", courseID.String()),
		zap.String("semester", semester),
	)

	// Check ownership
	course, err := s.cacheRepo.GetCourseCacheByID(ctx, courseID)
	if err != nil {
		return dto.FinalizeAttendanceResponse{}, errors.ErrCourseNotFound
	}
	if utils.PgUUIDToUUID(course.InstructorID) != instructorID {
		return dto.FinalizeAttendanceResponse{}, errors.ErrForbidden
	}
	// Under any other semester name the course has no sessions, every
	// threshold is zero and the whole class would pass.
	if semester != "" && semester != course.Semester {
		return dto.FinalizeAttendanceResponse{}, errors.ErrSemesterMismatch
	}
	semester = course.Semester

	if _, finalized, err := s.finalizations.GetCourseFinalization(ctx, courseID); err != nil {
		return dto.FinalizeAttendanceResponse{}, fmt.Errorf("read course finalization: %w", err)
	} else if finalized {
		return dto.FinalizeAttendanceResponse{}, errors.ErrAttendanceFinalized
	}

	// An open session's scans wait in Redis until it closes; counting now
	// would miss them and fail students who were there.
	sessions, err := s.sessionRepo.GetSessionsByCourse(ctx, courseID, semester)
	if err != nil {
		return dto.FinalizeAttendanceResponse{}, fmt.Errorf("list course sessions: %w", err)
	}
	for _, session := range sessions {
		if session.IsActive.Bool {
			return dto.FinalizeAttendanceResponse{}, errors.ErrOpenSessionExists
		}
	}
	if err := s.drainScanBuffers(ctx, sessions); err != nil {
		return dto.FinalizeAttendanceResponse{}, err
	}

	// Get total sessions by type. Thresholds scale with the sessions
	// actually held (10/14 and 11/14 ratios) instead of flat constants.
	// A total read as zero would lift every threshold and pass the whole
	// class, so a failed read stops the finalize instead.
	theoryTotal, err := s.attendanceRepo.GetTotalSessionsByCourseAndType(ctx, courseID, semester, db.AttendanceSessionTypeEnumTheory)
	if err != nil {
		return dto.FinalizeAttendanceResponse{}, fmt.Errorf("count theory sessions: %w", err)
	}
	labTotal, err := s.attendanceRepo.GetTotalSessionsByCourseAndType(ctx, courseID, semester, db.AttendanceSessionTypeEnumLab)
	if err != nil {
		return dto.FinalizeAttendanceResponse{}, fmt.Errorf("count lab sessions: %w", err)
	}
	minTheory := minTheoryRequired(theoryTotal)
	minLab := minLabRequired(labTotal)

	// Get failing students for theory (present_count < minTheory)
	var theoryFailing []db.GetFailingStudentsByCourseByTypeRow
	if theoryTotal > 0 {
		theoryFailing, err = s.attendanceRepo.GetFailingStudentsByCourseByType(ctx, courseID, semester, db.AttendanceSessionTypeEnumTheory, theoryTotal, int64(minTheory))
		if err != nil {
			return dto.FinalizeAttendanceResponse{}, err
		}
	}

	// Get failing students for lab (present_count < minLab)
	var labFailing []db.GetFailingStudentsByCourseByTypeRow
	if labTotal > 0 {
		labFailing, err = s.attendanceRepo.GetFailingStudentsByCourseByType(ctx, courseID, semester, db.AttendanceSessionTypeEnumLab, labTotal, int64(minLab))
		if err != nil {
			return dto.FinalizeAttendanceResponse{}, err
		}
	}

	failMap := mergeFailureRows(theoryFailing, labFailing)

	// Get all students for total count
	allStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, courseID, semester)
	if err != nil {
		logger.Error("failed to get enrolled students", zap.Error(err))
		return dto.FinalizeAttendanceResponse{}, err
	}

	failed := []dto.FailedStudent{}
	var failedEvents []repository.OutboxEvent

	for studentID, info := range failMap {
		failedType := deriveFailedType(info.TheoryFailed, info.LabFailed)

		failedStudent := dto.FailedStudent{
			StudentID:     studentID,
			StudentNumber: info.StudentNumber,
			StudentName:   info.StudentName,
			FailedType:    failedType,
		}

		if theoryTotal > 0 {
			failedStudent.Theory = &dto.SessionTypeAttendance{
				PresentCount:  info.TheoryPresent,
				AbsentCount:   info.TheoryAbsent,
				TotalSessions: int(theoryTotal),
				MinRequired:   minTheory,
				Passed:        !info.TheoryFailed,
			}
		}

		if labTotal > 0 {
			failedStudent.Lab = &dto.SessionTypeAttendance{
				PresentCount:  info.LabPresent,
				AbsentCount:   info.LabAbsent,
				TotalSessions: int(labTotal),
				MinRequired:   minLab,
				Passed:        !info.LabFailed,
			}
		}

		failed = append(failed, failedStudent)

		eventData := dto.AttendanceSemesterFailedEventData{
			StudentID:     studentID,
			StudentNumber: info.StudentNumber,
			StudentEmail:  info.StudentEmail,
			CourseID:      courseID,
			CourseCode:    course.CourseCode,
			CourseName:    course.CourseName,
			Semester:      semester,
			TotalWeeks:    course.TotalWeeks.Int16,
			FailedType:    failedType,
		}

		if theoryTotal > 0 {
			eventData.Theory = &dto.AttendanceFailedTypeDetail{
				TotalSessions: int(theoryTotal),
				PresentCount:  info.TheoryPresent,
				AbsentCount:   info.TheoryAbsent,
				MinRequired:   minTheory,
			}
		}

		if labTotal > 0 {
			eventData.Lab = &dto.AttendanceFailedTypeDetail{
				TotalSessions: int(labTotal),
				PresentCount:  info.LabPresent,
				AbsentCount:   info.LabAbsent,
				MinRequired:   minLab,
			}
		}

		payload, err := failedAttendancePayload(eventData)
		if err != nil {
			return dto.FinalizeAttendanceResponse{}, err
		}
		failedEvents = append(failedEvents, repository.OutboxEvent{
			EventType:  events.EventAttendanceSemesterFailed,
			RoutingKey: events.EventAttendanceSemesterFailed,
			Payload:    payload,
		})
	}
	sort.Slice(failed, func(i, j int) bool { return failed[i].StudentNumber < failed[j].StudentNumber })

	resp := dto.FinalizeAttendanceResponse{
		CourseID:      courseID,
		CourseCode:    course.CourseCode,
		Semester:      semester,
		TotalStudents: len(allStudents),
		TotalWeeks:    course.TotalWeeks.Int16,
		Thresholds: struct {
			TheoryMinRequired int `json:"theory_min_required"`
			LabMinRequired    int `json:"lab_min_required"`
		}{
			TheoryMinRequired: minTheory,
			LabMinRequired:    minLab,
		},
		FinalizationSummary: struct {
			PassingCount int `json:"passing_count"`
			FailingCount int `json:"failing_count"`
		}{
			PassingCount: len(allStudents) - len(failMap),
			FailingCount: len(failMap),
		},
		FailedStudents:  failed,
		EventsPublished: len(failedEvents),
		FinalizedAt:     clock.Now(),
	}

	result, err := json.Marshal(resp)
	if err != nil {
		return dto.FinalizeAttendanceResponse{}, fmt.Errorf("encode finalization result: %w", err)
	}
	saved, err := s.finalizations.SaveCourseFinalization(ctx, db.InsertCourseFinalizationParams{
		CourseID:    utils.UUIDToPgUUID(courseID),
		Semester:    semester,
		FinalizedBy: utils.UUIDToPgUUID(instructorID),
		FinalizedAt: pgtype.Timestamptz{Time: resp.FinalizedAt, Valid: true},
		Result:      result,
	}, failedEvents)
	if err != nil {
		return dto.FinalizeAttendanceResponse{}, fmt.Errorf("save course finalization: %w", err)
	}
	if !saved {
		// A concurrent call finalized the course between the check and here.
		return dto.FinalizeAttendanceResponse{}, errors.ErrAttendanceFinalized
	}
	return resp, nil
}

// drainScanBuffers makes sure no closed session of the course still has
// scans in Redis. Closing a session flushes its buffer, but a flush that
// failed there is left to the BufferFlusher; finalizing before it catches up
// would mark those students absent, and a finalization cannot be undone.
// Anything still buffered after one more flush refuses the finalize.
func (s *AttendanceService) drainScanBuffers(ctx context.Context, sessions []db.GetSessionsByCourseRow) error {
	for _, session := range sessions {
		sessionID := utils.PgUUIDToUUID(session.ID).String()
		if err := FlushSessionBuffer(ctx, s.attendanceRepo, s.redisService, sessionID); err != nil {
			logger.Warn("failed to flush scan buffer before finalization",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}
		pending, err := s.redisService.GetBuffer(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("read scan buffer of session %s: %w", sessionID, err)
		}
		if len(pending) > 0 {
			return errors.ErrScansPending
		}
	}
	return nil
}

// GetFinalization tells the course's teacher, or an admin, whether its
// attendance was finalized and, if so, the summary saved with it.
func (s *AttendanceService) GetFinalization(ctx context.Context, courseID, callerID uuid.UUID, isAdmin bool) (dto.FinalizationStatusResponse, error) {
	course, err := s.cacheRepo.GetCourseCacheByID(ctx, courseID)
	if err != nil {
		return dto.FinalizationStatusResponse{}, errors.ErrCourseNotFound
	}
	if !canManageSession(course.InstructorID, callerID, isAdmin) {
		return dto.FinalizationStatusResponse{}, errors.ErrForbidden
	}

	row, finalized, err := s.finalizations.GetCourseFinalization(ctx, courseID)
	if err != nil {
		return dto.FinalizationStatusResponse{}, fmt.Errorf("read course finalization: %w", err)
	}
	if !finalized {
		return dto.FinalizationStatusResponse{Finalized: false}, nil
	}
	var result dto.FinalizeAttendanceResponse
	if err := json.Unmarshal(row.Result, &result); err != nil {
		return dto.FinalizationStatusResponse{}, fmt.Errorf("decode course finalization: %w", err)
	}
	return dto.FinalizationStatusResponse{Finalized: true, Result: &result}, nil
}

// GetCourseSessions returns all sessions for a course
func (s *AttendanceService) GetCourseSessions(ctx context.Context, courseID, instructorID uuid.UUID) (dto.GetCourseSessionsResponse, error) {
	// Check ownership
	course, err := s.cacheRepo.GetCourseCacheByID(ctx, courseID)
	if err != nil {
		return dto.GetCourseSessionsResponse{}, errors.ErrCourseNotFound
	}
	if utils.PgUUIDToUUID(course.InstructorID) != instructorID {
		return dto.GetCourseSessionsResponse{}, errors.ErrForbidden
	}

	// Get enrolled students count for absent calculation
	enrolledStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, courseID, course.Semester)
	if err != nil {
		return dto.GetCourseSessionsResponse{}, err
	}
	totalEnrolled := len(enrolledStudents)

	// Get sessions
	sessions, err := s.sessionRepo.GetSessionsByCourse(ctx, courseID, course.Semester)
	if err != nil {
		return dto.GetCourseSessionsResponse{}, err
	}

	var sessionList []dto.SessionListItem
	completedSessions := 0

	for _, session := range sessions {
		sessionID := utils.PgUUIDToUUID(session.ID)
		isActive := utils.PgBoolToBool(session.IsActive)
		sessionDate := session.SessionDate.Time.Format("2006-01-02")

		// Get present count
		presentCount64, err := s.attendanceRepo.GetSessionAttendanceCount(ctx, sessionID)
		if err != nil {
			logger.Error("failed to get session attendance count", zap.String("session_id", sessionID.String()), zap.Error(err))
			continue
		}

		// Status
		status := "expired"
		if isActive {
			status = "active"
		}

		presentCount := int(presentCount64)
		absentCount := totalEnrolled - presentCount

		if !isActive {
			completedSessions++
		}

		sessionList = append(sessionList, dto.SessionListItem{
			SessionID:    &sessionID,
			WeekNumber:   session.WeekNumber,
			SessionType:  string(session.SessionType),
			SessionDate:  &sessionDate,
			PresentCount: &presentCount,
			AbsentCount:  &absentCount,
			IsActive:     &isActive,
			Status:       &status,
		})
	}

	return dto.GetCourseSessionsResponse{
		CourseID:   courseID,
		CourseCode: course.CourseCode,
		CourseName: course.CourseName,
		Semester:   course.Semester,
		TotalWeeks: course.TotalWeeks.Int16,
		Sessions:   sessionList,
		OverallStats: struct {
			CompletedSessions int `json:"completed_sessions"`
		}{
			CompletedSessions: completedSessions,
		},
	}, nil
}

// Helper functions

func (s *AttendanceService) getSessionWithFallback(ctx context.Context, sessionID string) (db.AttendanceSession, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return db.AttendanceSession{}, fmt.Errorf("invalid session ID: %w", err)
	}
	// DB is source of truth; session cache is cleared on close and not rehydrated here.
	// Read open or not: a student who scans late is told the session ended,
	// not that it does not exist.
	return s.sessionRepo.GetSessionByID(ctx, sessionUUID)
}

func (s *AttendanceService) checkEnrollmentWithFallback(ctx context.Context, sessionID string, studentID, courseID uuid.UUID, semester string) (bool, error) {
	// Positive hit trusted; miss may be stale (late enrollment) so verify against DB.
	enrolled, err := s.redisService.IsStudentEnrolled(ctx, sessionID, studentID.String())
	if err == nil && enrolled {
		return true, nil
	}
	return s.cacheRepo.CheckEnrollment(ctx, studentID, courseID, semester)
}

// failedAttendancePayload wraps the event data in the BaseEvent envelope
// grades' consumer reads.
func failedAttendancePayload(data dto.AttendanceSemesterFailedEventData) ([]byte, error) {
	payload, err := json.Marshal(dto.BaseEvent{
		EventID:   uuid.New(),
		EventType: events.EventAttendanceSemesterFailed,
		Timestamp: clock.Now(),
		Data:      data,
	})
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", events.EventAttendanceSemesterFailed, err)
	}
	return payload, nil
}

// GetSessionDetails returns session details for instructor
func (s *AttendanceService) GetSessionDetails(ctx context.Context, sessionID, instructorID uuid.UUID, isAdmin bool) (dto.GetSessionDetailsResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetSessionDetailsResponse{}, errors.ErrSessionNotFound
	}

	if !canManageSession(session.InstructorID, instructorID, isAdmin) {
		return dto.GetSessionDetailsResponse{}, errors.ErrForbidden
	}

	// Get course info
	courseID := utils.PgUUIDToUUID(session.CourseID)
	course, err := s.cacheRepo.GetCourseCacheByID(ctx, courseID)
	if err != nil {
		return dto.GetSessionDetailsResponse{}, errors.ErrCourseNotFound
	}

	// Get enrolled count
	enrolledStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, courseID, session.Semester)
	if err != nil {
		logger.Error("failed to get enrolled students", zap.Error(err))
		return dto.GetSessionDetailsResponse{}, err
	}

	// Get present count
	presentCount, err := s.attendanceRepo.GetSessionAttendanceCount(ctx, sessionID)
	if err != nil {
		logger.Error("failed to get attendance count", zap.Error(err))
		return dto.GetSessionDetailsResponse{}, err
	}

	totalEnrolled := len(enrolledStudents)

	return dto.GetSessionDetailsResponse{
		SessionID:            sessionID,
		CourseID:             courseID,
		CourseCode:           course.CourseCode,
		CourseName:           course.CourseName,
		WeekNumber:           session.WeekNumber,
		SessionType:          string(session.SessionType),
		SessionDate:          session.SessionDate.Time.Format("2006-01-02"),
		Semester:             session.Semester,
		IsActive:             utils.PgBoolToBool(session.IsActive),
		StartedAt:            utils.PgTimestampToTime(session.StartedAt),
		ExpiresAt:            utils.PgTimestampToTime(session.ExpiresAt),
		EnrolledStudentCount: totalEnrolled,
		PresentCount:         int(presentCount),
		AbsentCount:          totalEnrolled - int(presentCount),
		QRRotationInterval:   int(qrRotation(session) / time.Second),
	}, nil
}

// GetSessionRecords returns attendance records for a session
func (s *AttendanceService) GetSessionRecords(ctx context.Context, sessionID, instructorID uuid.UUID, isAdmin bool) (dto.GetSessionRecordsResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetSessionRecordsResponse{}, errors.ErrSessionNotFound
	}

	if !canManageSession(session.InstructorID, instructorID, isAdmin) {
		return dto.GetSessionRecordsResponse{}, errors.ErrForbidden
	}

	// Get all attendance records for this session (all records = present students)
	records, err := s.attendanceRepo.GetAttendanceRecordsBySession(ctx, sessionID)
	if err != nil {
		return dto.GetSessionRecordsResponse{}, err
	}

	var items []dto.AttendanceRecordItem

	for _, record := range records {
		studentID := utils.PgUUIDToUUID(record.StudentID)
		student, err := s.cacheRepo.GetStudentCacheByID(ctx, studentID)
		if err != nil {
			logger.Error("failed to get student info", zap.String("student_id", studentID.String()), zap.Error(err))
			continue
		}

		markedAt := utils.PgTimestampToTime(record.MarkedAt)
		var markedAtPtr *time.Time
		if !markedAt.IsZero() {
			markedAtPtr = &markedAt
		}

		var notePtr *string
		if record.ManualNote.Valid {
			notePtr = &record.ManualNote.String
		}

		items = append(items, dto.AttendanceRecordItem{
			ID:            utils.PgUUIDToUUID(record.ID),
			StudentID:     studentID,
			StudentNumber: student.StudentNumber,
			StudentName:   fmt.Sprintf("%s %s", utils.PgTextToString(student.FirstName), utils.PgTextToString(student.LastName)),
			MarkedVia:     record.MarkedVia,
			MarkedAt:      markedAtPtr,
			Note:          notePtr,
		})
	}

	return dto.GetSessionRecordsResponse{
		SessionID:    sessionID,
		WeekNumber:   session.WeekNumber,
		PresentCount: len(records),
		Records:      items,
	}, nil
}

// GetSessionStudents returns enrolled students for a session with their marked status
func (s *AttendanceService) GetSessionStudents(ctx context.Context, sessionID, instructorID uuid.UUID, isAdmin bool, search string) (dto.GetSessionStudentsResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetSessionStudentsResponse{}, errors.ErrSessionNotFound
	}

	if !canManageSession(session.InstructorID, instructorID, isAdmin) {
		return dto.GetSessionStudentsResponse{}, errors.ErrForbidden
	}

	courseID := utils.PgUUIDToUUID(session.CourseID)

	// Get enrolled students
	enrolledStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, courseID, session.Semester)
	if err != nil {
		return dto.GetSessionStudentsResponse{}, err
	}

	// Get marked students
	markedStudents, err := s.attendanceRepo.GetMarkedStudentsBySession(ctx, sessionID)
	if err != nil {
		return dto.GetSessionStudentsResponse{}, err
	}

	markedMap := make(map[uuid.UUID]bool)
	for _, id := range markedStudents {
		markedMap[id] = true
	}

	var students []dto.EnrolledStudentItem
	markedCount := 0

	for _, student := range enrolledStudents {
		studentID := utils.PgUUIDToUUID(student.ID)
		firstName := utils.PgTextToString(student.FirstName)
		lastName := utils.PgTextToString(student.LastName)
		email := utils.PgTextToString(student.Email)

		// Apply search filter if provided
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(student.StudentNumber), searchLower) &&
				!strings.Contains(strings.ToLower(firstName), searchLower) &&
				!strings.Contains(strings.ToLower(lastName), searchLower) {
				continue
			}
		}

		isMarked := markedMap[studentID]
		if isMarked {
			markedCount++
		}

		students = append(students, dto.EnrolledStudentItem{
			StudentID:     studentID,
			StudentNumber: student.StudentNumber,
			FirstName:     firstName,
			LastName:      lastName,
			Email:         email,
			IsMarked:      isMarked,
		})
	}

	return dto.GetSessionStudentsResponse{
		SessionID:     sessionID,
		CourseID:      courseID,
		TotalEnrolled: len(enrolledStudents),
		MarkedCount:   markedCount,
		Students:      students,
	}, nil
}

// GetSessionsByDateRange returns all sessions within a date range (admin use)
func (s *AttendanceService) GetSessionsByDateRange(ctx context.Context, startDate, endDate time.Time) (dto.AdminSessionsResponse, error) {
	start := utils.TimeToPgDate(startDate)
	end := utils.TimeToPgDate(endDate)

	rows, err := s.sessionRepo.GetSessionsByDateRange(ctx, db.GetSessionsByDateRangeParams{
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return dto.AdminSessionsResponse{}, fmt.Errorf("failed to get sessions: %w", err)
	}

	sessions := make([]dto.AdminSessionItem, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, dto.AdminSessionItem{
			SessionID:     utils.PgUUIDToUUID(row.ID),
			CourseID:      utils.PgUUIDToUUID(row.CourseID),
			CourseCode:    row.CourseCode,
			CourseName:    row.CourseName,
			InstructorID:  utils.PgUUIDToUUID(row.InstructorID),
			Semester:      row.Semester,
			WeekNumber:    row.WeekNumber,
			SessionType:   string(row.SessionType),
			SessionDate:   row.SessionDate.Time.Format("2006-01-02"),
			IsActive:      row.IsActive.Bool,
			StartedAt:     row.StartedAt.Time,
			ExpiresAt:     row.ExpiresAt.Time,
			PresentCount:  row.PresentCount,
			EnrolledCount: row.EnrolledCount,
		})
	}

	return dto.AdminSessionsResponse{
		Sessions: sessions,
		Total:    len(sessions),
	}, nil
}

// checkSemesterEnforcement checks hard_deadline + admin bypass + period window.
// Semester enforcement: Uses CanOperateInSemester() — the three-layer model.
// Admin can override period but NOT hard_deadline.
func (s *AttendanceService) checkSemesterEnforcement(ctx context.Context, semester string, isAdmin bool) error {
	// Fetch hard_deadline from catalog service
	semesterInfo, err := s.semesterClient.GetSemesterInfo(ctx, semester)
	if err != nil {
		// An unreachable catalog is not a semester without a hard deadline.
		// The deadline binds admins too, so a check we cannot perform has to
		// stop the request instead of waving it through.
		if goerrors.Is(err, client.ErrUnavailable) {
			logger.Error("catalog unreachable, refusing to skip semester enforcement",
				zap.String("semester", semester),
				zap.Error(err),
			)
			return errors.ErrSemesterInfoUnavailable
		}
		logger.Warn("failed to fetch semester info, skipping enforcement",
			zap.String("semester", semester),
			zap.Error(err),
		)
		return nil // graceful degradation: semester unknown to catalog
	}

	// Fetch period from local DB
	var periodStart, periodEnd *time.Time
	period, periodErr := s.periodRepo.GetActivePeriodBySemester(ctx, semester)
	if periodErr == nil {
		periodStart = &period.PeriodStart
		periodEnd = &period.PeriodEnd
	}

	result := rules.CanOperateInSemester(rules.SemesterOperationParams{
		HardDeadline:  semesterInfo.HardDeadline,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		IsAdminAction: isAdmin,
	})

	if !result.Allowed {
		switch result.Reason {
		case "semester_ended":
			return errors.ErrSemesterEnded
		case "period_not_started":
			return errors.ErrPeriodNotStarted
		case "period_ended":
			return errors.ErrPeriodEnded
		default:
			return errors.ErrPeriodEnded
		}
	}

	return nil
}
