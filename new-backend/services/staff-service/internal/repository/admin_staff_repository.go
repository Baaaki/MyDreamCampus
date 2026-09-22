package repository

import (
	"context"
	"errors"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminStaffRepository reads and writes the administrative staff directory.
type AdminStaffRepository struct {
	queries *db.Queries
}

func NewAdminStaffRepository(pool *pgxpool.Pool) *AdminStaffRepository {
	return &AdminStaffRepository{queries: db.New(pool)}
}

// List returns the active records, all of them when faculty is empty.
func (r *AdminStaffRepository) List(ctx context.Context, faculty string) ([]db.AdminStaff, error) {
	filter := pgtype.Text{String: faculty, Valid: faculty != ""}
	rows, err := r.queries.ListAdminStaff(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list admin staff: %v", sharedErrors.ErrQueryFailed, err)
	}
	return rows, nil
}

func (r *AdminStaffRepository) GetByID(ctx context.Context, id uuid.UUID) (db.AdminStaff, error) {
	row, err := r.queries.GetAdminStaffByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AdminStaff{}, fmt.Errorf("%w: admin staff %s", sharedErrors.ErrNotFoundRepo, id)
		}
		return db.AdminStaff{}, fmt.Errorf("%w: failed to get admin staff: %v", sharedErrors.ErrQueryFailed, err)
	}
	return row, nil
}

func (r *AdminStaffRepository) Create(ctx context.Context, params db.CreateAdminStaffParams) (db.AdminStaff, error) {
	row, err := r.queries.CreateAdminStaff(ctx, params)
	if err != nil {
		return db.AdminStaff{}, mapAdminStaffWriteError(err, "create")
	}
	return row, nil
}

func (r *AdminStaffRepository) Update(ctx context.Context, params db.UpdateAdminStaffParams) (db.AdminStaff, error) {
	row, err := r.queries.UpdateAdminStaff(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AdminStaff{}, fmt.Errorf("%w: admin staff for update", sharedErrors.ErrNotFoundRepo)
		}
		return db.AdminStaff{}, mapAdminStaffWriteError(err, "update")
	}
	return row, nil
}

// mapAdminStaffWriteError turns the email unique constraint into
// ErrAlreadyExistsRepo; everything else is a failed query.
func mapAdminStaffWriteError(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return fmt.Errorf("%w: admin staff email already exists", sharedErrors.ErrAlreadyExistsRepo)
	}
	return fmt.Errorf("%w: failed to %s admin staff: %v", sharedErrors.ErrQueryFailed, op, err)
}
