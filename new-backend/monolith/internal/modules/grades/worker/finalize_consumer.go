package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"go.uber.org/zap"
)

// QueueFinalizeRequested is the self-loop queue: LockAssessment/LockScore
// emit grade.finalize.requested to the outbox, and this consumer runs the
// heavy AutoFinalize computation off the request path.
const QueueFinalizeRequested = "grades.finalize_requested"

type FinalizeConsumer struct {
	consumer      *rabbitmq.Consumer
	gradeService  *service.GradeService
	completedRepo *repository.CompletedRepository
}

func NewFinalizeConsumer(
	consumer *rabbitmq.Consumer,
	gradeService *service.GradeService,
	completedRepo *repository.CompletedRepository,
) *FinalizeConsumer {
	return &FinalizeConsumer{
		consumer:      consumer,
		gradeService:  gradeService,
		completedRepo: completedRepo,
	}
}

func (w *FinalizeConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "FinalizeConsumer"))

	if err := w.consumer.Consume(QueueFinalizeRequested, func(body []byte) error {
		return w.handleFinalizeRequested(ctx, body)
	}); err != nil {
		log.Error("failed to start consuming", zap.Error(err))
		return err
	}

	log.Info("finalize consumer started", zap.String("queue", QueueFinalizeRequested))
	return nil
}

func (w *FinalizeConsumer) handleFinalizeRequested(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "FinalizeConsumer"),
		zap.String("method", "handleFinalizeRequested"),
	)

	var env outboxEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		log.Error("malformed envelope, dropping", zap.Error(err))
		return nil
	}

	var event dto.GradeFinalizeRequestedEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err))
		return nil
	}

	// Idempotency: a redelivered or duplicate finalize request for an
	// already-finalized course must not run again — AutoFinalize would
	// find no registrations (they are deleted on finalize) and error,
	// requeueing the message forever.
	existing, err := w.completedRepo.GetCompletedCoursesByCourse(ctx, event.Data.CourseID)
	if err != nil {
		log.Error("failed to check existing completed courses", zap.Error(err))
		return err
	}
	if len(existing) > 0 {
		log.Info("course already finalized, skipping",
			zap.String("course_id", event.Data.CourseID.String()),
			zap.String("triggered_by", event.Data.TriggeredBy),
		)
		return nil
	}

	log.Info("running auto-finalize from event",
		zap.String("course_id", event.Data.CourseID.String()),
		zap.String("triggered_by", event.Data.TriggeredBy),
	)

	if _, err := w.gradeService.AutoFinalize(ctx, event.Data.CourseID, event.Data.InstructorID); err != nil {
		log.Error("auto-finalize failed", zap.Error(err),
			zap.String("course_id", event.Data.CourseID.String()))
		return err
	}

	return nil
}
