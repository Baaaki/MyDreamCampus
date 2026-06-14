package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *StudentRepository) UpsertStudent(ctx context.Context, params db.UpsertStudentParams) (db.StudentsCache, error) {
	student, err := r.queries.UpsertStudent(ctx, params)
	if err != nil {
		return db.StudentsCache{}, fmt.Errorf("%w: failed to upsert student: %v", sharedErrors.ErrQueryFailed, err)
	}
	return student, nil
}

func (r *StudentRepository) GetStudentByID(ctx context.Context, id uuid.UUID) (db.StudentsCache, error) {
	student, err := r.queries.GetStudentByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.StudentsCache{}, fmt.Errorf("%w: student not found", sharedErrors.ErrNotFound)
		}
		return db.StudentsCache{}, fmt.Errorf("%w: failed to get student: %v", sharedErrors.ErrQueryFailed, err)
	}
	return student, nil
}

func (r *StudentRepository) DeactivateStudent(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeactivateStudent(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to deactivate student: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

func (r *StudentRepository) GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]db.StudentsCache, error) {
	students, err := r.queries.GetStudentsByAdvisorID(ctx, utils.UUIDToPgtypeNullable(advisorID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get students by advisor: %v", sharedErrors.ErrQueryFailed, err)
	}
	return students, nil
}
