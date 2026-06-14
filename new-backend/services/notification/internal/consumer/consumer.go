package consumer

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/notification/internal/db"
	"github.com/baaaki/mydreamcampus/notification/internal/repository"
	"github.com/baaaki/mydreamcampus/notification/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Consumer struct {
	ch   *amqp.Channel
	svc  *service.Service
	repo *repository.Repository
	log  *zap.Logger
}

func New(ch *amqp.Channel, svc *service.Service, repo *repository.Repository, log *zap.Logger) *Consumer {
	return &Consumer{
		ch:   ch,
		svc:  svc,
		repo: repo,
		log:  log,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	msgs, err := c.ch.Consume(
		QueueNotificationEvents, // queue
		"",                      // consumer
		false,                   // auto-ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	if err != nil {
		c.log.Fatal("failed to register consumer", zap.Error(err))
	}

	c.log.Info("notification consumer started")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("shutting down consumer")
			return
		case msg, ok := <-msgs:
			if !ok {
				c.log.Info("channel closed, stopping consumer")
				return
			}
			c.handleMessage(ctx, msg)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	// Extract message body and parse event
	var event map[string]any
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.log.Error("failed to unmarshal message", zap.Error(err))
		msg.Reject(false) // Drop unparseable messages
		return
	}

	eventID, _ := event["event_id"].(string)
	eventType, _ := event["event_type"].(string)

	if eventID == "" || eventType == "" {
		c.log.Error("missing event_id or event_type")
		msg.Reject(false)
		return
	}

	// Idempotency check
	processed, err := c.repo.IsEventProcessed(ctx, eventID)
	if err != nil {
		c.log.Error("failed to check idempotency", zap.Error(err))
		msg.Nack(false, true) // Requeue on DB error
		return
	}
	if processed {
		c.log.Info("event already processed, skipping", zap.String("event_id", eventID))
		msg.Ack(false)
		return
	}

	// Dispatch to handler
	if err := c.dispatch(ctx, eventID, eventType, event); err != nil {
		c.log.Error("failed to process event", zap.Error(err), zap.String("event_id", eventID))
		// For now, simple requeue. In production, we'd use DLQ with x-death header to count retries.
		msg.Nack(false, true)
		return
	}

	// Mark as processed
	err = c.repo.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   eventID,
		EventType: eventType,
	})
	if err != nil {
		c.log.Error("failed to mark event processed", zap.Error(err))
		msg.Nack(false, true) // Requeue
		return
	}

	msg.Ack(false)
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
