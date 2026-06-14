package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClosedDaysRepository struct {
	queries *db.Queries
}

func NewClosedDaysRepository(pool *pgxpool.Pool) *ClosedDaysRepository {
	return &ClosedDaysRepository{
		queries: db.New(pool),
	}
}

func (r *ClosedDaysRepository) CreateClosedDay(ctx context.Context, params db.CreateClosedDayParams) (db.ClosedDay, error) {
	return r.queries.CreateClosedDay(ctx, params)
}

func (r *ClosedDaysRepository) DeleteClosedDay(ctx context.Context, id pgtype.UUID) error {
	return r.queries.DeleteClosedDay(ctx, id)
}

func (r *ClosedDaysRepository) ListClosedDays(ctx context.Context, params db.ListClosedDaysParams) ([]db.ClosedDay, error) {
	return r.queries.ListClosedDays(ctx, params)
}

func (r *ClosedDaysRepository) IsDateClosed(ctx context.Context, date pgtype.Date) (bool, error) {
	return r.queries.IsDateClosed(ctx, date)
}

func (r *ClosedDaysRepository) GetClosedDaysByDateRange(ctx context.Context, params db.GetClosedDaysByDateRangeParams) ([]db.ClosedDay, error) {
	return r.queries.GetClosedDaysByDateRange(ctx, params)
}

func (r *ClosedDaysRepository) DeleteClosedDaysBySemester(ctx context.Context, semester string) error {
	return r.queries.DeleteClosedDaysBySemester(ctx, pgtype.Text{String: semester, Valid: true})
}
