package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *AuditRepository) InsertAuditLog(ctx context.Context, params db.InsertAuditLogParams) (db.AuditLog, error) {
	return r.queries.InsertAuditLog(ctx, params)
}

func (r *AuditRepository) ListAuditLog(ctx context.Context, params db.ListAuditLogParams) ([]db.AuditLog, error) {
	return r.queries.ListAuditLog(ctx, params)
}

func (r *AuditRepository) CountAuditLog(ctx context.Context, params db.CountAuditLogParams) (int64, error) {
	return r.queries.CountAuditLog(ctx, params)
}
