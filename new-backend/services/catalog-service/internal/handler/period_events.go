package handler

import (
	"context"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	platformHandler "github.com/baaaki/mydreamcampus/shared/platform/handler"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/jackc/pgx/v5"
)

// PeriodEvents lets the shared period handler queue projection events on
// catalog's outbox, with the same payloads the semester wizard sends.
type PeriodEvents struct{}

var _ platformHandler.PeriodEventQueuer = PeriodEvents{}

func (PeriodEvents) QueuePeriodEvent(ctx context.Context, tx pgx.Tx, p *sharedRepo.SimplePeriod, action string) error {
	return queuePeriodEvent(ctx, db.New(tx), p, p.PeriodType, action)
}

func (PeriodEvents) QueuePeriodDeletedEvent(ctx context.Context, tx pgx.Tx, semester, periodType string) error {
	return queuePeriodDeletedEvent(ctx, db.New(tx), semester, periodType)
}
