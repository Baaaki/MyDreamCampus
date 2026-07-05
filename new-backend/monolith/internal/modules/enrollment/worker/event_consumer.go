package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"go.uber.org/zap"
)

// QueueSyncEvents feeds enrollment's passed-prerequisite projection.
// Bindings are pre-declared in main.go (DeclareDownstreamBindings) so
// events published while this consumer is down stay queued.
const QueueSyncEvents = "enrollment.sync_events"

// outboxEnvelope is the wrapper the shared OutboxWorker puts on the wire:
// {event_id, event_type, timestamp, data}. Data stays raw because each
// producer ships a differently-shaped payload under it.
type outboxEnvelope struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// EventConsumer keeps enrollment's student_passed_prerequisites projection
// in sync. No processed_events dedup here — the upsert is naturally
// idempotent, so redelivery after a nack cannot corrupt state.
type EventConsumer struct {
	consumer         *rabbitmq.Consumer
	passedPrereqRepo *repository.PassedPrerequisitesRepository
}

func NewEventConsumer(
	consumer *rabbitmq.Consumer,
	passedPrereqRepo *repository.PassedPrerequisitesRepository,
) *EventConsumer {
	return &EventConsumer{
		consumer:         consumer,
		passedPrereqRepo: passedPrereqRepo,
	}
}

func (w *EventConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "EnrollmentEventConsumer"))

	if err := w.consumer.Consume(QueueSyncEvents, func(body []byte) error {
		return w.handleMessage(ctx, body)
	}); err != nil {
		log.Error("failed to start consuming", zap.Error(err))
		return err
	}

	log.Info("enrollment event consumer started", zap.String("queue", QueueSyncEvents))
	return nil
}

func (w *EventConsumer) handleMessage(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EnrollmentEventConsumer"),
		zap.String("method", "handleMessage"),
	)

	var env outboxEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		log.Error("malformed envelope, dropping", zap.Error(err))
		return nil // requeueing a permanently broken message loops forever
	}

	switch env.EventType {
	case "grade.student.prerequisite.passed":
		return w.handlePrerequisitePassed(ctx, env)
	default:
		log.Warn("unknown event type, dropping", zap.String("event_type", env.EventType))
		return nil
	}
}

func (w *EventConsumer) handlePrerequisitePassed(ctx context.Context, env outboxEnvelope) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "EnrollmentEventConsumer"),
		zap.String("method", "handlePrerequisitePassed"),
	)

	// grades' outbox payload is BaseEvent-shaped, so env.Data is
	// {event_type, ..., data: {student_id, course_id, ...}}.
	var event dto.GradeStudentPrerequisitePassedEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	if err := w.passedPrereqRepo.UpsertPassedPrerequisite(ctx,
		event.Data.StudentID,
		event.Data.CourseID,
		event.Data.CourseCode,
		event.Data.Semester,
		event.Data.GradePoint,
	); err != nil {
		log.Error("failed to upsert passed prerequisite", zap.Error(err),
			zap.String("student_id", event.Data.StudentID.String()),
			zap.String("course_code", event.Data.CourseCode),
		)
		return err
	}

	log.Info("passed prerequisite synced",
		zap.String("student_id", event.Data.StudentID.String()),
		zap.String("course_code", event.Data.CourseCode),
	)
	return nil
}
