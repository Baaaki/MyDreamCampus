package repository

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CacheRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCacheRepository(pool *pgxpool.Pool) *CacheRepository {
	return &CacheRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Student cache operations
func (r *CacheRepository) UpsertStudentCache(ctx context.Context, student db.UpsertStudentCacheParams) error {
	return r.queries.UpsertStudentCache(ctx, student)
}

func (r *CacheRepository) GetStudentCacheByID(ctx context.Context, studentID uuid.UUID) (db.StudentView, error) {
	return r.queries.GetStudentCacheByID(ctx, utils.UUIDToPgUUID(studentID))
}

func (r *CacheRepository) DeactivateStudentCache(ctx context.Context, studentID uuid.UUID) error {
	return r.queries.DeactivateStudentCache(ctx, utils.UUIDToPgUUID(studentID))
}

// Course cache operations
func (r *CacheRepository) UpsertCourseCache(ctx context.Context, course db.UpsertCourseCacheParams) error {
	return r.queries.UpsertCourseCache(ctx, course)
}

func (r *CacheRepository) GetCourseCacheByID(ctx context.Context, courseID uuid.UUID) (db.CourseView, error) {
	return r.queries.GetCourseCacheByID(ctx, utils.UUIDToPgUUID(courseID))
}

// Enrollment cache operations
// CreateEnrollmentCache records that the student takes the course under
// programID since enrolledAt; a zero enrolledAt (an event without it) means
// from the start.
func (r *CacheRepository) CreateEnrollmentCache(ctx context.Context, programID, studentID, courseID uuid.UUID, semester string, enrolledAt time.Time) error {
	since := pgtype.Timestamp{}
	if !enrolledAt.IsZero() {
		// Stored as UTC like started_at, which it is compared with.
		since = utils.TimeToPgTimestamp(enrolledAt.UTC())
	}
	return r.queries.CreateEnrollmentCache(ctx, db.CreateEnrollmentCacheParams{
		StudentID:  utils.UUIDToPgUUID(studentID),
		CourseID:   utils.UUIDToPgUUID(courseID),
		Semester:   semester,
		EnrolledAt: since,
		ProgramID:  utils.UUIDToPgUUID(programID),
	})
}

// DeleteProgramEnrollments takes the student off the courses of a program
// that was withdrawn or replaced, and returns how many rows it removed.
func (r *CacheRepository) DeleteProgramEnrollments(ctx context.Context, programID, studentID uuid.UUID, courseIDs []uuid.UUID) (int64, error) {
	ids := make([]pgtype.UUID, len(courseIDs))
	for i, id := range courseIDs {
		ids[i] = utils.UUIDToPgUUID(id)
	}
	return r.queries.DeleteProgramEnrollments(ctx, db.DeleteProgramEnrollmentsParams{
		StudentID: utils.UUIDToPgUUID(studentID),
		CourseIds: ids,
		ProgramID: utils.UUIDToPgUUID(programID),
	})
}

func (r *CacheRepository) GetEnrolledStudentsByCourse(ctx context.Context, courseID uuid.UUID, semester string) ([]db.GetEnrolledStudentsByCourseRow, error) {
	return r.queries.GetEnrolledStudentsByCourse(ctx, db.GetEnrolledStudentsByCourseParams{
		CourseID: utils.UUIDToPgUUID(courseID),
		Semester: semester,
	})
}

func (r *CacheRepository) CheckEnrollment(ctx context.Context, studentID, courseID uuid.UUID, semester string) (bool, error) {
	count, err := r.queries.CheckEnrollment(ctx, db.CheckEnrollmentParams{
		StudentID: utils.UUIDToPgUUID(studentID),
		CourseID:  utils.UUIDToPgUUID(courseID),
		Semester:  semester,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *CacheRepository) GetStudentEnrollmentsBySemester(ctx context.Context, studentID uuid.UUID, semester string) ([]db.GetStudentEnrollmentsBySemesterRow, error) {
	return r.queries.GetStudentEnrollmentsBySemester(ctx, db.GetStudentEnrollmentsBySemesterParams{
		StudentID: utils.UUIDToPgUUID(studentID),
		Semester:  semester,
	})
}
