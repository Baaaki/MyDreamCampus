package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
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

func (r *CacheRepository) GetStudentCacheByID(ctx context.Context, studentID uuid.UUID) (db.AttendanceStudentsView, error) {
	return r.queries.GetStudentCacheByID(ctx, utils.UUIDToPgUUID(studentID))
}

func (r *CacheRepository) DeactivateStudentCache(ctx context.Context, studentID uuid.UUID) error {
	return r.queries.DeactivateStudentCache(ctx, utils.UUIDToPgUUID(studentID))
}

// Course cache operations
func (r *CacheRepository) UpsertCourseCache(ctx context.Context, course db.UpsertCourseCacheParams) error {
	return r.queries.UpsertCourseCache(ctx, course)
}

func (r *CacheRepository) GetCourseCacheByID(ctx context.Context, courseID uuid.UUID) (db.AttendanceCoursesView, error) {
	return r.queries.GetCourseCacheByID(ctx, utils.UUIDToPgUUID(courseID))
}

// Enrollment cache operations
func (r *CacheRepository) CreateEnrollmentCache(ctx context.Context, studentID, courseID uuid.UUID, semester string) error {
	return r.queries.CreateEnrollmentCache(ctx, db.CreateEnrollmentCacheParams{
		StudentID: utils.UUIDToPgUUID(studentID),
		CourseID:  utils.UUIDToPgUUID(courseID),
		Semester:  semester,
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
