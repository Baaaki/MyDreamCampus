package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewAttendanceRepository(pool *pgxpool.Pool) *AttendanceRepository {
	return &AttendanceRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *AttendanceRepository) CreateAttendanceRecordQR(ctx context.Context, params db.CreateAttendanceRecordQRParams) error {
	return r.queries.CreateAttendanceRecordQR(ctx, params)
}

// BatchCreateAttendanceRecordsQR inserts many QR attendance records in a single query via UNNEST.
// ON CONFLICT DO NOTHING makes the call idempotent so flush retries are safe.
func (r *AttendanceRepository) BatchCreateAttendanceRecordsQR(ctx context.Context, records []db.CreateAttendanceRecordQRParams) error {
	if len(records) == 0 {
		return nil
	}

	params := db.BatchCreateAttendanceRecordsQRParams{
		SessionIds:   make([]pgtype.UUID, len(records)),
		StudentIds:   make([]pgtype.UUID, len(records)),
		CourseIds:    make([]pgtype.UUID, len(records)),
		Semesters:    make([]string, len(records)),
		WeekNumbers:  make([]int16, len(records)),
		ScannedAts:   make([]pgtype.Timestamp, len(records)),
		QrTimestamps: make([]int64, len(records)),
		SessionTypes: make([]interface{}, len(records)),
	}

	for i, rec := range records {
		params.SessionIds[i] = rec.SessionID
		params.StudentIds[i] = rec.StudentID
		params.CourseIds[i] = rec.CourseID
		params.Semesters[i] = rec.Semester
		params.WeekNumbers[i] = rec.WeekNumber
		params.ScannedAts[i] = rec.ScannedAt
		if rec.QrTimestamp.Valid {
			params.QrTimestamps[i] = rec.QrTimestamp.Int64
		}
		params.SessionTypes[i] = rec.SessionType
	}

	return r.queries.BatchCreateAttendanceRecordsQR(ctx, params)
}

func (r *AttendanceRepository) CreateAttendanceRecordManual(ctx context.Context, params db.CreateAttendanceRecordManualParams) (db.AttendanceAttendanceRecord, error) {
	return r.queries.CreateAttendanceRecordManual(ctx, params)
}

func (r *AttendanceRepository) GetMarkedStudentsBySession(ctx context.Context, sessionID uuid.UUID) ([]uuid.UUID, error) {
	pgtypeUUIDs, err := r.queries.GetMarkedStudentsBySession(ctx, utils.UUIDToPgUUID(sessionID))
	if err != nil {
		return nil, err
	}

	result := make([]uuid.UUID, len(pgtypeUUIDs))
	for i, pgtypeUUID := range pgtypeUUIDs {
		result[i] = utils.PgUUIDToUUID(pgtypeUUID)
	}
	return result, nil
}

func (r *AttendanceRepository) GetSessionAttendanceCount(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	return r.queries.GetSessionAttendanceCount(ctx, utils.UUIDToPgUUID(sessionID))
}

func (r *AttendanceRepository) GetStudentAttendanceByCourse(ctx context.Context, studentID, courseID uuid.UUID, semester string) ([]db.GetStudentAttendanceByCourseRow, error) {
	return r.queries.GetStudentAttendanceByCourse(ctx, db.GetStudentAttendanceByCourseParams{
		StudentID: utils.UUIDToPgUUID(studentID),
		CourseID:  utils.UUIDToPgUUID(courseID),
		Semester:  semester,
	})
}

func (r *AttendanceRepository) GetCourseAttendanceStats(ctx context.Context, courseID uuid.UUID, semester string) ([]db.GetCourseAttendanceStatsRow, error) {
	return r.queries.GetCourseAttendanceStats(ctx, db.GetCourseAttendanceStatsParams{
		CourseID: utils.UUIDToPgUUID(courseID),
		Semester: semester,
	})
}

func (r *AttendanceRepository) GetFailingStudentsByCourse(ctx context.Context, courseID uuid.UUID, semester string, totalSessions int64, maxAllowedAbsences int64) ([]db.GetFailingStudentsByCourseRow, error) {
	return r.queries.GetFailingStudentsByCourse(ctx, db.GetFailingStudentsByCourseParams{
		CourseID:           utils.UUIDToPgUUID(courseID),
		Semester:           semester,
		TotalSessions:      totalSessions,
		MaxAllowedAbsences: maxAllowedAbsences,
	})
}

func (r *AttendanceRepository) GetAttendanceRecordsBySession(ctx context.Context, sessionID uuid.UUID) ([]db.GetAttendanceRecordsBySessionRow, error) {
	return r.queries.GetAttendanceRecordsBySession(ctx, utils.UUIDToPgUUID(sessionID))
}

func (r *AttendanceRepository) GetTotalSessionsByCourse(ctx context.Context, courseID uuid.UUID, semester string) (int64, error) {
	return r.queries.GetTotalSessionsByCourse(ctx, db.GetTotalSessionsByCourseParams{
		CourseID: utils.UUIDToPgUUID(courseID),
		Semester: semester,
	})
}

func (r *AttendanceRepository) GetTotalSessionsByCourseAndType(ctx context.Context, courseID uuid.UUID, semester string, sessionType db.AttendanceSessionTypeEnum) (int64, error) {
	return r.queries.GetTotalSessionsByCourseAndType(ctx, db.GetTotalSessionsByCourseAndTypeParams{
		CourseID:    utils.UUIDToPgUUID(courseID),
		Semester:    semester,
		SessionType: sessionType,
	})
}

func (r *AttendanceRepository) GetFailingStudentsByCourseByType(ctx context.Context, courseID uuid.UUID, semester string, sessionType db.AttendanceSessionTypeEnum, totalSessions int64, minRequired int64) ([]db.GetFailingStudentsByCourseByTypeRow, error) {
	return r.queries.GetFailingStudentsByCourseByType(ctx, db.GetFailingStudentsByCourseByTypeParams{
		CourseID:              utils.UUIDToPgUUID(courseID),
		Semester:              semester,
		SessionType:           sessionType,
		TotalSessions:         totalSessions,
		MinRequiredAttendance: minRequired,
	})
}

func (r *AttendanceRepository) GetStudentPresentCountByType(ctx context.Context, studentID, courseID uuid.UUID, semester string, sessionType db.AttendanceSessionTypeEnum) (int64, error) {
	return r.queries.GetStudentPresentCountByType(ctx, db.GetStudentPresentCountByTypeParams{
		StudentID:   utils.UUIDToPgUUID(studentID),
		CourseID:    utils.UUIDToPgUUID(courseID),
		Semester:    semester,
		SessionType: sessionType,
	})
}
