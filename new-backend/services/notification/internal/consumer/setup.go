package consumer

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeAuth   = "auth.events"
	QueueNotificationEvents = "notification_events_queue"
)

func SetupTopology(ch *amqp.Channel) error {
	// Declare queue
	_, err := ch.QueueDeclare(
		QueueNotificationEvents,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Declare exchange if not already created by publisher
	err = ch.ExchangeDeclare(
		ExchangeAuth,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Bind queue to auth.events for user.registered
	err = ch.QueueBind(
		QueueNotificationEvents,
		"user.registered",
		ExchangeAuth,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to user.registered: %w", err)
	}

	// Bind queue to auth.events for user.password_reset_requested
	err = ch.QueueBind(
		QueueNotificationEvents,
		"user.password_reset_requested",
		ExchangeAuth,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to user.password_reset_requested: %w", err)
	}

	return nil
}
