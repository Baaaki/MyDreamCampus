package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	attErrors "github.com/baaaki/mydreamcampus/attendance/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.AttendanceSession{}, fmt.Errorf("begin session tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	courseID := utils.PgUUIDToUUID(params.CourseID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('attendance_course:' || $1::text))`, courseID.String()); err != nil {
		return db.AttendanceSession{}, fmt.Errorf("lock course for session: %w", err)
	}

	qtx := r.queries.WithTx(tx)
	if _, err := qtx.GetCourseFinalization(ctx, params.CourseID); err == nil {
		return db.AttendanceSession{}, attErrors.ErrAttendanceFinalized
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.AttendanceSession{}, fmt.Errorf("check course finalization: %w", err)
	}

	session, err := qtx.CreateAttendanceSession(ctx, params)
	if err != nil {
		return db.AttendanceSession{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.AttendanceSession{}, fmt.Errorf("commit session tx: %w", err)
	}
	return session, nil
}

func (r *SessionRepository) GetActiveSessionIDsByCourse(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM attendance.attendance_sessions WHERE course_id = $1 AND is_active = TRUE`, utils.UUIDToPgUUID(courseID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SessionRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (db.AttendanceSession, error) {
	return r.queries.GetSessionByID(ctx, utils.UUIDToPgUUID(sessionID))
}

func (r *SessionRepository) GetActiveSessionByID(ctx context.Context, sessionID uuid.UUID) (db.AttendanceSession, error) {
	return r.queries.GetActiveSessionByID(ctx, db.GetActiveSessionByIDParams{
		ID:  utils.UUIDToPgUUID(sessionID),
		Now: utils.TimeToPgTimestamp(clock.Now()),
	})
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
	return r.queries.GetExpiredSessions(ctx, utils.TimeToPgTimestamp(clock.Now()))
}

func (r *SessionRepository) GetSessionsByDateRange(ctx context.Context, params db.GetSessionsByDateRangeParams) ([]db.GetSessionsByDateRangeRow, error) {
	return r.queries.GetSessionsByDateRange(ctx, params)
}
