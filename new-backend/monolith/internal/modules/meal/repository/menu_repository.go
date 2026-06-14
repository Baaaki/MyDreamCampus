package repository

import (
	"context"
	"errors"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MenuRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// UpsertMonthlyMenu creates or updates monthly menu
func (r *MenuRepository) UpsertMonthlyMenu(ctx context.Context, params db.UpsertMonthlyMenuParams) (db.MonthlyMenu, error) {
	menu, err := r.queries.UpsertMonthlyMenu(ctx, params)
	if err != nil {
		return db.MonthlyMenu{}, fmt.Errorf("%w: failed to upsert monthly menu: %v", sharedErrors.ErrQueryFailed, err)
	}
	return menu, nil
}

// GetMonthlyMenu returns monthly menu for given year and month
func (r *MenuRepository) GetMonthlyMenu(ctx context.Context, params db.GetMonthlyMenuParams) (db.MonthlyMenu, error) {
	menu, err := r.queries.GetMonthlyMenu(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.MonthlyMenu{}, fmt.Errorf("%w", sharedErrors.ErrNotFoundRepo)
		}
		return db.MonthlyMenu{}, fmt.Errorf("%w: failed to get monthly menu: %v", sharedErrors.ErrQueryFailed, err)
	}
	return menu, nil
}
