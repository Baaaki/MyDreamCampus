package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
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

// InsertAuditLogFromEvent writes an entry that arrived over RabbitMQ. The
// event id makes it a no-op on redelivery.
func (r *AuditRepository) InsertAuditLogFromEvent(ctx context.Context, params db.InsertAuditLogFromEventParams) error {
	return r.queries.InsertAuditLogFromEvent(ctx, params)
}

func (r *AuditRepository) ListAuditLog(ctx context.Context, params db.ListAuditLogParams) ([]db.AuditLog, error) {
	return r.queries.ListAuditLog(ctx, params)
}

func (r *AuditRepository) CountAuditLog(ctx context.Context, params db.CountAuditLogParams) (int64, error) {
	return r.queries.CountAuditLog(ctx, params)
}
