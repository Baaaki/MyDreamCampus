package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/monolith/internal/platform/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rules"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	CacheTTLBuffer          = 5 * time.Minute
	MinTheoryAttendance     = 10 // 14 haftadan en az 10 teorik yoklama
	MinLabAttendance        = 11 // 14 haftadan en az 11 uygulama yoklama
	MaxTheoryAbsences       = 4  // 14 - 10
	MaxLabAbsences          = 3  // 14 - 11
)

type AttendanceService struct {
	cacheRepo      *repository.CacheRepository
	sessionRepo    *repository.SessionRepository
	attendanceRepo *repository.AttendanceRepository
	outboxRepo     *repository.OutboxRepository
	qrService      *QRService
	redisService   *RedisService
	semesterClient SemesterClient
	periodRepo     *sharedRepo.SimplePeriodRepository
}

func NewAttendanceService(
	cacheRepo *repository.CacheRepository,
	sessionRepo *repository.SessionRepository,
	attendanceRepo *repository.AttendanceRepository,
	outboxRepo *repository.OutboxRepository,
	qrService *QRService,
	redisService *RedisService,
	semesterClient SemesterClient,
	periodRepo *sharedRepo.SimplePeriodRepository,
) *AttendanceService {
	return &AttendanceService{
		cacheRepo:      cacheRepo,
		sessionRepo:    sessionRepo,
		attendanceRepo: attendanceRepo,
		outboxRepo:     outboxRepo,
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
	sessionType := db.AttendanceSessionTypeEnum(req.SessionType)
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

	session, err := s.sessionRepo.CreateAttendanceSession(ctx, db.CreateAttendanceSessionParams{
		CourseID:           utils.UUIDToPgUUID(req.CourseID),
		InstructorID:       utils.UUIDToPgUUID(instructorID),
		Semester:           course.Semester,
		WeekNumber:         req.WeekNumber,
		SessionDate:        utils.TimeToPgDate(now),
		QrSecret:  qrSecret,
		StartedAt: utils.TimeToPgTimestamp(now),
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
		s.redisService.AddEnrolledStudents(ctx, sessionID, studentIDs)
	}

	// Session cache
	s.redisService.SetSessionCache(ctx, sessionID, map[string]any{
		"course_id":            req.CourseID.String(),
		"instructor_id":        instructorID.String(),
		"semester":             course.Semester,
		"week_number":          fmt.Sprintf("%d", req.WeekNumber),
		"session_type":         req.SessionType,
		"qr_secret":  qrSecret,
		"expires_at": fmt.Sprintf("%d", expiresAt.Unix()),
		"enrolled_count":       fmt.Sprintf("%d", len(enrolledStudents)),
	}, time.Until(expiresAt)+CacheTTLBuffer)

	return dto.CreateSessionResponse{
		SessionID:            utils.PgUUIDToUUID(session.ID),
		CourseID:             req.CourseID,
		CourseCode:           course.CourseCode,
		CourseName:           course.CourseName,
		WeekNumber:  req.WeekNumber,
		SessionType: req.SessionType,
		SessionDate: now.Format("2006-01-02"),
		StartedAt:   now,
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

	// Semester enforcement: checks hard_deadline + period window
	if err := s.checkSemesterEnforcement(ctx, session.Semester, false); err != nil {
		return dto.ScanQRResponse{}, err
	}

	// 2. Validate QR signature
	if !s.qrService.ValidateQRSignature(req.QRPayload, session.QrSecret) {
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

	// Scanned SET TTL = session remaining time + 1 hour safety buffer
	scannedTTL := time.Until(utils.PgTimestampToTime(session.ExpiresAt)) + 1*time.Hour

	added, err := s.redisService.AddToBuffer(ctx, sessionID, studentID.String(), string(bufferJSON), scannedTTL)
	if err != nil {
		// Redis down, write directly to DB. ON CONFLICT keeps this safe on duplicates.
		logger.Warn("Redis down, writing directly to DB", zap.Error(err))
		if err := s.attendanceRepo.CreateAttendanceRecordQR(ctx, recordParams); err != nil {
			return dto.ScanQRResponse{}, err
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

	// Generate QR payload (static for entire session)
	payload := s.qrService.GenerateQRPayload(sessionID.String(), session.QrSecret)

	return dto.GetQRResponse{
		SessionID:  sessionID,
		QRPayload:  payload,
		ValidUntil: utils.PgTimestampToTime(session.ExpiresAt),
	}, nil
}

// CreateManualAttendance creates manual attendance record
func (s *AttendanceService) CreateManualAttendance(ctx context.Context, sessionID, instructorID uuid.UUID, req dto.ManualAttendanceRequest) (dto.ManualAttendanceResponse, error) {
	session, err := s.sessionRepo.GetActiveSessionByID(ctx, sessionID)
	if err != nil {
		return dto.ManualAttendanceResponse{}, errors.ErrSessionNotFound
	}

	// Check ownership
	if utils.PgUUIDToUUID(session.InstructorID) != instructorID {
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
	s.redisService.InvalidateStudentSummary(ctx, req.StudentID, session.Semester)

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
func (s *AttendanceService) CloseSession(ctx context.Context, sessionID, instructorID uuid.UUID) (dto.CloseSessionResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.CloseSessionResponse{}, errors.ErrSessionNotFound
	}

	// Check ownership
	if utils.PgUUIDToUUID(session.InstructorID) != instructorID {
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

	// Clear Redis keys
	s.redisService.ClearSessionKeys(ctx, sessionID.String())

	// Get counts
	courseID := utils.PgUUIDToUUID(session.CourseID)
	enrolledStudents, err := s.cacheRepo.GetEnrolledStudentsByCourse(ctx, courseID, session.Semester)
	if err != nil {
		logger.Error("failed to get enrolled students", zap.Error(err))
		return dto.CloseSessionResponse{}, err
	}

	presentCount, err := s.attendanceRepo.GetSessionAttendanceCount(ctx, sessionID)
	if err != nil {
		logger.Error("failed to get attendance count", zap.Error(err))
		return dto.CloseSessionResponse{}, err
	}

	totalEnrolled := len(enrolledStudents)

	return dto.CloseSessionResponse{
		SessionID: sessionID,
		ClosedAt:  clock.Now(),
		Summary: dto.SessionSummary{
			TotalEnrolled: totalEnrolled,
			PresentCount:  int(presentCount),
			AbsentCount:   totalEnrolled - int(presentCount),
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

	// Get enrollments
	enrollments, err := s.cacheRepo.GetStudentEnrollmentsBySemester(ctx, studentID, semester)
	if err != nil {
		return dto.GetMyAttendanceResponse{}, err
	}

	var courses []dto.CourseAttendanceDetail

	for _, enrollment := range enrollments {
		courseID := utils.PgUUIDToUUID(enrollment.CourseID)

		// Get attendance records (only records where student was present)
		records, err := s.attendanceRepo.GetStudentAttendanceByCourse(ctx, studentID, courseID, semester)
		if err != nil {
			continue
		}

		var weeklyRecords []dto.WeeklyAttendanceRecord
		for _, record := range records {
			weeklyRecords = append(weeklyRecords, dto.WeeklyAttendanceRecord{
				Week:        record.WeekNumber,
				SessionType: string(record.SessionType),
				Date:        record.SessionDate.Time.Format("2006-01-02"),
				MarkedVia:   record.MarkedVia,
			})
		}

		// Get theory stats
		theoryTotal, _ := s.attendanceRepo.GetTotalSessionsByCourseAndType(ctx, courseID, semester, db.AttendanceSessionTypeEnumTheory)
		theoryPresent, _ := s.attendanceRepo.GetStudentPresentCountByType(ctx, studentID, courseID, semester, db.AttendanceSessionTypeEnumTheory)

		// Get lab stats
		labTotal, _ := s.attendanceRepo.GetTotalSessionsByCourseAndType(ctx, courseID, semester, db.AttendanceSessionTypeEnumLab)
		labPresent, _ := s.attendanceRepo.GetStudentPresentCountByType(ctx, studentID, courseID, semester, db.AttendanceSessionTypeEnumLab)

		detail := dto.CourseAttendanceDetail{
			CourseID:      courseID,
			CourseCode:    enrollment.CourseCode,
			CourseName:    enrollment.CourseName,
			Instructor:    utils.PgTextToString(enrollment.InstructorFullname),
			TotalWeeks:    enrollment.TotalWeeks.Int16,
			WeeklyRecords: weeklyRecords,
		}

		if theoryTotal > 0 {
			detail.Theory = &dto.SessionTypeAttendance{
				PresentCount:  int(theoryPresent),
				AbsentCount:   int(theoryTotal) - int(theoryPresent),
				TotalSessions: int(theoryTotal),
				MinRequired:   MinTheoryAttendance,
				Passed:        int(theoryPresent) >= MinTheoryAttendance,
			}
		}

		if labTotal > 0 {
			detail.Lab = &dto.SessionTypeAttendance{
				PresentCount:  int(labPresent),
				AbsentCount:   int(labTotal) - int(labPresent),
				TotalSessions: int(labTotal),
				MinRequired:   MinLabAttendance,
				Passed:        int(labPresent) >= MinLabAttendance,
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

// FinalizeAttendance finalizes attendance for a course and publishes events
// Checks theory (min 10/14) and lab (min 11/14) separately
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

	// Get total sessions by type
	theoryTotal, _ := s.attendanceRepo.GetTotalSessionsByCourseAndType(ctx, courseID, semester, db.AttendanceSessionTypeEnumTheory)
	labTotal, _ := s.attendanceRepo.GetTotalSessionsByCourseAndType(ctx, courseID, semester, db.AttendanceSessionTypeEnumLab)

	// Get failing students for theory (present_count < MinTheoryAttendance)
	var theoryFailing []db.GetFailingStudentsByCourseByTypeRow
	if theoryTotal > 0 {
		theoryFailing, err = s.attendanceRepo.GetFailingStudentsByCourseByType(ctx, courseID, semester, db.AttendanceSessionTypeEnumTheory, theoryTotal, int64(MinTheoryAttendance))
		if err != nil {
			return dto.FinalizeAttendanceResponse{}, err
		}
	}

	// Get failing students for lab (present_count < MinLabAttendance)
	var labFailing []db.GetFailingStudentsByCourseByTypeRow
	if labTotal > 0 {
		labFailing, err = s.attendanceRepo.GetFailingStudentsByCourseByType(ctx, courseID, semester, db.AttendanceSessionTypeEnumLab, labTotal, int64(MinLabAttendance))
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

	var failed []dto.FailedStudent
	eventsPublished := 0

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
				MinRequired:   MinTheoryAttendance,
				Passed:        !info.TheoryFailed,
			}
		}

		if labTotal > 0 {
			failedStudent.Lab = &dto.SessionTypeAttendance{
				PresentCount:  info.LabPresent,
				AbsentCount:   info.LabAbsent,
				TotalSessions: int(labTotal),
				MinRequired:   MinLabAttendance,
				Passed:        !info.LabFailed,
			}
		}

		failed = append(failed, failedStudent)

		// Publish event
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
				MinRequired:   MinTheoryAttendance,
			}
		}

		if labTotal > 0 {
			eventData.Lab = &dto.AttendanceFailedTypeDetail{
				TotalSessions: int(labTotal),
				PresentCount:  info.LabPresent,
				AbsentCount:   info.LabAbsent,
				MinRequired:   MinLabAttendance,
			}
		}

		if err := s.publishFailedAttendanceEvent(ctx, eventData); err == nil {
			eventsPublished++
		}
	}

	return dto.FinalizeAttendanceResponse{
		CourseID:      courseID,
		CourseCode:    course.CourseCode,
		Semester:      semester,
		TotalStudents: len(allStudents),
		TotalWeeks:    course.TotalWeeks.Int16,
		Thresholds: struct {
			TheoryMinRequired int `json:"theory_min_required"`
			LabMinRequired    int `json:"lab_min_required"`
		}{
			TheoryMinRequired: MinTheoryAttendance,
			LabMinRequired:    MinLabAttendance,
		},
		FinalizationSummary: struct {
			PassingCount int `json:"passing_count"`
			FailingCount int `json:"failing_count"`
		}{
			PassingCount: len(allStudents) - len(failMap),
			FailingCount: len(failMap),
		},
		FailedStudents:  failed,
		EventsPublished: eventsPublished,
		FinalizedAt:     clock.Now(),
	}, nil
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

func (s *AttendanceService) getSessionWithFallback(ctx context.Context, sessionID string) (db.AttendanceAttendanceSession, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return db.AttendanceAttendanceSession{}, fmt.Errorf("invalid session ID: %w", err)
	}
	// DB is source of truth; session cache is cleared on close and not rehydrated here.
	return s.sessionRepo.GetActiveSessionByID(ctx, sessionUUID)
}

func (s *AttendanceService) checkEnrollmentWithFallback(ctx context.Context, sessionID string, studentID, courseID uuid.UUID, semester string) (bool, error) {
	// Positive hit trusted; miss may be stale (late enrollment) so verify against DB.
	enrolled, err := s.redisService.IsStudentEnrolled(ctx, sessionID, studentID.String())
	if err == nil && enrolled {
		return true, nil
	}
	return s.cacheRepo.CheckEnrollment(ctx, studentID, courseID, semester)
}

// publishFailedAttendanceEvent publishes event to outbox
func (s *AttendanceService) publishFailedAttendanceEvent(ctx context.Context, data dto.AttendanceSemesterFailedEventData) error {
	payload, err := json.Marshal(dto.BaseEvent{
		EventID:   uuid.New(),
		EventType: events.EventAttendanceSemesterFailed,
		Timestamp: clock.Now(),
		Data:      data,
	})
	if err != nil {
		return err
	}

	return s.outboxRepo.CreateOutboxEvent(ctx, events.EventAttendanceSemesterFailed, events.EventAttendanceSemesterFailed, payload)
}

// GetSessionDetails returns session details for instructor
func (s *AttendanceService) GetSessionDetails(ctx context.Context, sessionID, instructorID uuid.UUID) (dto.GetSessionDetailsResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetSessionDetailsResponse{}, errors.ErrSessionNotFound
	}

	// Check ownership
	if utils.PgUUIDToUUID(session.InstructorID) != instructorID {
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
		IsActive:  utils.PgBoolToBool(session.IsActive),
		StartedAt: utils.PgTimestampToTime(session.StartedAt),
		ExpiresAt:            utils.PgTimestampToTime(session.ExpiresAt),
		EnrolledStudentCount: totalEnrolled,
		PresentCount:         int(presentCount),
		AbsentCount:          totalEnrolled - int(presentCount),
	}, nil
}

// GetSessionRecords returns attendance records for a session
func (s *AttendanceService) GetSessionRecords(ctx context.Context, sessionID, instructorID uuid.UUID) (dto.GetSessionRecordsResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetSessionRecordsResponse{}, errors.ErrSessionNotFound
	}

	// Check ownership
	if utils.PgUUIDToUUID(session.InstructorID) != instructorID {
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
func (s *AttendanceService) GetSessionStudents(ctx context.Context, sessionID, instructorID uuid.UUID, search string) (dto.GetSessionStudentsResponse, error) {
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return dto.GetSessionStudentsResponse{}, errors.ErrSessionNotFound
	}

	// Check ownership
	if utils.PgUUIDToUUID(session.InstructorID) != instructorID {
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
// See: docs/semester-wizard-plan.md
func (s *AttendanceService) checkSemesterEnforcement(ctx context.Context, semester string, isAdmin bool) error {
	// Fetch hard_deadline from catalog service
	semesterInfo, err := s.semesterClient.GetSemesterInfo(ctx, semester)
	if err != nil {
		logger.Warn("failed to fetch semester info, skipping enforcement",
			zap.String("semester", semester),
			zap.Error(err),
		)
		return nil // graceful degradation: if catalog is unreachable, allow operation
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
