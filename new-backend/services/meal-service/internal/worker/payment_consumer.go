package worker

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	"github.com/baaaki/mydreamcampus/meal/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentEventConsumer handles payment events from RabbitMQ
type PaymentEventConsumer struct {
	reservationRepo     *repository.ReservationRepository
	processedEventsRepo *repository.ProcessedEventsRepository
	logger              *zap.Logger
}

func NewPaymentEventConsumer(
	reservationRepo *repository.ReservationRepository,
	processedEventsRepo *repository.ProcessedEventsRepository,
	logger *zap.Logger,
) *PaymentEventConsumer {
	return &PaymentEventConsumer{
		reservationRepo:     reservationRepo,
		processedEventsRepo: processedEventsRepo,
		logger:              logger,
	}
}

// HandlePaymentCompleted handles payment.completed event
func (c *PaymentEventConsumer) HandlePaymentCompleted(ctx context.Context, body []byte) error {
	var event dto.PaymentCompletedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("failed to unmarshal payment.completed event", zap.Error(err))
		return err
	}

	// Check if event already processed
	eventID, _ := uuid.Parse(event.EventID)
	processed, err := c.processedEventsRepo.IsEventProcessed(ctx, eventID)
	if err != nil {
		return err
	}

	if processed {
		c.logger.Debug("event already processed, skipping", zap.String("event_id", event.EventID))
		return nil
	}

	// Parse reference_id to determine if it's single or batch
	parsedID, isBatch, parseErr := parseReferenceID(event.Data.ReferenceID)
	if parseErr != nil {
		c.logger.Error("invalid reference_id", zap.Error(parseErr), zap.String("reference_id", event.Data.ReferenceID))
		return parseErr
	}

	if isBatch {
		// Update all reservations in batch
		err = c.reservationRepo.UpdateReservationsByBatchID(ctx, db.UpdateReservationsByBatchIDParams{
			BatchID:   pgtype.UUID{Bytes: parsedID, Valid: true},
			Status:    db.MealReservationStatusEnumConfirmed,
			ExpiresAt: pgtype.Timestamptz{Valid: false}, // Clear expires_at
		})
		if err != nil {
			c.logger.Error("failed to confirm batch reservations", zap.Error(err))
			return err
		}

		c.logger.Info("batch reservations confirmed", zap.String("batch_id", parsedID.String()))
	} else {
		// Update single reservation
		_, err = c.reservationRepo.UpdateReservationByID(ctx, db.UpdateReservationByIDParams{
			ID:        utils.UUIDToPgtype(parsedID),
			Status:    db.MealReservationStatusEnumConfirmed,
			ExpiresAt: pgtype.Timestamptz{Valid: false}, // Clear expires_at
		})
		if err != nil {
			c.logger.Error("failed to confirm reservation", zap.Error(err))
			return err
		}

		c.logger.Info("reservation confirmed", zap.String("reservation_id", parsedID.String()))
	}

	// Mark event as processed
	if err := c.processedEventsRepo.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: event.EventType,
	}); err != nil {
		return err
	}

	return nil
}

// HandlePaymentFailed handles payment.failed event
func (c *PaymentEventConsumer) HandlePaymentFailed(ctx context.Context, body []byte) error {
	var event dto.PaymentFailedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("failed to unmarshal payment.failed event", zap.Error(err))
		return err
	}

	// Check if event already processed
	eventID, _ := uuid.Parse(event.EventID)
	processed, err := c.processedEventsRepo.IsEventProcessed(ctx, eventID)
	if err != nil {
		return err
	}

	if processed {
		c.logger.Debug("event already processed, skipping", zap.String("event_id", event.EventID))
		return nil
	}

	// Parse reference_id
	parsedID, isBatch, parseErr := parseReferenceID(event.Data.ReferenceID)
	if parseErr != nil {
		c.logger.Error("invalid reference_id", zap.Error(parseErr), zap.String("reference_id", event.Data.ReferenceID))
		return parseErr
	}

	if isBatch {
		// A declined card cancels the reservation outright: "expired" is kept
		// for payments nobody completed in time.
		err = c.reservationRepo.UpdateReservationsByBatchID(ctx, db.UpdateReservationsByBatchIDParams{
			BatchID:   pgtype.UUID{Bytes: parsedID, Valid: true},
			Status:    db.MealReservationStatusEnumCancelled,
			ExpiresAt: pgtype.Timestamptz{Valid: false},
		})
		if err != nil {
			c.logger.Error("failed to cancel batch reservations", zap.Error(err))
			return err
		}

		c.logger.Info("batch reservations cancelled due to payment failure", zap.String("batch_id", parsedID.String()))
	} else {
		// Cancel single reservation
		_, err = c.reservationRepo.UpdateReservationByID(ctx, db.UpdateReservationByIDParams{
			ID:        utils.UUIDToPgtype(parsedID),
			Status:    db.MealReservationStatusEnumCancelled,
			ExpiresAt: pgtype.Timestamptz{Valid: false},
		})
		if err != nil {
			c.logger.Error("failed to cancel reservation", zap.Error(err))
			return err
		}

		c.logger.Info("reservation cancelled due to payment failure", zap.String("reservation_id", parsedID.String()))
	}

	// Mark event as processed
	if err := c.processedEventsRepo.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   utils.UUIDToPgtype(eventID),
		EventType: event.EventType,
	}); err != nil {
		return err
	}

	return nil
}
