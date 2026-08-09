package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// MessageHandler is a function that processes a message
type MessageHandler func(body []byte) error

// Consumer handles event consumption from RabbitMQ
type Consumer struct {
	conn *Connection
}

// NewConsumer creates a new consumer
func NewConsumer(conn *Connection) *Consumer {
	return &Consumer{
		conn: conn,
	}
}

// DeclareQueue declares a durable queue together with its dead-letter
// exchange and queue. The two are declared as a unit: a queue whose
// x-dead-letter-exchange points at a missing exchange drops rejected
// messages instead of parking them for inspection.
func (c *Consumer) DeclareQueue(queueName string) error {
	return SetupDLQ(c.conn.Channel(), queueName)
}

// BindQueue binds a queue to an exchange with routing key
func (c *Consumer) BindQueue(queueName, exchangeName, routingKey string) error {
	return c.conn.Channel().QueueBind(
		queueName,    // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,        // no-wait
		nil,          // arguments
	)
}

// Consume starts consuming messages from a queue
func (c *Consumer) Consume(queueName string, handler MessageHandler) error {
	msgs, err := c.conn.Channel().Consume(
		queueName, // queue
		"",        // consumer tag (auto-generated)
		false,     // auto-ack (manual ack for reliability)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	logger.Info("consumer started", zap.String("queue", queueName))

	// Process messages
	go func() {
		for msg := range msgs {
			logger.Debug("message received",
				zap.String("queue", queueName),
				zap.String("routing_key", msg.RoutingKey),
				zap.Int("body_size", len(msg.Body)),
			)

			// Process message
			if err := handler(msg.Body); err != nil {
				logger.Error("message processing failed",
					zap.Error(err),
					zap.String("queue", queueName),
					zap.String("routing_key", msg.RoutingKey),
				)

				// Negative acknowledgment - requeue the message
				if nackErr := msg.Nack(false, true); nackErr != nil {
					logger.Warn("nack failed", zap.String("queue", queueName), zap.Error(nackErr))
				}
				continue
			}

			// Acknowledge successful processing
			if ackErr := msg.Ack(false); ackErr != nil {
				logger.Warn("ack failed", zap.String("queue", queueName), zap.Error(ackErr))
			}
		}
	}()

	return nil
}

// ConsumeWithDLQ consumes messages with Dead Letter Queue support
func (c *Consumer) ConsumeWithDLQ(queueName string, handler MessageHandler, maxRetries int) error {
	msgs, err := c.conn.Channel().Consume(
		queueName,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	logger.Info("consumer started with DLQ support",
		zap.String("queue", queueName),
		zap.Int("max_retries", maxRetries),
	)

	go func() {
		for msg := range msgs {
			retryCount := getRetryCount(msg.Headers)

			logger.Debug("message received",
				zap.String("queue", queueName),
				zap.Int("retry_count", retryCount),
			)

			// Process message
			if err := handler(msg.Body); err != nil {
				logger.Error("message processing failed",
					zap.Error(err),
					zap.String("queue", queueName),
					zap.Int("retry_count", retryCount),
				)

				// Check retry limit
				if retryCount < maxRetries {
					// Republish with incremented retry count
					c.republishWithRetry(msg, retryCount+1)
					if ackErr := msg.Ack(false); ackErr != nil {
						logger.Warn("ack failed after republish", zap.String("queue", queueName), zap.Error(ackErr))
					}
				} else {
					// Max retries exceeded - send to DLQ
					logger.Warn("max retries exceeded, sending to DLQ",
						zap.String("queue", queueName),
						zap.Int("retry_count", retryCount),
					)
					if nackErr := msg.Nack(false, false); nackErr != nil { // Don't requeue - goes to DLQ
						logger.Warn("nack to DLQ failed", zap.String("queue", queueName), zap.Error(nackErr))
					}
				}
				continue
			}

			// Success
			if ackErr := msg.Ack(false); ackErr != nil {
				logger.Warn("ack failed", zap.String("queue", queueName), zap.Error(ackErr))
			}
		}
	}()

	return nil
}

// EnvelopeHandler processes one event body with a context that already
// carries the originating request's ID.
type EnvelopeHandler func(ctx context.Context, body []byte) error

// correlated is the sliver of the outbox envelope this layer reads. The rest
// of the envelope stays the handler's business.
type correlated struct {
	CorrelationID string `json:"correlation_id"`
}

// MaxDeliveryAttempts is how often a failing message is retried before it is
// parked in the queue's DLQ. Kept as one constant rather than a per-consumer
// knob: the number that matters is "not infinite", and a message that failed
// three times is not going to succeed on the fourth.
const MaxDeliveryAttempts = 3

// ConsumeEnvelope is the entry point every module consumer uses. On top of
// ConsumeWithDLQ it pulls correlation_id out of the envelope and puts it in
// the handler's context, so the consumer's log lines join the chain of the
// HTTP request that produced the event. Solving it here rather than in each
// consumer is what keeps the chain unbroken across nine services.
func (c *Consumer) ConsumeEnvelope(ctx context.Context, queueName string, handler EnvelopeHandler) error {
	return c.ConsumeWithDLQ(queueName, func(body []byte) error {
		var env correlated
		// A body that does not parse as an envelope is still the handler's
		// call — it owns the drop-or-retry decision for malformed messages.
		_ = json.Unmarshal(body, &env)
		// WithRequestIDValue mints an ID when correlation_id is empty, which
		// is the case for worker- and scheduler-driven events.
		return handler(logger.WithRequestIDValue(ctx, env.CorrelationID), body)
	}, MaxDeliveryAttempts)
}

// getRetryCount extracts retry count from message headers
func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	if retryCount, ok := headers["x-retry-count"].(int32); ok {
		return int(retryCount)
	}

	return 0
}

// republishWithRetry republishes a message with incremented retry count
func (c *Consumer) republishWithRetry(msg amqp.Delivery, retryCount int) {
	headers := msg.Headers
	if headers == nil {
		headers = amqp.Table{}
	}
	headers["x-retry-count"] = utils.ClampToInt32(retryCount)

	err := c.conn.Channel().Publish(
		msg.Exchange,   // exchange
		msg.RoutingKey, // routing key
		false,
		false,
		amqp.Publishing{
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		},
	)

	if err != nil {
		logger.Error("failed to republish message with retry",
			zap.Error(err),
			zap.Int("retry_count", retryCount),
		)
	}
}

// UnmarshalEvent unmarshals JSON event body
func UnmarshalEvent(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return nil
}
