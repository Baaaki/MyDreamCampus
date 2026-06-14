package repository

import (
	"context"
	"errors"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CafeteriaRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCafeteriaRepository(pool *pgxpool.Pool) *CafeteriaRepository {
	return &CafeteriaRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// GetActiveCafeterias returns all active cafeterias
func (r *CafeteriaRepository) GetActiveCafeterias(ctx context.Context) ([]db.Cafeteria, error) {
	cafeterias, err := r.queries.GetActiveCafeterias(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get active cafeterias: %v", sharedErrors.ErrQueryFailed, err)
	}
	return cafeterias, nil
}

// GetAllCafeterias returns all cafeterias including inactive ones
func (r *CafeteriaRepository) GetAllCafeterias(ctx context.Context) ([]db.Cafeteria, error) {
	cafeterias, err := r.queries.GetAllCafeterias(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get all cafeterias: %v", sharedErrors.ErrQueryFailed, err)
	}
	return cafeterias, nil
}

// GetCafeteriaByID returns cafeteria by ID
func (r *CafeteriaRepository) GetCafeteriaByID(ctx context.Context, id uuid.UUID) (db.Cafeteria, error) {
	cafeteria, err := r.queries.GetCafeteriaByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Cafeteria{}, fmt.Errorf("%w", serviceErrors.ErrCafeteriaNotFoundRepo)
		}
		return db.Cafeteria{}, fmt.Errorf("%w: failed to get cafeteria: %v", sharedErrors.ErrQueryFailed, err)
	}
	return cafeteria, nil
}

// CreateCafeteria creates a new cafeteria
func (r *CafeteriaRepository) CreateCafeteria(ctx context.Context, params db.CreateCafeteriaParams) (db.Cafeteria, error) {
	cafeteria, err := r.queries.CreateCafeteria(ctx, params)
	if err != nil {
		return db.Cafeteria{}, fmt.Errorf("%w: failed to create cafeteria: %v", sharedErrors.ErrQueryFailed, err)
	}
	return cafeteria, nil
}

// UpdateCafeteria updates a cafeteria
func (r *CafeteriaRepository) UpdateCafeteria(ctx context.Context, params db.UpdateCafeteriaParams) (db.Cafeteria, error) {
	cafeteria, err := r.queries.UpdateCafeteria(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Cafeteria{}, fmt.Errorf("%w", serviceErrors.ErrCafeteriaNotFoundRepo)
		}
		return db.Cafeteria{}, fmt.Errorf("%w: failed to update cafeteria: %v", sharedErrors.ErrQueryFailed, err)
	}
	return cafeteria, nil
}

// DeactivateCafeteria soft deletes a cafeteria
func (r *CafeteriaRepository) DeactivateCafeteria(ctx context.Context, id uuid.UUID) error {
	_, err := r.queries.DeactivateCafeteria(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w", serviceErrors.ErrCafeteriaNotFoundRepo)
		}
		return fmt.Errorf("%w: failed to deactivate cafeteria: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}
