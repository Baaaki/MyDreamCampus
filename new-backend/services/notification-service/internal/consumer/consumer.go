package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/notification/internal/db"
	"github.com/baaaki/mydreamcampus/notification/internal/repository"
	"github.com/baaaki/mydreamcampus/notification/internal/service"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type Consumer struct {
	svc  *service.Service
	repo *repository.Repository
	log  *zap.Logger
}

func New(svc *service.Service, repo *repository.Repository, log *zap.Logger) *Consumer {
	return &Consumer{
		svc:  svc,
		repo: repo,
		log:  log,
	}
}

// Start subscribes through the shared consumer, like every other service:
// a failed message is retried through the queue's retry line and parked in
// the DLQ after rabbitmq.MaxDeliveryAttempts, and the subscription survives
// a broker restart. This consumer used to Nack(requeue=true) on every
// error, so with SMTP down one message cycled forever at full speed.
func (c *Consumer) Start(ctx context.Context, rc *rabbitmq.Consumer) error {
	if err := rc.ConsumeEnvelope(ctx, QueueNotificationEvents, c.handleMessage); err != nil {
		return err
	}
	c.log.Info("notification consumer started")
	return nil
}

// handleMessage returns an error to ask for a retry; nil acknowledges.
func (c *Consumer) handleMessage(ctx context.Context, body []byte) error {
	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		// Retried a few times, then parked in the DLQ for inspection.
		return fmt.Errorf("unparseable notification event: %w", err)
	}

	eventID, _ := event["event_id"].(string)
	eventType, _ := event["event_type"].(string)
	if eventID == "" || eventType == "" {
		return errors.New("notification event without event_id or event_type")
	}

	processed, err := c.repo.IsEventProcessed(ctx, eventID)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if processed {
		c.log.Info("event already processed, skipping", zap.String("event_id", eventID))
		return nil
	}

	if err := c.dispatch(ctx, eventID, eventType, event); err != nil {
		return fmt.Errorf("dispatch %s: %w", eventType, err)
	}

	if err := c.repo.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   eventID,
		EventType: eventType,
	}); err != nil {
		return fmt.Errorf("mark event processed: %w", err)
	}
	return nil
}

func (c *Consumer) logDelivery(ctx context.Context, eventID, eventType, channel, recipient, template string, status string, err error) {
	var errorText pgtype.Text
	if err != nil {
		errorText = pgtype.Text{String: err.Error(), Valid: true}
	} else {
		errorText = pgtype.Text{Valid: false}
	}

	_, _ = c.repo.CreateDeliveryLog(ctx, db.CreateDeliveryLogParams{
		EventID:   eventID,
		EventType: eventType,
		Channel:   channel,
		Recipient: recipient,
		Template:  template,
		Status:    status,
		Error:     errorText,
	})
}
