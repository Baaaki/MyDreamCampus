package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewImportRepository(pool *pgxpool.Pool) *ImportRepository {
	return &ImportRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateImportJob creates a new import job
func (r *ImportRepository) CreateImportJob(ctx context.Context, params db.CreateImportJobParams) (db.ImportJob, error) {
	job, err := r.queries.CreateImportJob(ctx, params)
	if err != nil {
		return db.ImportJob{}, fmt.Errorf("failed to create import job: %w", err)
	}
	return job, nil
}

// GetImportJobByID retrieves import job by ID
func (r *ImportRepository) GetImportJobByID(ctx context.Context, id uuid.UUID) (db.ImportJob, error) {
	job, err := r.queries.GetImportJobByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return db.ImportJob{}, fmt.Errorf("failed to get import job: %w", err)
	}
	return job, nil
}

// UpdateImportJobProgress updates import job progress
func (r *ImportRepository) UpdateImportJobProgress(ctx context.Context, params db.UpdateImportJobProgressParams) error {
	err := r.queries.UpdateImportJobProgress(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update import job progress: %w", err)
	}
	return nil
}

// StartImportJob marks import job as processing
func (r *ImportRepository) StartImportJob(ctx context.Context, id uuid.UUID) error {
	err := r.queries.StartImportJob(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("failed to start import job: %w", err)
	}
	return nil
}

// CompleteImportJob marks import job as completed
func (r *ImportRepository) CompleteImportJob(ctx context.Context, id uuid.UUID) error {
	err := r.queries.CompleteImportJob(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("failed to complete import job: %w", err)
	}
	return nil
}

// FailImportJob marks import job as failed
func (r *ImportRepository) FailImportJob(ctx context.Context, id uuid.UUID) error {
	err := r.queries.FailImportJob(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("failed to mark import job as failed: %w", err)
	}
	return nil
}

// ListImportJobsByUser lists import jobs for a user
func (r *ImportRepository) ListImportJobsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]db.ImportJob, int64, error) {
	// Get total count
	count, err := r.queries.CountImportJobsByUser(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count import jobs: %w", err)
	}

	// Get jobs list
	jobs, err := r.queries.ListImportJobsByUser(ctx, db.ListImportJobsByUserParams{
		CreatedBy: utils.UUIDToPgtype(userID),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list import jobs: %w", err)
	}

	return jobs, count, nil
}

// BulkInsertStudents performs bulk insert using PostgreSQL COPY command
func (r *ImportRepository) BulkInsertStudents(ctx context.Context, students []db.CreateStudentParams) error {
	// Use pgx CopyFrom for high-performance bulk insert
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare data for COPY
	columns := []string{"student_number", "first_name", "last_name", "email", "faculty", "department", "enrollment_year", "class_level", "advisor_id"}
	rows := make([][]interface{}, len(students))

	for i, student := range students {
		rows[i] = []interface{}{
			student.StudentNumber,
			student.FirstName,
			student.LastName,
			student.Email,
			student.Faculty,
			student.Department,
			student.EnrollmentYear,
			student.ClassLevel,
			student.AdvisorID,
		}
	}

	// Use CopyFrom for bulk insert
	copyCount, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"students"}, // table name
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed to bulk insert students: %w", err)
	}

	if copyCount != int64(len(students)) {
		return fmt.Errorf("expected to insert %d students, but inserted %d", len(students), copyCount)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
