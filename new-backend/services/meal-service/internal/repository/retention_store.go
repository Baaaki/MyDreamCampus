package repository

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RetentionStore lets the shared eventbus.RetentionWorker prune this
// schema's event tables without knowing its sqlc rows.
type RetentionStore struct {
	queries *db.Queries
}

func NewRetentionStore(pool *pgxpool.Pool) *RetentionStore {
	return &RetentionStore{queries: db.New(pool)}
}

var _ eventbus.RetentionStore = (*RetentionStore)(nil)

func (s *RetentionStore) DeleteRelayedOutboxEvents(ctx context.Context, before time.Time) (int64, error) {
	return s.queries.DeletePublishedOutboxEvents(ctx, utils.TimeToPgTimestamptz(before))
}

func (s *RetentionStore) DeleteProcessedEvents(ctx context.Context, before time.Time) (int64, error) {
	return s.queries.DeleteOldProcessedEvents(ctx, utils.TimeToPgTimestamptz(before))
}
