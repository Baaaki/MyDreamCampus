package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OutboxEvent is a transport-shaped row that every module's outbox repository
// converts its sqlc-generated row into. Keeping a flat value type here means
// the generic OutboxWorker does not have to know about per-module db packages.
type OutboxEvent struct {
	ID         uuid.UUID
	EventType  string
	RoutingKey string
	Payload    json.RawMessage
	CreatedAt  time.Time
	RetryCount int16
	MaxRetries int16
	// CorrelationID is the request that produced the event, empty for
	// worker- and scheduler-driven ones. It rides along in the envelope so
	// the chain survives the hop through the broker.
	CorrelationID string
}

// OutboxStore is the contract every module repository implements so the worker
// can drive its outbox table without knowing the module's sqlc rows.
type OutboxStore interface {
	GetPending(ctx context.Context, limit int32) ([]OutboxEvent, error)
	GetFailed(ctx context.Context, limit int32) ([]OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMessage string) error
	Reset(ctx context.Context, id uuid.UUID) error
}
