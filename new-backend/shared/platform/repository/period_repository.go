package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AcademicPeriod represents a row in the academic_periods table.
type AcademicPeriod struct {
	ID          uuid.UUID
	Semester    string
	CourseID    *uuid.UUID // NULL = global period, non-nil = course-specific override
	PeriodStart time.Time
	PeriodEnd   time.Time
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PeriodRepository provides database operations for the academic_periods table.
// Uses raw pgx queries so it can work with any service's database pool.
type PeriodRepository struct {
	pool *pgxpool.Pool
}

func NewPeriodRepository(pool *pgxpool.Pool) *PeriodRepository {
	return &PeriodRepository{pool: pool}
}

// CreatePeriod inserts a new academic period.
func (r *PeriodRepository) CreatePeriod(ctx context.Context, p AcademicPeriod) (*AcademicPeriod, error) {
	var result AcademicPeriod
	err := r.pool.QueryRow(ctx, `
		INSERT INTO course_catalog.academic_periods (semester, course_id, period_start, period_end, is_active)
		VALUES ($1, $2, $3, $4, COALESCE($5, true))
		RETURNING id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
	`, p.Semester, p.CourseID, p.PeriodStart, p.PeriodEnd, p.IsActive,
	).Scan(
		&result.ID, &result.Semester, &result.CourseID,
		&result.PeriodStart, &result.PeriodEnd, &result.IsActive,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPeriodsBySemester returns all periods for a given semester.
func (r *PeriodRepository) GetPeriodsBySemester(ctx context.Context, semester string) ([]AcademicPeriod, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
		FROM course_catalog.academic_periods
		WHERE semester = $1
		ORDER BY created_at DESC
	`, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPeriods(rows)
}

// GetAllPeriods returns all academic periods ordered by semester and creation time.
func (r *PeriodRepository) GetAllPeriods(ctx context.Context) ([]AcademicPeriod, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
		FROM course_catalog.academic_periods
		ORDER BY semester DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPeriods(rows)
}

// GetPeriodByID returns a single period by its ID.
func (r *PeriodRepository) GetPeriodByID(ctx context.Context, id uuid.UUID) (*AcademicPeriod, error) {
	var p AcademicPeriod
	err := r.pool.QueryRow(ctx, `
		SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
		FROM course_catalog.academic_periods
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Semester, &p.CourseID,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePeriod updates a period's end date and/or active status.
func (r *PeriodRepository) UpdatePeriod(ctx context.Context, id uuid.UUID, periodEnd *time.Time, isActive *bool) (*AcademicPeriod, error) {
	var p AcademicPeriod
	err := r.pool.QueryRow(ctx, `
		UPDATE course_catalog.academic_periods
		SET period_end = COALESCE($2, period_end),
		    is_active = COALESCE($3, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
	`, id, periodEnd, isActive).Scan(
		&p.ID, &p.Semester, &p.CourseID,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePeriod removes a period by its ID.
func (r *PeriodRepository) DeletePeriod(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM course_catalog.academic_periods WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetEffectiveDeadline returns the effective period end for a given semester and optional course.
// It first looks for a course-specific override, then falls back to the global period.
// Returns nil if no active period is found.
func (r *PeriodRepository) GetEffectiveDeadline(ctx context.Context, semester string, courseID *uuid.UUID) (*AcademicPeriod, error) {
	// 1. Try course-specific override first
	if courseID != nil {
		var p AcademicPeriod
		err := r.pool.QueryRow(ctx, `
			SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
			FROM course_catalog.academic_periods
			WHERE semester = $1 AND course_id = $2 AND is_active = true
			LIMIT 1
		`, semester, courseID).Scan(
			&p.ID, &p.Semester, &p.CourseID,
			&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err == nil {
			return &p, nil
		}
		// Not found — fall through to global
	}

	// 2. Fall back to global period (course_id IS NULL)
	var p AcademicPeriod
	err := r.pool.QueryRow(ctx, `
		SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
		FROM course_catalog.academic_periods
		WHERE semester = $1 AND course_id IS NULL AND is_active = true
		LIMIT 1
	`, semester).Scan(
		&p.ID, &p.Semester, &p.CourseID,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePeriodBySemester removes all periods for a given semester.
func (r *PeriodRepository) DeletePeriodBySemester(ctx context.Context, semester string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM course_catalog.academic_periods WHERE semester = $1`, semester)
	return err
}

// UpdatePeriodBySemester updates the global (course_id IS NULL) period for a semester.
func (r *PeriodRepository) UpdatePeriodBySemester(ctx context.Context, semester string, periodStart, periodEnd time.Time) (*AcademicPeriod, error) {
	var p AcademicPeriod
	err := r.pool.QueryRow(ctx, `
		UPDATE course_catalog.academic_periods
		SET period_start = $2, period_end = $3, updated_at = NOW()
		WHERE semester = $1 AND course_id IS NULL
		RETURNING id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
	`, semester, periodStart, periodEnd).Scan(
		&p.ID, &p.Semester, &p.CourseID,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPeriods(rows pgx.Rows) ([]AcademicPeriod, error) {
	var periods []AcademicPeriod
	for rows.Next() {
		var p AcademicPeriod
		if err := rows.Scan(
			&p.ID, &p.Semester, &p.CourseID,
			&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}
