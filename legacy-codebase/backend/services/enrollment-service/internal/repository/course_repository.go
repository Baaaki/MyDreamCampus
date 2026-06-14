package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) *CourseRepository {
	return &CourseRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *CourseRepository) GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]db.SemesterCoursesCache, error) {
	courses, err := r.queries.GetAvailableCourses(ctx, db.GetAvailableCoursesParams{
		Department: utils.StringToPgText(department),
		ClassLevel: utils.Int16ToPgtypeNullable(classLevel),
		Semester:   semester,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get available courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

func (r *CourseRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (db.SemesterCoursesCache, error) {
	course, err := r.queries.GetCourseByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.SemesterCoursesCache{}, fmt.Errorf("%w: course not found", sharedErrors.ErrNotFound)
		}
		return db.SemesterCoursesCache{}, fmt.Errorf("%w: failed to get course: %v", sharedErrors.ErrQueryFailed, err)
	}
	return course, nil
}

func (r *CourseRepository) GetCoursesByIDs(ctx context.Context, ids []uuid.UUID) ([]db.SemesterCoursesCache, error) {
	courses, err := r.queries.GetCoursesByIDs(ctx, utils.UUIDArrayToPgtype(ids))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get courses by IDs: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

func (r *CourseRepository) UpsertSemesterCourse(ctx context.Context, params db.UpsertSemesterCourseParams) (db.SemesterCoursesCache, error) {
	course, err := r.queries.UpsertSemesterCourse(ctx, params)
	if err != nil {
		return db.SemesterCoursesCache{}, fmt.Errorf("%w: failed to upsert semester course: %v", sharedErrors.ErrQueryFailed, err)
	}
	return course, nil
}

func (r *CourseRepository) IncrementEnrollment(ctx context.Context, courseID uuid.UUID) (int64, error) {
	rowsAffected, err := r.queries.IncrementEnrollment(ctx, utils.UUIDToPgtype(courseID))
	if err != nil {
		return 0, fmt.Errorf("%w: failed to increment enrollment: %v", sharedErrors.ErrQueryFailed, err)
	}
	return rowsAffected, nil
}

func (r *CourseRepository) DecrementEnrollment(ctx context.Context, courseID uuid.UUID) error {
	_, err := r.queries.DecrementEnrollment(ctx, utils.UUIDToPgtype(courseID))
	if err != nil {
		return fmt.Errorf("%w: failed to decrement enrollment: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// GetCoursesForCapacityCheck gets courses with FOR UPDATE lock
func (r *CourseRepository) GetCoursesForCapacityCheck(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]db.GetCoursesForCapacityCheckRow, error) {
	qtx := r.queries.WithTx(tx)
	courses, err := qtx.GetCoursesForCapacityCheck(ctx, utils.UUIDArrayToPgtype(ids))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get courses for capacity check: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

func (r *CourseRepository) GetSessionsByCourseIDs(ctx context.Context, ids []uuid.UUID) ([]db.CourseSessionsCache, error) {
	sessions, err := r.queries.GetSessionsByCourseIDs(ctx, utils.UUIDArrayToPgtype(ids))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get sessions: %v", sharedErrors.ErrQueryFailed, err)
	}
	return sessions, nil
}

func (r *CourseRepository) CheckScheduleConflict(ctx context.Context, courseIDs []uuid.UUID) ([]db.CheckScheduleConflictRow, error) {
	conflicts, err := r.queries.CheckScheduleConflict(ctx, utils.UUIDArrayToPgtype(courseIDs))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check schedule conflict: %v", sharedErrors.ErrQueryFailed, err)
	}
	return conflicts, nil
}

func (r *CourseRepository) CheckScheduleConflictWithExisting(ctx context.Context, courseIDs []uuid.UUID, studentID uuid.UUID) ([]db.CheckScheduleConflictRow, error) {
	conflicts, err := r.queries.CheckScheduleConflictWithExisting(ctx, db.CheckScheduleConflictWithExistingParams{
		Dollar1:   utils.UUIDArrayToPgtype(courseIDs),
		StudentID: utils.UUIDToPgtype(studentID),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check schedule conflict with existing: %v", sharedErrors.ErrQueryFailed, err)
	}
	return conflicts, nil
}

func (r *CourseRepository) UpsertCourseSession(ctx context.Context, params db.UpsertCourseSessionParams) (db.CourseSessionsCache, error) {
	session, err := r.queries.UpsertCourseSession(ctx, params)
	if err != nil {
		return db.CourseSessionsCache{}, fmt.Errorf("%w: failed to upsert course session: %v", sharedErrors.ErrQueryFailed, err)
	}
	return session, nil
}

func (r *CourseRepository) DeleteCourseSessionsByCourseID(ctx context.Context, courseID uuid.UUID) error {
	err := r.queries.DeleteCourseSessionsByCourseID(ctx, utils.UUIDToPgtype(courseID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete course sessions: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// UpsertCourseWithSessionsAndProcessedEvent atomically upserts a semester course, replaces its
// sessions, and marks the originating event as processed. Either all changes commit or none —
// so a crash mid-sync cannot leave cached course/session/processed-event state inconsistent.
func (r *CourseRepository) UpsertCourseWithSessionsAndProcessedEvent(
	ctx context.Context,
	courseParams db.UpsertSemesterCourseParams,
	sessionParams []db.UpsertCourseSessionParams,
	eventID uuid.UUID,
	eventType string,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrQueryFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	if _, err := qtx.UpsertSemesterCourse(ctx, courseParams); err != nil {
		return fmt.Errorf("%w: failed to upsert semester course: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := qtx.DeleteCourseSessionsByCourseID(ctx, courseParams.ID); err != nil {
		return fmt.Errorf("%w: failed to delete course sessions: %v", sharedErrors.ErrQueryFailed, err)
	}

	for _, sp := range sessionParams {
		if _, err := qtx.UpsertCourseSession(ctx, sp); err != nil {
			return fmt.Errorf("%w: failed to upsert course session: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	if _, err := qtx.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: eventType,
	}); err != nil {
		return fmt.Errorf("%w: failed to mark event processed: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
