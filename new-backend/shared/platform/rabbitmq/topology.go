package rabbitmq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DLQName and DLQExchangeName derive a queue's dead-letter names. Every
// caller must go through these: a queue and its DLQ are matched by name
// convention only, so a second spelling silently orphans poison messages.
func DLQName(queueName string) string { return queueName + ".dlq" }

func DLQExchangeName(queueName string) string { return queueName + ".dlq.exchange" }

// RetryQueueName is where a failed delivery waits before its next attempt.
func RetryQueueName(queueName string) string { return queueName + ".retry" }

// RetryDelay is how long a failed message waits before it is redelivered.
// Long enough for a restarting dependency (DB, SMTP) to come back, short
// enough that a transient failure is invisible to users.
const RetryDelay = 5 * time.Second

// WorkQueueArgs are the arguments every consumer queue is declared with.
//
// RabbitMQ rejects a re-declare whose arguments differ from the existing
// queue (PRECONDITION_FAILED, which also kills the channel). Publisher-side
// pre-declare, consumer-side declare and definitions.json therefore all have
// to pass exactly this table — otherwise whichever one runs second fails.
func WorkQueueArgs(queueName string) amqp.Table {
	return amqp.Table{"x-dead-letter-exchange": DLQExchangeName(queueName)}
}

// RetryQueueArgs make the retry queue a delay line back into ONE queue: a
// message expires after RetryDelay and is dead-lettered through the default
// exchange straight to queueName. Republishing to the original exchange
// instead — as this used to — fanned every retry out to every queue bound
// to that routing key, so one service's failure re-ran other services'
// handlers. The same exact-match rule as WorkQueueArgs applies here — and
// RabbitMQ compares argument TYPES too, so the TTL is int64 (AMQP "long"),
// the type a definitions.json import would produce.
//
// Retry queues are deliberately not in definitions.json: that file exists so
// messages published before a consumer's first boot are kept, and only the
// consumer itself ever publishes to its retry queue.
func RetryQueueArgs(queueName string) amqp.Table {
	return amqp.Table{
		"x-message-ttl":             int64(RetryDelay / time.Millisecond),
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": queueName,
	}
}
