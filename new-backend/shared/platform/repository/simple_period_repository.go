package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Period types stored in the catalog source-of-truth table: one row per
// consuming service, so each consumer's deadline can be moved on its own.
const (
	PeriodTypeCatalog    = "catalog"
	PeriodTypeEnrollment = "enrollment"
	PeriodTypeGrading    = "grading"
	PeriodTypeAttendance = "attendance"
)

// ProjectedPeriodTypes are the types catalog ships to other services as
// events. The catalog type stays local — catalog reads its own row directly.
var ProjectedPeriodTypes = []string{PeriodTypeEnrollment, PeriodTypeGrading, PeriodTypeAttendance}

// catalogPeriodSchema holds the source of truth. Its academic_periods table
// carries period_type; every other schema holds an event-fed projection with
// one row per semester and no type column.
const catalogPeriodSchema = "course_catalog"

// allowedPeriodSchemas gates the schema name before it reaches SQL as an
// identifier. The value comes from wiring code, never from a request, but
// identifiers cannot be parameterised so it is whitelisted anyway.
var allowedPeriodSchemas = map[string]bool{
	catalogPeriodSchema: true,
	"enrollment":        true,
	"attendance":        true,
	"grades":            true,
}

// SimplePeriod represents a row in the academic_periods table (without course_id).
// Used by services that only need global semester-level deadlines.
type SimplePeriod struct {
	ID          uuid.UUID
	Semester    string
	PeriodStart time.Time
	PeriodEnd   time.Time
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TypedPeriod pairs a catalog row with the service it belongs to. Only the
// catalog table has this shape.
type TypedPeriod struct {
	SimplePeriod
	PeriodType string
}

// periodExecutor is satisfied by both *pgxpool.Pool and pgx.Tx so a period
// write can join the caller's transaction — the outbox row announcing it has
// to commit or roll back with it.
type periodExecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// SimplePeriodRepository reads and writes the academic_periods table of one
// schema. Raw pgx instead of sqlc because the table exists in four schemas
// with the same shape and the query text only differs by identifier.
type SimplePeriodRepository struct {
	pool  *pgxpool.Pool
	table string
	// hasPeriodType is true for the catalog table. Every query made through
	// the pre-existing (untyped) methods is then scoped to
	// period_type='catalog', so catalog's own deadline checks and the admin
	// period CRUD never read or mutate another service's row.
	hasPeriodType bool
}

// NewSimplePeriodRepository binds the repository to one schema's
// academic_periods table.
func NewSimplePeriodRepository(pool *pgxpool.Pool, schema string) *SimplePeriodRepository {
	if !allowedPeriodSchemas[schema] {
		panic("invalid period schema: " + schema)
	}
	return &SimplePeriodRepository{
		pool:          pool,
		table:         schema + ".academic_periods",
		hasPeriodType: schema == catalogPeriodSchema,
	}
}

// scope appends the period_type predicate to a WHERE clause that already has
// at least one condition, extending args with the type value.
func (r *SimplePeriodRepository) scope(periodType string, args []any) (string, []any) {
	if !r.hasPeriodType {
		return "", args
	}
	args = append(args, periodType)
	return fmt.Sprintf(" AND period_type = $%d", len(args)), args
}

// requireCatalog guards the methods that only make sense against the source
// of truth.
func (r *SimplePeriodRepository) requireCatalog(op string) error {
	if !r.hasPeriodType {
		return fmt.Errorf("%s requires the catalog period table, got %s", op, r.table)
	}
	return nil
}

// CreatePeriod inserts this schema's own period. On the catalog table the row
// is typed 'catalog'; consumer-service rows go through CreatePeriodOfTypeTx.
func (r *SimplePeriodRepository) CreatePeriod(ctx context.Context, p SimplePeriod) (*SimplePeriod, error) {
	return r.createPeriod(ctx, r.pool, p, PeriodTypeCatalog)
}

// CreatePeriodOfTypeTx inserts one consuming service's period row inside the
// caller's transaction so the matching outbox event commits with it.
func (r *SimplePeriodRepository) CreatePeriodOfTypeTx(ctx context.Context, tx pgx.Tx, p SimplePeriod, periodType string) (*SimplePeriod, error) {
	if err := r.requireCatalog("typed period insert"); err != nil {
		return nil, err
	}
	return r.createPeriod(ctx, tx, p, periodType)
}

func (r *SimplePeriodRepository) createPeriod(ctx context.Context, ex periodExecutor, p SimplePeriod, periodType string) (*SimplePeriod, error) {
	columns := "semester, period_start, period_end, is_active"
	values := "$1, $2, $3, COALESCE($4, true)"
	args := []any{p.Semester, p.PeriodStart, p.PeriodEnd, p.IsActive}
	if r.hasPeriodType {
		columns += ", period_type"
		values += ", $5"
		args = append(args, periodType)
	}

	var result SimplePeriod
	err := ex.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
		RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at
	`, r.table, columns, values), args...,
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

// UpsertPeriod writes one projected period row. The event consumer is the only
// writer, and a redelivered event must be a no-op, so catalog's id is carried
// over and the semester unique index drives the conflict target.
func (r *SimplePeriodRepository) UpsertPeriod(ctx context.Context, p SimplePeriod) error {
	if r.hasPeriodType {
		return fmt.Errorf("upsert targets a projection table, not %s", r.table)
	}

	_, err := r.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, semester, period_start, period_end, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (semester) DO UPDATE SET
			period_start = EXCLUDED.period_start,
			period_end   = EXCLUDED.period_end,
			is_active    = EXCLUDED.is_active,
			updated_at   = NOW()
	`, r.table), p.ID, p.Semester, p.PeriodStart, p.PeriodEnd, p.IsActive)
	return err
}

