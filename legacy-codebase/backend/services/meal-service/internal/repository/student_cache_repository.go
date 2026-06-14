package repository

import (
	"context"
	"errors"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/meal-service/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentCacheRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewStudentCacheRepository(pool *pgxpool.Pool) *StudentCacheRepository {
	return &StudentCacheRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// UpsertStudentCache inserts or updates student cache
func (r *StudentCacheRepository) UpsertStudentCache(ctx context.Context, params db.UpsertStudentCacheParams) (db.StudentsCache, error) {
	student, err := r.queries.UpsertStudentCache(ctx, params)
	if err != nil {
		return db.StudentsCache{}, fmt.Errorf("%w: failed to upsert student cache: %v", sharedErrors.ErrQueryFailed, err)
	}
	return student, nil
}

// GetStudentCacheByID returns student from cache
func (r *StudentCacheRepository) GetStudentCacheByID(ctx context.Context, id uuid.UUID) (db.StudentsCache, error) {
	student, err := r.queries.GetStudentCacheByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.StudentsCache{}, fmt.Errorf("%w", sharedErrors.ErrNotFoundRepo)
		}
		return db.StudentsCache{}, fmt.Errorf("%w: failed to get student cache: %v", sharedErrors.ErrQueryFailed, err)
	}
	return student, nil
}

// DeleteStudentCache deletes student from cache
func (r *StudentCacheRepository) DeleteStudentCache(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteStudentCache(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to delete student cache: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// DeactivateStudentCache marks student as inactive
func (r *StudentCacheRepository) DeactivateStudentCache(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeactivateStudentCache(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to deactivate student cache: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
