// Package worker holds catalog's RabbitMQ consumers.
package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	"github.com/baaaki/mydreamcampus/catalog/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QueueAuditEvents collects audit.entry.created from every service that
// writes audit entries but does not own the table. The bindings live in
// main.go, one per publishing exchange.
const QueueAuditEvents = "catalog.audit_events"

// outboxEnvelope is the wrapper the shared OutboxWorker puts on the wire.
type outboxEnvelope struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// AuditConsumer writes audit entries other services publish into catalog's
// audit_log. Dedup is on the envelope's event_id, so a redelivery after a
// nack cannot double-write an immutable row.
type AuditConsumer struct {
	consumer  *rabbitmq.Consumer
	auditRepo *repository.AuditRepository
}

func NewAuditConsumer(consumer *rabbitmq.Consumer, auditRepo *repository.AuditRepository) *AuditConsumer {
	return &AuditConsumer{consumer: consumer, auditRepo: auditRepo}
}

func (w *AuditConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "CatalogAuditConsumer"))

	if err := w.consumer.ConsumeEnvelope(ctx, QueueAuditEvents, w.handleMessage); err != nil {
		log.Error("failed to start consuming", zap.Error(err))
		return err
	}

	log.Info("catalog audit consumer started", zap.String("queue", QueueAuditEvents))
	return nil
}

func (w *AuditConsumer) handleMessage(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "CatalogAuditConsumer"),
		zap.String("method", "handleMessage"),
	)

	var env outboxEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		log.Error("malformed envelope, dropping", zap.Error(err))
		return nil // requeueing a permanently broken message loops forever
	}

	var event audit.AuditEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err),
			zap.String("event_type", env.EventType))
		return nil
	}

	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		log.Error("envelope carried no usable event id, dropping", zap.Error(err))
		return nil
	}

	// actor_id is NOT NULL in the table. An entry without a parsable actor
	// can never be stored, so retrying it would block the queue forever.
	actorID, err := uuid.Parse(event.ActorID)
	if err != nil {
		log.Error("audit entry has no valid actor, dropping", zap.Error(err),
			zap.String("service", event.Service), zap.String("action", event.Action))
		return nil
	}

	var details []byte
	if event.Details != nil {
		if details, err = json.Marshal(event.Details); err != nil {
			log.Error("audit details are not serialisable, dropping", zap.Error(err))
			return nil
		}
	}

	params := db.InsertAuditLogFromEventParams{
		EventID:      utils.UUIDToPgtype(eventID),
		Service:      event.Service,
		ActorID:      utils.UUIDToPgtype(actorID),
		ActorRole:    event.ActorRole,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		Details:      details,
	}
	if resourceID, parseErr := uuid.Parse(event.ResourceID); parseErr == nil {
		params.ResourceID = utils.UUIDToPgtype(resourceID)
	}

	if err := w.auditRepo.InsertAuditLogFromEvent(ctx, params); err != nil {
		log.Error("failed to write audit entry", zap.Error(err),
			zap.String("service", event.Service), zap.String("action", event.Action))
		return err
	}

	log.Info("audit entry stored",
		zap.String("service", event.Service),
		zap.String("action", event.Action),
	)
	return nil
}
