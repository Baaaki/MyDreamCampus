package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/db"
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

// Student Cache
func (r *CacheRepository) UpsertStudentCache(ctx context.Context, arg db.UpsertStudentCacheParams) (db.GradesStudentsView, error) {
	return r.queries.UpsertStudentCache(ctx, arg)
}

func (r *CacheRepository) GetStudentCacheByID(ctx context.Context, id uuid.UUID) (db.GradesStudentsView, error) {
	return r.queries.GetStudentCacheByID(ctx, id)
}

func (r *CacheRepository) DeactivateStudentCache(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeactivateStudentCache(ctx, id)
}

// Course Cache
func (r *CacheRepository) UpsertCourseCache(ctx context.Context, arg db.UpsertCourseCacheParams) (db.GradesCoursesView, error) {
	return r.queries.UpsertCourseCache(ctx, arg)
}

func (r *CacheRepository) GetCourseCacheByID(ctx context.Context, id uuid.UUID) (db.GradesCoursesView, error) {
	return r.queries.GetCourseCacheByID(ctx, id)
}

func (r *CacheRepository) IsPrerequisiteCourse(ctx context.Context, courseCode string) (bool, error) {
	return r.queries.IsPrerequisiteCourse(ctx, courseCode)
}
