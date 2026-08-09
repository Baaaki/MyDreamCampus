package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
)

// auditAggregateType labels the outbox row. meal's outbox requires an
// aggregate, but an audit entry has none of its own, so it carries a fresh
// id and its own type rather than pointing at a reservation.
const (
	auditAggregateType = "audit_entry"
	// Matches what reservation events use.
	auditOutboxMaxRetries = 5
)

// AuditOutbox routes audit entries through meal's own outbox instead of
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
		AggregateID:   utils.UUIDToPgtype(uuid.New()),
		AggregateType: auditAggregateType,
		EventType:     audit.EventAuditEntryCreated,
		Payload:       payload,
		MaxRetries:    auditOutboxMaxRetries,
	}); err != nil {
		return fmt.Errorf("queue audit event: %w", err)
	}
	return nil
}

var _ audit.AuditOutbox = (*AuditOutbox)(nil)
