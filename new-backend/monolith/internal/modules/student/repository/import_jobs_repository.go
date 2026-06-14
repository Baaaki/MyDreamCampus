package repository

import (
	"context"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportJobsRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewImportJobsRepository(pool *pgxpool.Pool) *ImportJobsRepository {
	return &ImportJobsRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateImportJob creates a new import job
func (r *ImportJobsRepository) CreateImportJob(ctx context.Context, params db.CreateImportJobParams) (db.ImportJob, error) {
	job, err := r.queries.CreateImportJob(ctx, params)
	if err != nil {
		return db.ImportJob{}, fmt.Errorf("%w: failed to create import job: %v", sharedErrors.ErrQueryFailed, err)
	}
	return job, nil
}

// GetImportJobByID retrieves import job by ID
func (r *ImportJobsRepository) GetImportJobByID(ctx context.Context, id uuid.UUID) (db.ImportJob, error) {
	job, err := r.queries.GetImportJobByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return db.ImportJob{}, fmt.Errorf("%w: failed to get import job: %v", sharedErrors.ErrQueryFailed, err)
	}
	return job, nil
}

// StartImportJob marks import job as processing
func (r *ImportJobsRepository) StartImportJob(ctx context.Context, id uuid.UUID) error {
	err := r.queries.StartImportJob(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to start import job: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// UpdateImportJobProgress updates import job progress
func (r *ImportJobsRepository) UpdateImportJobProgress(ctx context.Context, params db.UpdateImportJobProgressParams) error {
	err := r.queries.UpdateImportJobProgress(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: failed to update import job progress: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// CompleteImportJob marks import job as completed
func (r *ImportJobsRepository) CompleteImportJob(ctx context.Context, id uuid.UUID) error {
	err := r.queries.CompleteImportJob(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to complete import job: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// FailImportJob marks import job as failed
func (r *ImportJobsRepository) FailImportJob(ctx context.Context, id uuid.UUID) error {
	err := r.queries.FailImportJob(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to mark import job as failed: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// ListImportJobsByUser lists import jobs for a specific user
func (r *ImportJobsRepository) ListImportJobsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]db.ImportJob, int64, error) {
	// Get total count
	total, err := r.queries.CountImportJobsByUser(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to count import jobs: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Get job list
	jobs, err := r.queries.ListImportJobsByUser(ctx, db.ListImportJobsByUserParams{
		CreatedBy: utils.UUIDToPgtype(userID),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to list import jobs: %v", sharedErrors.ErrQueryFailed, err)
	}

	return jobs, total, nil
}
