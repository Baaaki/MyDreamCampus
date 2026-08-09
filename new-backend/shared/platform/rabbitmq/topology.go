package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

// DLQName and DLQExchangeName derive a queue's dead-letter names. Every
// caller must go through these: a queue and its DLQ are matched by name
// convention only, so a second spelling silently orphans poison messages.
func DLQName(queueName string) string { return queueName + ".dlq" }

func DLQExchangeName(queueName string) string { return queueName + ".dlq.exchange" }

// WorkQueueArgs are the arguments every consumer queue is declared with.
//
// RabbitMQ rejects a re-declare whose arguments differ from the existing
// queue (PRECONDITION_FAILED, which also kills the channel). Publisher-side
// pre-declare, consumer-side declare and definitions.json therefore all have
// to pass exactly this table — otherwise whichever one runs second fails.
func WorkQueueArgs(queueName string) amqp.Table {
	return amqp.Table{"x-dead-letter-exchange": DLQExchangeName(queueName)}
}