// GetPeriodsBySemester returns all periods for a given semester.
func (r *SimplePeriodRepository) GetPeriodsBySemester(ctx context.Context, semester string) ([]SimplePeriod, error) {
	clause, args := r.scope(PeriodTypeCatalog, []any{semester})

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM %s
		WHERE semester = $1%s
		ORDER BY created_at DESC
	`, r.table, clause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimplePeriods(rows)
}

// GetAllPeriods returns all academic periods.
func (r *SimplePeriodRepository) GetAllPeriods(ctx context.Context) ([]SimplePeriod, error) {
	query := fmt.Sprintf(`
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM %s
	`, r.table)
	var args []any
	if r.hasPeriodType {
		query += " WHERE period_type = $1"
		args = append(args, PeriodTypeCatalog)
	}
	query += " ORDER BY semester DESC, created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimplePeriods(rows)
}

// ListProjectedPeriods returns every catalog row that belongs to a consuming
// service. Feeds the republish endpoint that backfills cold projections.
func (r *SimplePeriodRepository) ListProjectedPeriods(ctx context.Context) ([]TypedPeriod, error) {
	if err := r.requireCatalog("projected period listing"); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at, period_type
		FROM %s
		WHERE period_type <> $1
		ORDER BY semester DESC, period_type
	`, r.table), PeriodTypeCatalog)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []TypedPeriod
	for rows.Next() {
		var p TypedPeriod
		if err := rows.Scan(
			&p.ID, &p.Semester,
			&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt, &p.PeriodType,
		); err != nil {
			return nil, err
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}

// GetPeriodByID returns a single period by its ID.
func (r *SimplePeriodRepository) GetPeriodByID(ctx context.Context, id uuid.UUID) (*SimplePeriod, error) {
	clause, args := r.scope(PeriodTypeCatalog, []any{id})

	var p SimplePeriod
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM %s
		WHERE id = $1%s
	`, r.table, clause), args...).Scan(
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
	clause, args := r.scope(PeriodTypeCatalog, []any{id, periodEnd, isActive})

	var p SimplePeriod
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE %s
		SET period_end = COALESCE($2, period_end),
		    is_active = COALESCE($3, is_active),
		    updated_at = NOW()
		WHERE id = $1%s
		RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at
	`, r.table, clause), args...).Scan(
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
	clause, args := r.scope(PeriodTypeCatalog, []any{id})

	ct, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1%s`, r.table, clause), args...)
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
	clause, args := r.scope(PeriodTypeCatalog, []any{semester})

	var p SimplePeriod
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
		FROM %s
		WHERE semester = $1 AND is_active = true%s
		LIMIT 1
	`, r.table, clause), args...).Scan(
		&p.ID, &p.Semester,
		&p.PeriodStart, &p.PeriodEnd, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePeriodBySemester removes this schema's period for a given semester.
func (r *SimplePeriodRepository) DeletePeriodBySemester(ctx context.Context, semester string) error {
	clause, args := r.scope(PeriodTypeCatalog, []any{semester})

	_, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE semester = $1%s`, r.table, clause), args...)
	return err
}

// UpdatePeriodBySemester updates the period for a semester.
func (r *SimplePeriodRepository) UpdatePeriodBySemester(ctx context.Context, semester string, periodStart, periodEnd time.Time) (*SimplePeriod, error) {
	return r.updatePeriodBySemester(ctx, r.pool, semester, PeriodTypeCatalog, periodStart, periodEnd)
}

// UpdatePeriodBySemesterAndTypeTx moves one consuming service's period inside
// the caller's transaction so the matching outbox event commits with it.
func (r *SimplePeriodRepository) UpdatePeriodBySemesterAndTypeTx(ctx context.Context, tx pgx.Tx, semester, periodType string, periodStart, periodEnd time.Time) (*SimplePeriod, error) {
	if err := r.requireCatalog("typed period update"); err != nil {
		return nil, err
	}
	return r.updatePeriodBySemester(ctx, tx, semester, periodType, periodStart, periodEnd)
}

func (r *SimplePeriodRepository) updatePeriodBySemester(ctx context.Context, ex periodExecutor, semester, periodType string, periodStart, periodEnd time.Time) (*SimplePeriod, error) {
	clause, args := r.scope(periodType, []any{semester, periodStart, periodEnd})

	var p SimplePeriod
	err := ex.QueryRow(ctx, fmt.Sprintf(`
		UPDATE %s
		SET period_start = $2, period_end = $3, updated_at = NOW()
		WHERE semester = $1%s
		RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at
	`, r.table, clause), args...).Scan(
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
