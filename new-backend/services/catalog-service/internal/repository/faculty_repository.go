package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FacultyRepository struct {
	queries *db.Queries
}

func NewFacultyRepository(pool *pgxpool.Pool) *FacultyRepository {
	return &FacultyRepository{queries: db.New(pool)}
}

func (r *FacultyRepository) ListFaculties(ctx context.Context) ([]db.Faculty, error) {
	faculties, err := r.queries.ListFaculties(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list faculties: %v", sharedErrors.ErrQueryFailed, err)
	}
	return faculties, nil
}

func (r *FacultyRepository) ListDepartments(ctx context.Context) ([]db.Department, error) {
	departments, err := r.queries.ListDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list departments: %v", sharedErrors.ErrQueryFailed, err)
	}
	return departments, nil
}
