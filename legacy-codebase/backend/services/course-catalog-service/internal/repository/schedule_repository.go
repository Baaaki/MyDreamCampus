package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduleRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewScheduleRepository(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// WithTx returns a new repository instance that uses the given transaction
func (r *ScheduleRepository) WithTx(tx pgx.Tx) *ScheduleRepository {
	return &ScheduleRepository{
		queries: db.New(tx),
		pool:    r.pool,
	}
}

// GetScheduleSessionsByCourseID retrieves all schedule sessions for a semester course
func (r *ScheduleRepository) GetScheduleSessionsByCourseID(ctx context.Context, semesterCourseID uuid.UUID) ([]db.GetScheduleSessionsByCourseIDRow, error) {
	sessions, err := r.queries.GetScheduleSessionsByCourseID(ctx, utils.UUIDToPgtype(semesterCourseID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get schedule sessions: %v", sharedErrors.ErrQueryFailed, err)
	}
	return sessions, nil
}

// GetScheduleSessionsByMultipleCourseIDs retrieves schedule sessions for multiple courses (prevents N+1 query)
func (r *ScheduleRepository) GetScheduleSessionsByMultipleCourseIDs(ctx context.Context, courseIDs []uuid.UUID) ([]db.GetScheduleSessionsByMultipleCourseIDsRow, error) {
	// Convert uuid.UUID slice to pgtype.UUID slice
	pgtypeIDs := make([]pgtype.UUID, len(courseIDs))
	for i, id := range courseIDs {
		pgtypeIDs[i] = utils.UUIDToPgtype(id)
	}

	sessions, err := r.queries.GetScheduleSessionsByMultipleCourseIDs(ctx, pgtypeIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get schedule sessions for multiple courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return sessions, nil
}

// CreateScheduleSession creates a new schedule session
func (r *ScheduleRepository) CreateScheduleSession(ctx context.Context, params db.CreateScheduleSessionParams) (db.CreateScheduleSessionRow, error) {
	session, err := r.queries.CreateScheduleSession(ctx, params)
	if err != nil {
		return db.CreateScheduleSessionRow{}, fmt.Errorf("%w: failed to create schedule session: %v", sharedErrors.ErrQueryFailed, err)
	}
	return session, nil
}

// BulkCreateScheduleSessions creates multiple schedule sessions using sqlc-generated batch operation
// This uses sqlc's :batchexec annotation for type-safe batch inserts
func (r *ScheduleRepository) BulkCreateScheduleSessions(ctx context.Context, sessions []db.CreateScheduleSessionParams) error {
	if len(sessions) == 0 {
		return nil
	}

	// Convert CreateScheduleSessionParams to BatchCreateScheduleSessionsParams
	batchParams := make([]db.BatchCreateScheduleSessionsParams, len(sessions))
	for i, s := range sessions {
		batchParams[i] = db.BatchCreateScheduleSessionsParams{
			SemesterCourseID: s.SemesterCourseID,
			DayOfWeek:        s.DayOfWeek,
			SlotNumber:       s.SlotNumber,
			SessionType:      s.SessionType,
		}
	}

	// Use sqlc-generated batch method
	results := r.queries.BatchCreateScheduleSessions(ctx, batchParams)

	// Execute batch and collect errors
	var batchErr error
	results.Exec(func(i int, err error) {
		if err != nil && batchErr == nil {
			batchErr = fmt.Errorf("%w: batch insert failed at session %d: %v", sharedErrors.ErrQueryFailed, i, err)
		}
	})

	return batchErr
}

// DeleteScheduleSessionsByCourseID deletes all schedule sessions for a semester course
func (r *ScheduleRepository) DeleteScheduleSessionsByCourseID(ctx context.Context, semesterCourseID uuid.UUID) error {
	err := r.queries.DeleteScheduleSessionsByCourseID(ctx, utils.UUIDToPgtype(semesterCourseID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete schedule sessions: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// CheckInstructorScheduleConflict checks if instructor has a schedule conflict
// Returns list of conflicting courses
func (r *ScheduleRepository) CheckInstructorScheduleConflict(ctx context.Context, params db.CheckInstructorScheduleConflictParams) ([]db.CheckInstructorScheduleConflictRow, error) {
	conflicts, err := r.queries.CheckInstructorScheduleConflict(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check schedule conflict: %v", sharedErrors.ErrQueryFailed, err)
	}
	return conflicts, nil
}
