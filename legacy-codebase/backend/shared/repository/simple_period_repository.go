package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SimplePeriod represents a row in the academic_periods table (without course_id).
// Used by services that only need global semester-level deadlines (catalog, enrollment).
type SimplePeriod struct {
	ID          uuid.UUID
	Semester    string
	PeriodStart time.Time
	PeriodEnd   time.Time
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SimplePeriodRepository provides database operations for a simplified academic_periods table
// that has no course_id column. Used by catalog and enrollment services.
type SimplePeriodRepository struct {
	pool *pgxpool.Pool
}

func NewSimplePeriodRepository(pool *pgxpool.Pool) *SimplePeriodRepository {
	return &SimplePeriodRepository{pool: pool}
}

// CreatePeriod inserts a new academic period.
func (r *SimplePeriodRepository) CreatePeriod(ctx context.Context, p SimplePeriod) (*SimplePeriod, error) {
	var result SimplePeriod
	err := r.pool.QueryRow(ctx, `
		INSERT INTO academic_periods (semester, period_start, period_end, is_active)
		VALUES ($1, $2, $3, COALESCE($4, true))
		RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at
	`, p.Semester, p.PeriodStart, p.PeriodEnd, p.IsActive,
	).Scan(
		&result.ID, &result.Semester,
		&result.PeriodStart, &result.PeriodEnd, &result.IsActive,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPeriodsBySemester returns all periods for a given semester.
func (r *SimplePeriodRepository) GetPeriodsBySemester(ctx context.Context, semester string) ([]SimplePeriod, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM academic_periods
		WHERE semester = $1
		ORDER BY created_at DESC
	`, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimplePeriods(rows)
}

// GetAllPeriods returns all academic periods.
func (r *SimplePeriodRepository) GetAllPeriods(ctx context.Context) ([]SimplePeriod, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM academic_periods
		ORDER BY semester DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimplePeriods(rows)
}

// GetPeriodByID returns a single period by its ID.
func (r *SimplePeriodRepository) GetPeriodByID(ctx context.Context, id uuid.UUID) (*SimplePeriod, error) {
	var p SimplePeriod
	err := r.pool.QueryRow(ctx, `
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM academic_periods
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Semester,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePeriod updates a period's end date and/or active status.
func (r *SimplePeriodRepository) UpdatePeriod(ctx context.Context, id uuid.UUID, periodEnd *time.Time, isActive *bool) (*SimplePeriod, error) {
	var p SimplePeriod
	err := r.pool.QueryRow(ctx, `
		UPDATE academic_periods
		SET period_end = COALESCE($2, period_end),
		    is_active = COALESCE($3, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at
	`, id, periodEnd, isActive).Scan(
		&p.ID, &p.Semester,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePeriod removes a period by its ID.
func (r *SimplePeriodRepository) DeletePeriod(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM academic_periods WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetActivePeriodBySemester returns the active period for a given semester.
func (r *SimplePeriodRepository) GetActivePeriodBySemester(ctx context.Context, semester string) (*SimplePeriod, error) {
	var p SimplePeriod
	err := r.pool.QueryRow(ctx, `
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM academic_periods
		WHERE semester = $1 AND is_active = true
		LIMIT 1
	`, semester).Scan(
		&p.ID, &p.Semester,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePeriodBySemester removes all periods for a given semester.
func (r *SimplePeriodRepository) DeletePeriodBySemester(ctx context.Context, semester string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM academic_periods WHERE semester = $1`, semester)
	return err
}

// UpdatePeriodBySemester updates the period for a semester.
func (r *SimplePeriodRepository) UpdatePeriodBySemester(ctx context.Context, semester string, periodStart, periodEnd time.Time) (*SimplePeriod, error) {
	var p SimplePeriod
	err := r.pool.QueryRow(ctx, `
		UPDATE academic_periods
		SET period_start = $2, period_end = $3, updated_at = NOW()
		WHERE semester = $1
		RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at
	`, semester, periodStart, periodEnd).Scan(
		&p.ID, &p.Semester,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanSimplePeriods(rows pgx.Rows) ([]SimplePeriod, error) {
	var periods []SimplePeriod
	for rows.Next() {
		var p SimplePeriod
		if err := rows.Scan(
			&p.ID, &p.Semester,
			&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}
