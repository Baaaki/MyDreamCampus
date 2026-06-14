package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/grades-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScoreRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewScoreRepository(pool *pgxpool.Pool) *ScoreRepository {
	return &ScoreRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *ScoreRepository) UpsertAssessmentScore(ctx context.Context, arg db.UpsertAssessmentScoreParams) (db.StudentAssessmentScore, error) {
	return r.queries.UpsertAssessmentScore(ctx, arg)
}

// UpsertAssessmentScoreWithEvent atomically writes the score and the outbox
// event in a single transaction. If either write fails, nothing is committed —
// this guarantees we never persist a score without its corresponding event,
// or publish an event for a score that never landed.
func (r *ScoreRepository) UpsertAssessmentScoreWithEvent(
	ctx context.Context,
	scoreParams db.UpsertAssessmentScoreParams,
	eventParams *db.CreateOutboxEventParams,
) (db.StudentAssessmentScore, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.StudentAssessmentScore{}, fmt.Errorf("%w: failed to begin tx: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	score, err := qtx.UpsertAssessmentScore(ctx, scoreParams)
	if err != nil {
		return db.StudentAssessmentScore{}, fmt.Errorf("%w: failed to upsert score: %v", sharedErrors.ErrQueryFailed, err)
	}

	if eventParams != nil {
		if _, err := qtx.CreateOutboxEvent(ctx, *eventParams); err != nil {
			return db.StudentAssessmentScore{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return db.StudentAssessmentScore{}, fmt.Errorf("%w: failed to commit tx: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return score, nil
}

func (r *ScoreRepository) GetScoresByRegistration(ctx context.Context, registrationID uuid.UUID) ([]db.StudentAssessmentScore, error) {
	return r.queries.GetScoresByRegistration(ctx, registrationID)
}

func (r *ScoreRepository) GetLockedRegistrationsBySlug(ctx context.Context, slug string, ids []uuid.UUID) ([]uuid.UUID, error) {
	return r.queries.GetLockedRegistrationsBySlug(ctx, db.GetLockedRegistrationsBySlugParams{
		Slug:    slug,
		Column2: ids,
	})
}

// BulkUpsertEntry couples a score upsert with its outbox event so they land
// atomically inside the same transaction. EventParams may be nil when the
// entry has no numeric score to publish (absence-only upserts).
type BulkUpsertEntry struct {
	ScoreParams db.UpsertAssessmentScoreParams
	EventParams *db.CreateOutboxEventParams
}

// BulkUpsertAssessmentScoresWithEvents writes every (score, event) pair in a
// single transaction. If any write fails, the whole batch rolls back —
// callers are expected to have already filtered out invalid entries.
func (r *ScoreRepository) BulkUpsertAssessmentScoresWithEvents(ctx context.Context, entries []BulkUpsertEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to begin tx: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	count := 0
	for _, entry := range entries {
		if _, err := qtx.UpsertAssessmentScore(ctx, entry.ScoreParams); err != nil {
			return 0, fmt.Errorf("%w: failed to upsert score: %v", sharedErrors.ErrQueryFailed, err)
		}
		if entry.EventParams != nil {
			if _, err := qtx.CreateOutboxEvent(ctx, *entry.EventParams); err != nil {
				return 0, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
			}
		}
		count++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("%w: failed to commit tx: %v", sharedErrors.ErrTransactionFailed, err)
	}
	return count, nil
}

func (r *ScoreRepository) CountScoresBySlugAndCourse(ctx context.Context, arg db.CountScoresBySlugAndCourseParams) (int64, error) {
	return r.queries.CountScoresBySlugAndCourse(ctx, arg)
}

func (r *ScoreRepository) DeleteScoresByCourse(ctx context.Context, courseID uuid.UUID) error {
	return r.queries.DeleteScoresByCourse(ctx, courseID)
}

func (r *ScoreRepository) GetScoreByRegistrationAndSlug(ctx context.Context, registrationID uuid.UUID, slug string) (db.StudentAssessmentScore, error) {
	return r.queries.GetScoreByRegistrationAndSlug(ctx, db.GetScoreByRegistrationAndSlugParams{
		RegistrationID: registrationID,
		Slug:           slug,
	})
}

func (r *ScoreRepository) UnlockScore(ctx context.Context, registrationID uuid.UUID, slug string) error {
	return r.queries.UnlockScore(ctx, db.UnlockScoreParams{
		RegistrationID: registrationID,
		Slug:           slug,
	})
}

func (r *ScoreRepository) LockScore(ctx context.Context, registrationID uuid.UUID, slug string) error {
	return r.queries.LockScore(ctx, db.LockScoreParams{
		RegistrationID: registrationID,
		Slug:           slug,
	})
}

func (r *ScoreRepository) CountLockedScoresBySlugAndCourse(ctx context.Context, courseID uuid.UUID, slug string) (int64, error) {
	return r.queries.CountLockedScoresBySlugAndCourse(ctx, db.CountLockedScoresBySlugAndCourseParams{
		CourseID: courseID,
		Slug:     slug,
	})
}

func (r *ScoreRepository) LockScoresByCourseAndSlug(ctx context.Context, courseID uuid.UUID, slug string) error {
	return r.queries.LockScoresByCourseAndSlug(ctx, db.LockScoresByCourseAndSlugParams{
		CourseID: courseID,
		Slug:     slug,
	})
}
