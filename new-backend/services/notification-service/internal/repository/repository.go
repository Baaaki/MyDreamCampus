package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/notification/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *Repository) CreateDeliveryLog(ctx context.Context, params db.CreateDeliveryLogParams) (db.DeliveryLog, error) {
	return r.queries.CreateDeliveryLog(ctx, params)
}

func (r *Repository) UpdateDeliveryLogStatus(ctx context.Context, params db.UpdateDeliveryLogStatusParams) error {
	return r.queries.UpdateDeliveryLogStatus(ctx, params)
}

func (r *Repository) MarkEventProcessed(ctx context.Context, params db.MarkEventProcessedParams) error {
	return r.queries.MarkEventProcessed(ctx, params)
}

func (r *Repository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	return r.queries.IsEventProcessed(ctx, eventID)
}
