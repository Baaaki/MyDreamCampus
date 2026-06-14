package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReservationRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewReservationRepository(pool *pgxpool.Pool) *ReservationRepository {
	return &ReservationRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateReservation creates a new reservation
func (r *ReservationRepository) CreateReservation(ctx context.Context, params db.CreateReservationParams) (db.Reservation, error) {
	reservation, err := r.queries.CreateReservation(ctx, params)
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to create reservation: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservation, nil
}

// CreateReservationWithEvent creates reservation and outbox event atomically
func (r *ReservationRepository) CreateReservationWithEvent(ctx context.Context, reservationParams db.CreateReservationParams, eventParams db.CreateOutboxEventParams) (db.Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create reservation
	reservation, err := qtx.CreateReservation(ctx, reservationParams)
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to create reservation: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	_, err = qtx.CreateOutboxEvent(ctx, eventParams)
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return reservation, nil
}

// CreateBatchReservations creates multiple reservations atomically
func (r *ReservationRepository) CreateBatchReservations(ctx context.Context, reservations []db.CreateReservationParams) ([]db.Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	createdReservations := make([]db.Reservation, 0, len(reservations))
	for _, params := range reservations {
		reservation, err := qtx.CreateReservation(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to create reservation in batch: %v", sharedErrors.ErrQueryFailed, err)
		}
		createdReservations = append(createdReservations, reservation)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return createdReservations, nil
}

// GetReservationByID returns reservation by ID with cafeteria info
func (r *ReservationRepository) GetReservationByID(ctx context.Context, id uuid.UUID) (db.GetReservationByIDRow, error) {
	reservation, err := r.queries.GetReservationByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetReservationByIDRow{}, fmt.Errorf("%w", serviceErrors.ErrReservationNotFoundRepo)
		}
		return db.GetReservationByIDRow{}, fmt.Errorf("%w: failed to get reservation: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservation, nil
}

// CheckActiveReservation checks if student has active reservation for given date and meal time
func (r *ReservationRepository) CheckActiveReservation(ctx context.Context, params db.CheckActiveReservationParams) (*db.CheckActiveReservationRow, error) {
	reservation, err := r.queries.CheckActiveReservation(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No active reservation found
		}
		return nil, fmt.Errorf("%w: failed to check active reservation: %v", sharedErrors.ErrQueryFailed, err)
	}
	return &reservation, nil
}

// CheckActiveReservationsForSlots resolves the entire batch in one query.
func (r *ReservationRepository) CheckActiveReservationsForSlots(ctx context.Context, params db.CheckActiveReservationsForSlotsParams) ([]db.CheckActiveReservationsForSlotsRow, error) {
	rows, err := r.queries.CheckActiveReservationsForSlots(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check active reservations for slots: %v", sharedErrors.ErrQueryFailed, err)
	}
	return rows, nil
}

// GetStudentReservations returns all reservations for a student
func (r *ReservationRepository) GetStudentReservations(ctx context.Context, studentID uuid.UUID) ([]db.GetStudentReservationsRow, error) {
	reservations, err := r.queries.GetStudentReservations(ctx, utils.UUIDToPgtype(studentID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get student reservations: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservations, nil
}

// GetStudentReservationsFiltered returns filtered reservations for a student
func (r *ReservationRepository) GetStudentReservationsFiltered(ctx context.Context, params db.GetStudentReservationsFilteredParams) ([]db.GetStudentReservationsFilteredRow, error) {
	reservations, err := r.queries.GetStudentReservationsFiltered(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get filtered student reservations: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservations, nil
}

// CountStudentReservationsFiltered returns the total count of filtered reservations for a student
func (r *ReservationRepository) CountStudentReservationsFiltered(ctx context.Context, params db.CountStudentReservationsFilteredParams) (int64, error) {
	count, err := r.queries.CountStudentReservationsFiltered(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count student reservations: %v", sharedErrors.ErrQueryFailed, err)
	}
	return count, nil
}

// UpdateReservationByID updates reservation status and expires_at
func (r *ReservationRepository) UpdateReservationByID(ctx context.Context, params db.UpdateReservationByIDParams) (db.Reservation, error) {
	reservation, err := r.queries.UpdateReservationByID(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Reservation{}, fmt.Errorf("%w", serviceErrors.ErrReservationNotFoundRepo)
		}
		return db.Reservation{}, fmt.Errorf("%w: failed to update reservation: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservation, nil
}

// UpdateReservationsByBatchID updates all reservations in a batch
func (r *ReservationRepository) UpdateReservationsByBatchID(ctx context.Context, params db.UpdateReservationsByBatchIDParams) error {
	err := r.queries.UpdateReservationsByBatchID(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: failed to update reservations by batch ID: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// MarkReservationUsed marks a reservation as used
func (r *ReservationRepository) MarkReservationUsed(ctx context.Context, id uuid.UUID) (db.Reservation, error) {
	reservation, err := r.queries.MarkReservationUsed(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Reservation{}, fmt.Errorf("%w", serviceErrors.ErrReservationNotFoundRepo)
		}
		return db.Reservation{}, fmt.Errorf("%w: failed to mark reservation as used: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservation, nil
}

// FindReservationForQR finds reservation for QR validation
func (r *ReservationRepository) FindReservationForQR(ctx context.Context, params db.FindReservationForQRParams) (db.FindReservationForQRRow, error) {
	reservation, err := r.queries.FindReservationForQR(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.FindReservationForQRRow{}, fmt.Errorf("%w", serviceErrors.ErrReservationNotFoundRepo)
		}
		return db.FindReservationForQRRow{}, fmt.Errorf("%w: failed to find reservation for QR: %v", sharedErrors.ErrQueryFailed, err)
	}
	return reservation, nil
}

// CancelReservationWithRefund cancels reservation and creates outbox event atomically
func (r *ReservationRepository) CancelReservationWithRefund(ctx context.Context, reservationID uuid.UUID, eventPayload map[string]any) (db.Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Cancel reservation
	reservation, err := qtx.CancelReservation(ctx, utils.UUIDToPgtype(reservationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Reservation{}, fmt.Errorf("%w", serviceErrors.ErrReservationNotFoundRepo)
		}
		return db.Reservation{}, fmt.Errorf("%w: failed to cancel reservation: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payloadJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}

	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		AggregateID:   utils.UUIDToPgtype(reservationID),
		AggregateType: "reservation",
		EventType:     "meal.reservation.cancelled",
		Payload:       payloadJSON,
		MaxRetries:    5,
	})
	if err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return db.Reservation{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return reservation, nil
}

// ExpirePendingReservations expires pending reservations that have timed out
func (r *ReservationRepository) ExpirePendingReservations(ctx context.Context, limit int32) error {
	err := r.queries.ExpirePendingReservations(ctx, limit)
	if err != nil {
		return fmt.Errorf("%w: failed to expire pending reservations: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// CleanupExpiredReservations removes old expired reservations
func (r *ReservationRepository) CleanupExpiredReservations(ctx context.Context, limit int32) error {
	err := r.queries.CleanupExpiredReservations(ctx, limit)
	if err != nil {
		return fmt.Errorf("%w: failed to cleanup expired reservations: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// ConfirmReservationsWithEvents confirms reservations and creates outbox events atomically
func (r *ReservationRepository) ConfirmReservationsWithEvents(ctx context.Context, reservationIDs []uuid.UUID, eventPayloads []map[string]any) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update reservations to confirmed status
	for _, id := range reservationIDs {
		_, err := qtx.UpdateReservationByID(ctx, db.UpdateReservationByIDParams{
			ID:        utils.UUIDToPgtype(id),
			Status:    db.MealReservationStatusEnumConfirmed,
			ExpiresAt: pgtype.Timestamptz{Valid: false}, // Clear expires_at
		})
		if err != nil {
			return fmt.Errorf("%w: failed to confirm reservation: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	// Create outbox events
	for i, payload := range eventPayloads {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
		}

		_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			AggregateID:   utils.UUIDToPgtype(reservationIDs[i]),
			AggregateType: "reservation",
			EventType:     "meal.reservation.created",
			Payload:       payloadJSON,
			MaxRetries:    5,
		})
		if err != nil {
			return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return nil
}
