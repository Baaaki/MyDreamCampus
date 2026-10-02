package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ErrNotConfirmed means the broker refused a message (basic.nack), the
// channel closed before it answered, or it did not answer in time. The
// message has to be treated as never sent.
var ErrNotConfirmed = errors.New("rabbitmq: publish not confirmed by the broker")

// confirmTimeout bounds the wait for the broker's answer to one publish.
const confirmTimeout = 5 * time.Second

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

// Publish publishes a message to an exchange with routing key and returns
// once the broker has confirmed it. A nil error is the broker taking
// responsibility for the message, the one moment the outbox relay may mark
// an event sent.
func (p *Publisher) Publish(ctx context.Context, exchangeName, routingKey string, payload any) error {
	// Serialize payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = publishConfirmed(ctx, p.conn.Channel(), exchangeName, routingKey, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // persist to disk
		Timestamp:    time.Now(),      // transport metadata, not a business time
	})
	if err != nil {
		logger.Error("failed to publish message",
			zap.Error(err),
			zap.String("exchange", exchangeName),
			zap.String("routing_key", routingKey),
		)
		return err
	}

	logger.Debug("message published",
		zap.String("exchange", exchangeName),
		zap.String("routing_key", routingKey),
		zap.Int("body_size", len(body)),
	)

	return nil
}

// publishConfirmed publishes on a channel in confirm mode and waits for the
// broker's ack, at most confirmTimeout.
func publishConfirmed(ctx context.Context, ch *amqp.Channel, exchange, routingKey string, msg amqp.Publishing) error {
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, exchange, routingKey, false, false, msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	if confirmation == nil {
		// Only a channel that never entered confirm mode hands back none.
		return fmt.Errorf("%w: channel is not in confirm mode", ErrNotConfirmed)
	}

	waitCtx, cancel := context.WithTimeout(ctx, confirmTimeout)
	defer cancel()
	acked, err := confirmation.WaitContext(waitCtx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotConfirmed, err)
	}
	if !acked {
		return ErrNotConfirmed
	}
	return nil
}

// DeclareAndBindQueue declares a durable queue and binds it to an exchange with routing key.
// This ensures the queue exists before publishing, so messages are not lost when the consumer is offline.
func (p *Publisher) DeclareAndBindQueue(queueName, exchangeName, routingKey string) error {
	ch := p.conn.Channel()

	// Declares the queue and its dead-letter side in one go — see SetupDLQ.
	if err := SetupDLQ(ch, queueName); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	if err := ch.QueueBind(queueName, routingKey, exchangeName, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue %s to %s with key %s: %w", queueName, exchangeName, routingKey, err)
	}

	return nil
}

// PublishWithRetry publishes a message with retry logic
func (p *Publisher) PublishWithRetry(ctx context.Context, exchangeName, routingKey string, payload any, maxRetries int) error {
	var err error
	for i := range maxRetries {
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
