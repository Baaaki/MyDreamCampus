package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Publisher handles event publishing to RabbitMQ
type Publisher struct {
	conn *Connection
}

// NewPublisher creates a new publisher
func NewPublisher(conn *Connection) *Publisher {
	return &Publisher{
		conn: conn,
	}
}

// DeclareExchange declares a topic exchange
func (p *Publisher) DeclareExchange(exchangeName string) error {
	return p.conn.Channel().ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}

// Publish publishes a message to an exchange with routing key
func (p *Publisher) Publish(ctx context.Context, exchangeName, routingKey string, payload interface{}) error {
	// Serialize payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Publish message
	err = p.conn.Channel().PublishWithContext(
		ctx,
		exchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // persist to disk
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		logger.Error("failed to publish message",
			zap.Error(err),
			zap.String("exchange", exchangeName),
			zap.String("routing_key", routingKey),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logger.Debug("message published",
		zap.String("exchange", exchangeName),
		zap.String("routing_key", routingKey),
		zap.Int("body_size", len(body)),
	)

	return nil
}

// PublishWithRetry publishes a message with retry logic
func (p *Publisher) PublishWithRetry(ctx context.Context, exchangeName, routingKey string, payload interface{}, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = p.Publish(ctx, exchangeName, routingKey, payload)
		if err == nil {
			return nil
		}

		logger.Warn("publish failed, retrying",
			zap.Error(err),
			zap.Int("attempt", i+1),
			zap.Int("max_retries", maxRetries),
		)

		// Exponential backoff
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return fmt.Errorf("publish failed after %d retries: %w", maxRetries, err)
}
