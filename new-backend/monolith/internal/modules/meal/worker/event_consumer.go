package worker

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"go.uber.org/zap"
)

type EventConsumer struct {
	consumer        *rabbitmq.Consumer
	studentConsumer *StudentEventConsumer
	paymentConsumer *PaymentEventConsumer
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	studentConsumer *StudentEventConsumer,
	paymentConsumer *PaymentEventConsumer,
) *EventConsumer {
	return &EventConsumer{
		consumer:        consumer,
		studentConsumer: studentConsumer,
		paymentConsumer: paymentConsumer,
	}
}

func (c *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EventConsumer"))
	log.Info("starting meal event consumer")

	// Subscribe to student events
	err := c.consumer.DeclareQueue("meal.student_created_queue")
	if err != nil {
		return fmt.Errorf("failed to declare queue meal.student_created_queue: %w", err)
	}
	err = c.consumer.BindQueue("meal.student_created_queue", "student.events", "student.created")
	if err != nil {
		return fmt.Errorf("failed to bind queue meal.student_created_queue: %w", err)
	}
	err = c.consumer.Consume("meal.student_created_queue", func(body []byte) error { return c.studentConsumer.HandleStudentCreated(ctx, body) })
	if err != nil {
		return fmt.Errorf("failed to consume meal.student_created_queue: %w", err)
	}

	err = c.consumer.DeclareQueue("meal.student_updated_queue")
	if err != nil {
		return fmt.Errorf("failed to declare queue meal.student_updated_queue: %w", err)
	}
	err = c.consumer.BindQueue("meal.student_updated_queue", "student.events", "student.updated")
	if err != nil {
		return fmt.Errorf("failed to bind queue meal.student_updated_queue: %w", err)
	}
	err = c.consumer.Consume("meal.student_updated_queue", func(body []byte) error { return c.studentConsumer.HandleStudentUpdated(ctx, body) })
	if err != nil {
		return fmt.Errorf("failed to consume meal.student_updated_queue: %w", err)
	}

	err = c.consumer.DeclareQueue("meal.student_deactivated_queue")
	if err != nil {
		return fmt.Errorf("failed to declare queue meal.student_deactivated_queue: %w", err)
	}
	err = c.consumer.BindQueue("meal.student_deactivated_queue", "student.events", "student.deactivated")
	if err != nil {
		return fmt.Errorf("failed to bind queue meal.student_deactivated_queue: %w", err)
	}
	err = c.consumer.Consume("meal.student_deactivated_queue", func(body []byte) error { return c.studentConsumer.HandleStudentDeactivated(ctx, body) })
	if err != nil {
		return fmt.Errorf("failed to consume meal.student_deactivated_queue: %w", err)
	}

	// Subscribe to payment events
	err = c.consumer.DeclareQueue("meal.payment_completed_queue")
	if err != nil {
		return fmt.Errorf("failed to declare queue meal.payment_completed_queue: %w", err)
	}
	err = c.consumer.BindQueue("meal.payment_completed_queue", "payment.events", "payment.completed")
	if err != nil {
		return fmt.Errorf("failed to bind queue meal.payment_completed_queue: %w", err)
	}
	err = c.consumer.Consume("meal.payment_completed_queue", func(body []byte) error { return c.paymentConsumer.HandlePaymentCompleted(ctx, body) })
	if err != nil {
		return fmt.Errorf("failed to consume meal.payment_completed_queue: %w", err)
	}

	err = c.consumer.DeclareQueue("meal.payment_failed_queue")
	if err != nil {
		return fmt.Errorf("failed to declare queue meal.payment_failed_queue: %w", err)
	}
	err = c.consumer.BindQueue("meal.payment_failed_queue", "payment.events", "payment.failed")
	if err != nil {
		return fmt.Errorf("failed to bind queue meal.payment_failed_queue: %w", err)
	}
	err = c.consumer.Consume("meal.payment_failed_queue", func(body []byte) error { return c.paymentConsumer.HandlePaymentFailed(ctx, body) })
	if err != nil {
		return fmt.Errorf("failed to consume meal.payment_failed_queue: %w", err)
	}

	log.Info("meal event consumer started successfully")
	return nil
}
