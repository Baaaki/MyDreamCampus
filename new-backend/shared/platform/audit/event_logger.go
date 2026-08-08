package audit

import (
	"context"
	"encoding/json"
	"fmt"
)

// EventAuditEntryCreated is the routing key audit entries travel under.
// Catalog binds catalog.audit_events to it on every publisher exchange.
const EventAuditEntryCreated = "audit.entry.created"

// AuditOutbox is the slice of a module's outbox the event logger needs.
// Each module implements it over its own sqlc queries.
type AuditOutbox interface {
	// QueueAuditEvent writes one audit entry to the module's outbox inside
	// the caller's transaction context.
	QueueAuditEvent(ctx context.Context, payload []byte) error
}

// EventAuditLogger publishes audit entries instead of writing them into
// catalog's table directly. An audit entry is a side effect, not something
// the request has to wait on; the outbox keeps the delivery guarantee that
// the in-process write used to provide.
type EventAuditLogger struct {
	outbox      AuditOutbox
	serviceName string
}

func NewEventAuditLogger(outbox AuditOutbox, serviceName string) *EventAuditLogger {
	return &EventAuditLogger{outbox: outbox, serviceName: serviceName}
}

// Log queues the entry. The error return is kept so callers keep treating a
// failed audit write as a warning, exactly as before.
func (l *EventAuditLogger) Log(ctx context.Context, event AuditEvent) error {
	event.Service = l.serviceName

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}
	return l.outbox.QueueAuditEvent(ctx, payload)
}

var _ Logger = (*EventAuditLogger)(nil)
