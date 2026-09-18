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

// ConsumeWithDLQ consumes a queue, retrying a failed message through the
// queue's retry delay line up to maxRetries times before parking it in the
// DLQ. The first subscription is made before it returns, so a missing queue
// still fails startup; after that it re-subscribes on its own whenever the
// connection is re-established.
func (c *Consumer) ConsumeWithDLQ(queueName string, handler MessageHandler, maxRetries int) error {
	// Taken before subscribing: a reconnect between the two must not be missed.
	next := c.conn.Reconnected()
	msgs, err := c.subscribe(queueName)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	logger.Info("consumer started with DLQ support",
		zap.String("queue", queueName),
		zap.Int("max_retries", maxRetries),
	)

	go func() {
		for {
			for msg := range msgs {
				c.handleDelivery(queueName, msg, handler, maxRetries)
			}
			// The delivery channel closes when the AMQP channel or
			// connection dies. Wait for the supervisor to bring up a new one
			// and subscribe again — returning here is how every consumer
			// used to stop for good after a broker restart.
			for {
				select {
				case <-c.conn.Done():
					return
				case <-next:
				}
				next = c.conn.Reconnected()
				msgs, err = c.subscribe(queueName)
				if err == nil {
					logger.Info("consumer resubscribed after reconnect", zap.String("queue", queueName))
					break
				}
				logger.Error("consumer resubscribe failed, waiting for the next reconnect",
					zap.Error(err), zap.String("queue", queueName))
			}
		}
	}()

	return nil
}

func (c *Consumer) subscribe(queueName string) (<-chan amqp.Delivery, error) {
	return c.conn.Channel().Consume(
		queueName,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
}

func (c *Consumer) handleDelivery(queueName string, msg amqp.Delivery, handler MessageHandler, maxRetries int) {
	retryCount := getRetryCount(msg.Headers)

	logger.Debug("message received",
		zap.String("queue", queueName),
		zap.Int("retry_count", retryCount),
	)

	if err := handler(msg.Body); err != nil {
		logger.Error("message processing failed",
			zap.Error(err),
			zap.String("queue", queueName),
			zap.Int("retry_count", retryCount),
		)

		if retryCount < maxRetries {
			if pubErr := c.scheduleRetry(queueName, msg, retryCount+1); pubErr != nil {
				// Could not park it for later: requeue now rather than lose it.
				logger.Error("failed to schedule retry, requeueing",
					zap.Error(pubErr), zap.String("queue", queueName))
				if nackErr := msg.Nack(false, true); nackErr != nil {
					logger.Warn("nack failed", zap.String("queue", queueName), zap.Error(nackErr))
				}
				return
			}
			if ackErr := msg.Ack(false); ackErr != nil {
				logger.Warn("ack failed after scheduling retry", zap.String("queue", queueName), zap.Error(ackErr))
			}
			return
		}

		logger.Warn("max retries exceeded, sending to DLQ",
			zap.String("queue", queueName),
			zap.Int("retry_count", retryCount),
		)
		if nackErr := msg.Nack(false, false); nackErr != nil { // Don't requeue - goes to DLQ
			logger.Warn("nack to DLQ failed", zap.String("queue", queueName), zap.Error(nackErr))
		}
		return
	}

	if ackErr := msg.Ack(false); ackErr != nil {
		logger.Warn("ack failed", zap.String("queue", queueName), zap.Error(ackErr))
	}
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

// getRetryCount extracts retry count from message headers. The broker may
// hand an integer header back in any width, so all of them are accepted.
func getRetryCount(headers amqp.Table) int {
	switch v := headers[retryCountHeader].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	case int16:
		return int(v)
	case int8:
		return int(v)
	}
	return 0
}

const retryCountHeader = "x-retry-count"

// scheduleRetry parks the message in the queue's retry delay line. It goes
// through the default exchange by queue name, so only this queue sees it
// again.
func (c *Consumer) scheduleRetry(queueName string, msg amqp.Delivery, retryCount int) error {
	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	headers[retryCountHeader] = utils.ClampToInt32(retryCount)

	return c.conn.Channel().Publish(
		"",                        // default exchange: routes by queue name
		RetryQueueName(queueName), // routing key = the retry queue
		false,
		false,
		amqp.Publishing{
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		},
	)
}

// UnmarshalEvent unmarshals JSON event body
func UnmarshalEvent(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return nil
}
