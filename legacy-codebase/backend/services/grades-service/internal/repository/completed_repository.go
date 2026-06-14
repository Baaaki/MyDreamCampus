package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/grades-service/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompletedRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCompletedRepository(pool *pgxpool.Pool) *CompletedRepository {
	return &CompletedRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *CompletedRepository) CreateCompletedCourse(ctx context.Context, arg db.CreateCompletedCourseParams) (db.StudentCompletedCourse, error) {
	return r.queries.CreateCompletedCourse(ctx, arg)
}

func (r *CompletedRepository) DeleteCompletedCourse(ctx context.Context, arg db.DeleteCompletedCourseParams) error {
	return r.queries.DeleteCompletedCourse(ctx, arg)
}

func (r *CompletedRepository) GetCompletedCoursesByStudent(ctx context.Context, studentID uuid.UUID) ([]db.StudentCompletedCourse, error) {
	return r.queries.GetCompletedCoursesByStudent(ctx, studentID)
}

func (r *CompletedRepository) GetCompletedCoursesByCourse(ctx context.Context, courseID uuid.UUID) ([]db.StudentCompletedCourse, error) {
	return r.queries.GetCompletedCoursesByCourse(ctx, courseID)
}

func (r *CompletedRepository) CalculateStudentGPA(ctx context.Context, studentID uuid.UUID) (db.CalculateStudentGPARow, error) {
	return r.queries.CalculateStudentGPA(ctx, studentID)
}

func (r *CompletedRepository) GetTranscriptData(ctx context.Context, studentID uuid.UUID) ([]db.GetTranscriptDataRow, error) {
	return r.queries.GetTranscriptData(ctx, studentID)
}

func (r *CompletedRepository) GetCompletedCourseByStudentAndCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID) (db.StudentCompletedCourse, error) {
	return r.queries.GetCompletedCourseByStudentAndCourse(ctx, db.GetCompletedCourseByStudentAndCourseParams{
		StudentID: studentID,
		CourseID:  courseID,
	})
}

func (r *CompletedRepository) UpdateCompletedCourseAfterAppeal(ctx context.Context, arg db.UpdateCompletedCourseAfterAppealParams) error {
	return r.queries.UpdateCompletedCourseAfterAppeal(ctx, arg)
}
