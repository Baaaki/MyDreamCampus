package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QueuePeriodEvents carries catalog's academic period definitions. The binding
// (main.go) filters on course_catalog.period.attendance.*, so only this
// module's period type ever lands here.
const QueuePeriodEvents = "attendance.period_events"

// periodEventData is catalog's period payload as it sits under the outbox
// envelope's "data" key. Deleted events carry only semester and period_type.
type periodEventData struct {
	ID          uuid.UUID `json:"id"`
	Semester    string    `json:"semester"`
	PeriodType  string    `json:"period_type"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	IsActive    bool      `json:"is_active"`
}

// PeriodConsumer keeps attendance's local academic_periods projection in sync
// with catalog. Unlike the cache consumer this needs no processed_events
// dedup: the upsert keys on semester and the delete is by semester, so
// redelivery after a nack cannot corrupt state.
type PeriodConsumer struct {
	consumer   *rabbitmq.Consumer
	periodRepo *platformRepo.SimplePeriodRepository
}

func NewPeriodConsumer(
	consumer *rabbitmq.Consumer,
	periodRepo *platformRepo.SimplePeriodRepository,
) *PeriodConsumer {
	return &PeriodConsumer{
		consumer:   consumer,
		periodRepo: periodRepo,
	}
}

func (w *PeriodConsumer) Start(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "AttendancePeriodConsumer"))

	if err := w.consumer.ConsumeEnvelope(ctx, QueuePeriodEvents, w.handleMessage); err != nil {
		log.Error("failed to start consuming", zap.Error(err))
		return err
	}

	log.Info("attendance period consumer started", zap.String("queue", QueuePeriodEvents))
	return nil
}

func (w *PeriodConsumer) handleMessage(ctx context.Context, body []byte) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "AttendancePeriodConsumer"),
		zap.String("method", "handleMessage"),
	)

	var base dto.BaseEvent
	if err := json.Unmarshal(body, &base); err != nil {
		log.Error("malformed envelope, dropping", zap.Error(err))
		return nil // requeueing a permanently broken message loops forever
	}

	event, err := unwrapEventData[periodEventData](body)
	if err != nil {
		log.Error("payload did not match contract, dropping", zap.Error(err),
			zap.String("event_type", base.EventType))
		return nil
	}

	// The broker already filtered on our period type, so only the action matters.
	switch events.PeriodEventAction(base.EventType) {
	case events.PeriodActionCreated, events.PeriodActionUpdated:
		if err := w.periodRepo.UpsertPeriod(ctx, platformRepo.SimplePeriod{
			ID:          event.ID,
			Semester:    event.Semester,
			PeriodStart: event.PeriodStart,
			PeriodEnd:   event.PeriodEnd,
			IsActive:    event.IsActive,
		}); err != nil {
			log.Error("failed to upsert academic period", zap.Error(err),
				zap.String("semester", event.Semester))
			return err
		}
		log.Info("academic period projected", zap.String("semester", event.Semester))
		return nil

	case events.PeriodActionDeleted:
		if err := w.periodRepo.DeletePeriodBySemester(ctx, event.Semester); err != nil {
			log.Error("failed to delete academic period", zap.Error(err),
				zap.String("semester", event.Semester))
			return err
		}
		log.Info("academic period removed", zap.String("semester", event.Semester))
		return nil

	default:
		log.Warn("unknown event type, dropping", zap.String("event_type", base.EventType))
		return nil
	}
}
