package service

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/google/uuid"
)

// AttendanceService depends on these narrow views of its repositories and
// of Redis rather than on the concrete types, so the attendance workflows
// can be tested without a database or a Redis server.

type attendanceCache interface {
	GetCourseCacheByID(ctx context.Context, courseID uuid.UUID) (db.CourseView, error)
	GetStudentCacheByID(ctx context.Context, studentID uuid.UUID) (db.StudentView, error)
	GetEnrolledStudentsByCourse(ctx context.Context, courseID uuid.UUID, semester string) ([]db.GetEnrolledStudentsByCourseRow, error)
	GetStudentEnrollmentsBySemester(ctx context.Context, studentID uuid.UUID, semester string) ([]db.GetStudentEnrollmentsBySemesterRow, error)
	CheckEnrollment(ctx context.Context, studentID, courseID uuid.UUID, semester string) (bool, error)
}

type sessionStore interface {
	CheckSessionExists(ctx context.Context, courseID uuid.UUID, weekNumber int16, sessionType db.SessionTypeEnum) (bool, error)
	CreateAttendanceSession(ctx context.Context, params db.CreateAttendanceSessionParams) (db.AttendanceSession, error)
	GetActiveSessionByID(ctx context.Context, sessionID uuid.UUID) (db.AttendanceSession, error)
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (db.AttendanceSession, error)
	GetSessionsByCourse(ctx context.Context, courseID uuid.UUID, semester string) ([]db.GetSessionsByCourseRow, error)
	GetSessionsByDateRange(ctx context.Context, params db.GetSessionsByDateRangeParams) ([]db.GetSessionsByDateRangeRow, error)
	DeactivateSession(ctx context.Context, sessionID uuid.UUID) error
}

type attendanceStore interface {
	CreateAttendanceRecordQR(ctx context.Context, params db.CreateAttendanceRecordQRParams) (bool, error)
	CreateAttendanceRecordManual(ctx context.Context, params db.CreateAttendanceRecordManualParams) (db.AttendanceRecord, error)
	GetAttendanceRecordsBySession(ctx context.Context, sessionID uuid.UUID) ([]db.GetAttendanceRecordsBySessionRow, error)
	GetMarkedStudentsBySession(ctx context.Context, sessionID uuid.UUID) ([]uuid.UUID, error)
	GetSessionAttendanceCount(ctx context.Context, sessionID uuid.UUID) (int64, error)
	GetStudentAttendanceByCourse(ctx context.Context, studentID, courseID uuid.UUID, semester string) ([]db.GetStudentAttendanceByCourseRow, error)
	GetStudentPresentCountByType(ctx context.Context, studentID, courseID uuid.UUID, semester string, sessionType db.SessionTypeEnum) (int64, error)
	GetTotalSessionsByCourseAndType(ctx context.Context, courseID uuid.UUID, semester string, sessionType db.SessionTypeEnum) (int64, error)
	GetFailingStudentsByCourseByType(ctx context.Context, courseID uuid.UUID, semester string, sessionType db.SessionTypeEnum, totalSessions int64, minRequired int64) ([]db.GetFailingStudentsByCourseByTypeRow, error)
}

type attendanceOutbox interface {
	CreateOutboxEvent(ctx context.Context, eventType, routingKey string, payload []byte) error
}

type attendanceRedis interface {
	SetSessionCache(ctx context.Context, sessionID string, data map[string]any, ttl time.Duration) error
	AddEnrolledStudents(ctx context.Context, sessionID string, studentIDs []uuid.UUID, ttl time.Duration) error
	IsStudentEnrolled(ctx context.Context, sessionID, studentID string) (bool, error)
	AddToBuffer(ctx context.Context, sessionID, studentID, data string, scannedTTL time.Duration) (bool, error)
	GetBuffer(ctx context.Context, sessionID string) (map[string]string, error)
	ClearSessionKeys(ctx context.Context, sessionID string) error
	InvalidateStudentSummary(ctx context.Context, studentID uuid.UUID, semester string) error
}

type attendancePeriods interface {
	GetActivePeriodBySemester(ctx context.Context, semester string) (*sharedRepo.SimplePeriod, error)
}
