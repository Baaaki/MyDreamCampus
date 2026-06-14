package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/grades-service/internal/dto"
	"github.com/baaaki/mydreamcampus/grades-service/internal/repository"
	"github.com/baaaki/mydreamcampus/grades-service/internal/service"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"go.uber.org/zap"
)

// FinalizeConsumer listens for grade.finalize.requested events on the
// grade.events exchange and runs AutoFinalize off the request path.
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
	log.Info("starting finalize consumer")

	queue := "grades-service-finalize"
	if err := w.consumer.DeclareQueue(queue); err != nil {
		return err
	}
	if err := w.consumer.BindQueue(queue, "grade.events", "grade.finalize.requested"); err != nil {
		return err
	}
	if err := w.consumer.Consume(queue, func(body []byte) error {
		return w.handleFinalizeRequested(ctx, body)
	}); err != nil {
		return err
	}

	log.Info("finalize consumer started")

	<-ctx.Done()
	log.Info("finalize consumer stopped")
	return nil
}

func (w *FinalizeConsumer) handleFinalizeRequested(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "FinalizeConsumer"),
		zap.String("method", "handleFinalizeRequested"),
	)

	var event dto.GradeFinalizeRequestedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error("failed to unmarshal grade.finalize.requested event", zap.Error(err))
		return nil // malformed: drop, don't requeue forever
	}

	// Idempotency: if the course is already finalized, skip. AutoFinalize
	// would otherwise fail on the UNIQUE(student_id, course_id) constraint
	// and requeue the message indefinitely.
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
		zap.String("instructor_id", event.Data.InstructorID.String()),
		zap.String("triggered_by", event.Data.TriggeredBy),
	)

	if _, err := w.gradeService.AutoFinalize(ctx, event.Data.CourseID, event.Data.InstructorID); err != nil {
		log.Error("auto-finalize failed", zap.Error(err), zap.String("course_id", event.Data.CourseID.String()))
		return err
	}

	return nil
}
