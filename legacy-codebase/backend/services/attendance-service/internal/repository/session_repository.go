package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *SessionRepository) CreateAttendanceSession(ctx context.Context, params db.CreateAttendanceSessionParams) (db.AttendanceSession, error) {
	return r.queries.CreateAttendanceSession(ctx, params)
}

func (r *SessionRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (db.AttendanceSession, error) {
	return r.queries.GetSessionByID(ctx, utils.UUIDToPgUUID(sessionID))
}

func (r *SessionRepository) GetActiveSessionByID(ctx context.Context, sessionID uuid.UUID) (db.AttendanceSession, error) {
	return r.queries.GetActiveSessionByID(ctx, utils.UUIDToPgUUID(sessionID))
}

func (r *SessionRepository) CheckSessionExists(ctx context.Context, courseID uuid.UUID, weekNumber int16, sessionType db.SessionTypeEnum) (bool, error) {
	count, err := r.queries.CheckSessionExists(ctx, db.CheckSessionExistsParams{
		CourseID:    utils.UUIDToPgUUID(courseID),
		WeekNumber:  weekNumber,
		SessionType: sessionType,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SessionRepository) GetSessionsByCourse(ctx context.Context, courseID uuid.UUID, semester string) ([]db.GetSessionsByCourseRow, error) {
	return r.queries.GetSessionsByCourse(ctx, db.GetSessionsByCourseParams{
		CourseID: utils.UUIDToPgUUID(courseID),
		Semester: semester,
	})
}

func (r *SessionRepository) DeactivateSession(ctx context.Context, sessionID uuid.UUID) error {
	return r.queries.DeactivateSession(ctx, utils.UUIDToPgUUID(sessionID))
}

func (r *SessionRepository) GetExpiredSessions(ctx context.Context) ([]db.AttendanceSession, error) {
	return r.queries.GetExpiredSessions(ctx)
}

func (r *SessionRepository) GetSessionsByDateRange(ctx context.Context, params db.GetSessionsByDateRangeParams) ([]db.GetSessionsByDateRangeRow, error) {
	return r.queries.GetSessionsByDateRange(ctx, params)
}
