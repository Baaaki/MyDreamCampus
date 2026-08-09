package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
)

// AuditOutbox routes audit entries through grades' own outbox instead of
// writing into catalog's table. Catalog consumes them and owns the row.
type AuditOutbox struct {
	repo *OutboxRepository
}

func NewAuditOutbox(repo *OutboxRepository) *AuditOutbox {
	return &AuditOutbox{repo: repo}
}

func (a *AuditOutbox) QueueAuditEvent(ctx context.Context, payload []byte) error {
	if _, err := a.repo.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     audit.EventAuditEntryCreated,
		RoutingKey:    audit.EventAuditEntryCreated,
		Payload:       payload,
	}); err != nil {
		return fmt.Errorf("queue audit event: %w", err)
	}
	return nil
}

var _ audit.AuditOutbox = (*AuditOutbox)(nil)
