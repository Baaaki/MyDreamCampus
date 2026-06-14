import os

dir_path = "/home/nautilus/Desktop/Playground/mydreamcampus/new-backend/monolith/internal/modules/meal"

event_consumer_content = """package worker

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
	if err != nil { return fmt.Errorf("failed to declare queue meal.student_created_queue: %w", err) }
	err = c.consumer.BindQueue("meal.student_created_queue", "student.events", "student.created")
	if err != nil { return fmt.Errorf("failed to bind queue meal.student_created_queue: %w", err) }
	err = c.consumer.Consume("meal.student_created_queue", func(body []byte) error { return c.studentConsumer.HandleStudentCreated(ctx, body) })
	if err != nil { return fmt.Errorf("failed to consume meal.student_created_queue: %w", err) }

	err = c.consumer.DeclareQueue("meal.student_updated_queue")
	if err != nil { return fmt.Errorf("failed to declare queue meal.student_updated_queue: %w", err) }
	err = c.consumer.BindQueue("meal.student_updated_queue", "student.events", "student.updated")
	if err != nil { return fmt.Errorf("failed to bind queue meal.student_updated_queue: %w", err) }
	err = c.consumer.Consume("meal.student_updated_queue", func(body []byte) error { return c.studentConsumer.HandleStudentUpdated(ctx, body) })
	if err != nil { return fmt.Errorf("failed to consume meal.student_updated_queue: %w", err) }

	err = c.consumer.DeclareQueue("meal.student_deactivated_queue")
	if err != nil { return fmt.Errorf("failed to declare queue meal.student_deactivated_queue: %w", err) }
	err = c.consumer.BindQueue("meal.student_deactivated_queue", "student.events", "student.deactivated")
	if err != nil { return fmt.Errorf("failed to bind queue meal.student_deactivated_queue: %w", err) }
	err = c.consumer.Consume("meal.student_deactivated_queue", func(body []byte) error { return c.studentConsumer.HandleStudentDeactivated(ctx, body) })
	if err != nil { return fmt.Errorf("failed to consume meal.student_deactivated_queue: %w", err) }

	// Subscribe to payment events
	err = c.consumer.DeclareQueue("meal.payment_completed_queue")
	if err != nil { return fmt.Errorf("failed to declare queue meal.payment_completed_queue: %w", err) }
	err = c.consumer.BindQueue("meal.payment_completed_queue", "payment.events", "payment.completed")
	if err != nil { return fmt.Errorf("failed to bind queue meal.payment_completed_queue: %w", err) }
	err = c.consumer.Consume("meal.payment_completed_queue", func(body []byte) error { return c.paymentConsumer.HandlePaymentCompleted(ctx, body) })
	if err != nil { return fmt.Errorf("failed to consume meal.payment_completed_queue: %w", err) }

	err = c.consumer.DeclareQueue("meal.payment_failed_queue")
	if err != nil { return fmt.Errorf("failed to declare queue meal.payment_failed_queue: %w", err) }
	err = c.consumer.BindQueue("meal.payment_failed_queue", "payment.events", "payment.failed")
	if err != nil { return fmt.Errorf("failed to bind queue meal.payment_failed_queue: %w", err) }
	err = c.consumer.Consume("meal.payment_failed_queue", func(body []byte) error { return c.paymentConsumer.HandlePaymentFailed(ctx, body) })
	if err != nil { return fmt.Errorf("failed to consume meal.payment_failed_queue: %w", err) }

	log.Info("meal event consumer started successfully")
	return nil
}
"""
with open(os.path.join(dir_path, "worker", "event_consumer.go"), "w") as f:
    f.write(event_consumer_content)


module_path = os.path.join(dir_path, "module.go")
with open(module_path, "r") as f:
    mod_content = f.read()

# Replace struct fields
mod_content = mod_content.replace(
	"m.studentConsumer   *worker.StudentEventConsumer\n\tm.paymentConsumer   *worker.PaymentEventConsumer",
	"m.eventConsumer   *worker.EventConsumer"
).replace(
    "	studentConsumer   *worker.StudentEventConsumer\n	paymentConsumer   *worker.PaymentEventConsumer",
    "	eventConsumer     *worker.EventConsumer"
)

# Replace Workers section in Bootstrap
old_workers = """	// Workers
	m.studentConsumer = worker.NewStudentEventConsumer(studentCacheRepo, processedEventsRepo, m.logger)
	m.paymentConsumer = worker.NewPaymentEventConsumer(reservationRepo, processedEventsRepo, m.logger)
	m.reservationWorker = worker.NewReservationWorker(reservationRepo, m.logger)

	// Start reservation worker jobs
	m.reservationWorker.Start(ctx)

	consumer := rabbitmq.NewConsumer(m.rabbitConn)

	// Subscribe to student events
	err := consumer.DeclareQueue("meal.student_created_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.student_created_queue", "student.events", "student.created")
	if err != nil { return err }
	err = consumer.Consume("meal.student_created_queue", func(body []byte) error { return m.studentConsumer.HandleStudentCreated(ctx, body) })
	if err != nil {
		return err
	}
	err = consumer.DeclareQueue("meal.student_updated_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.student_updated_queue", "student.events", "student.updated")
	if err != nil { return err }
	err = consumer.Consume("meal.student_updated_queue", func(body []byte) error { return m.studentConsumer.HandleStudentUpdated(ctx, body) })
	if err != nil {
		return err
	}
	err = consumer.DeclareQueue("meal.student_deactivated_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.student_deactivated_queue", "student.events", "student.deactivated")
	if err != nil { return err }
	err = consumer.Consume("meal.student_deactivated_queue", func(body []byte) error { return m.studentConsumer.HandleStudentDeactivated(ctx, body) })
	if err != nil {
		return err
	}

	// Subscribe to payment events
	err = consumer.DeclareQueue("meal.payment_completed_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.payment_completed_queue", "payment.events", "payment.completed")
	if err != nil { return err }
	err = consumer.Consume("meal.payment_completed_queue", func(body []byte) error { return m.paymentConsumer.HandlePaymentCompleted(ctx, body) })
	if err != nil {
		return err
	}
	err = consumer.DeclareQueue("meal.payment_failed_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.payment_failed_queue", "payment.events", "payment.failed")
	if err != nil { return err }
	err = consumer.Consume("meal.payment_failed_queue", func(body []byte) error { return m.paymentConsumer.HandlePaymentFailed(ctx, body) })
	if err != nil {
		return err
	}"""

new_workers = """	// Workers
	studentConsumer := worker.NewStudentEventConsumer(studentCacheRepo, processedEventsRepo, m.logger)
	paymentConsumer := worker.NewPaymentEventConsumer(reservationRepo, processedEventsRepo, m.logger)
	m.reservationWorker = worker.NewReservationWorker(reservationRepo, m.logger)
	
	consumer := rabbitmq.NewConsumer(m.rabbitConn)
	m.eventConsumer = worker.NewEventConsumer(consumer, studentConsumer, paymentConsumer)

	// Start reservation worker jobs
	m.reservationWorker.Start(ctx)

	// Start event consumer
	if err := m.eventConsumer.Start(ctx); err != nil {
		return err
	}"""

mod_content = mod_content.replace(old_workers, new_workers)

with open(module_path, "w") as f:
    f.write(mod_content)

print("Meal module refactored")
