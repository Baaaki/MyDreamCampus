package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SetupDLQ declares a work queue together with its retry delay queue and its
// dead-letter exchange and queue.
func SetupDLQ(channel *amqp.Channel, queueName string) error {
	dlqName := DLQName(queueName)
	dlqExchangeName := DLQExchangeName(queueName)

	// 1. Declare DLQ exchange
	if err := channel.ExchangeDeclare(
		dlqExchangeName,
		"fanout", // fanout sends to all bound queues
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	); err != nil {
		return fmt.Errorf("failed to declare DLQ exchange: %w", err)
	}

	// 2. Declare DLQ queue
	if _, err := channel.QueueDeclare(
		dlqName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// 3. Bind DLQ to DLQ exchange
	if err := channel.QueueBind(
		dlqName,
		"",              // routing key (empty for fanout)
		dlqExchangeName, // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// 4. Declare the retry delay queue. Nothing binds it: consumers publish
	// to it by name through the default exchange.
	if _, err := channel.QueueDeclare(
		RetryQueueName(queueName),
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		RetryQueueArgs(queueName),
	); err != nil {
		return fmt.Errorf("failed to declare retry queue: %w", err)
	}

	// 5. Declare main queue with DLQ configuration
	if _, err := channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		WorkQueueArgs(queueName),
	); err != nil {
		return fmt.Errorf("failed to declare queue with DLQ: %w", err)
	}

	return nil
}
