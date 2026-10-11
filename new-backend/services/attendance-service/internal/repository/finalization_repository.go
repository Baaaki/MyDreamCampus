package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxEvent is one event queued together with the write that caused it.
type OutboxEvent struct {
	EventType  string
	RoutingKey string
	Payload    []byte
}

type FinalizationRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewFinalizationRepository(pool *pgxpool.Pool) *FinalizationRepository {
	return &FinalizationRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// GetCourseFinalization reports whether the course was finalized and, if so,
// the stored record.
func (r *FinalizationRepository) GetCourseFinalization(ctx context.Context, courseID uuid.UUID) (db.CourseFinalization, bool, error) {
	row, err := r.queries.GetCourseFinalization(ctx, utils.UUIDToPgUUID(courseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return db.CourseFinalization{}, false, nil
	}
	if err != nil {
		return db.CourseFinalization{}, false, err
	}
	return row, true, nil
}

// SaveCourseFinalization records the finalization and queues its events in
// one transaction, so a course is never marked finalized with some of its
// events missing. It reports false, queuing nothing, when the course was
// already finalized — the primary key settles two concurrent calls too.
func (r *FinalizationRepository) SaveCourseFinalization(ctx context.Context, params db.InsertCourseFinalizationParams, events []OutboxEvent) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin finalization tx: %w", err)
	}
	// Rollback after a successful commit is a no-op returning ErrTxClosed.
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	inserted, err := queries.InsertCourseFinalization(ctx, params)
	if err != nil {
		return false, fmt.Errorf("insert course finalization: %w", err)
	}
	if inserted == 0 {
		return false, nil
	}

	correlationID := utils.CorrelationIDFromContext(ctx)
	for _, event := range events {
		if err := queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			CorrelationID: correlationID,
			EventType:     event.EventType,
			RoutingKey:    event.RoutingKey,
			Payload:       event.Payload,
		}); err != nil {
			return false, fmt.Errorf("queue %s: %w", event.EventType, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit finalization tx: %w", err)
	}
	return true, nil
}
