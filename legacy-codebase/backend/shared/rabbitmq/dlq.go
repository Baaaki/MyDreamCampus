package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SetupDLQ sets up Dead Letter Queue for a given queue
func SetupDLQ(channel *amqp.Channel, queueName string) error {
	dlqName := queueName + ".dlq"
	dlqExchangeName := queueName + ".dlq.exchange"

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

	// 4. Declare main queue with DLQ configuration
	args := amqp.Table{
		"x-dead-letter-exchange": dlqExchangeName,
		// Optional: message TTL (time to live)
		// "x-message-ttl": 86400000, // 24 hours in milliseconds
	}

	if _, err := channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // arguments with DLQ config
	); err != nil {
		return fmt.Errorf("failed to declare queue with DLQ: %w", err)
	}

	return nil
}

// SetupDLQWithTTL sets up DLQ with message TTL
func SetupDLQWithTTL(channel *amqp.Channel, queueName string, ttlMs int) error {
	dlqName := queueName + ".dlq"
	dlqExchangeName := queueName + ".dlq.exchange"

	// Declare DLQ exchange
	if err := channel.ExchangeDeclare(
		dlqExchangeName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare DLQ exchange: %w", err)
	}

	// Declare DLQ queue
	if _, err := channel.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ
	if err := channel.QueueBind(
		dlqName,
		"",
		dlqExchangeName,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// Declare main queue with DLQ and TTL
	args := amqp.Table{
		"x-dead-letter-exchange": dlqExchangeName,
		"x-message-ttl":          int32(ttlMs), // Message TTL in milliseconds
	}

	if _, err := channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		args,
	); err != nil {
		return fmt.Errorf("failed to declare queue with DLQ and TTL: %w", err)
	}

	return nil
}
