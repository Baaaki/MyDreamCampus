package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type EventService struct {
	authRepo  *repository.AuthRepository
	eventRepo *repository.EventRepository
	pool      *pgxpool.Pool
}

// isDuplicateUser classifies CreateUser failures that mean "the user is
// already there": pgx.ErrNoRows (ON CONFLICT (id) DO NOTHING swallowed the
// insert on redelivery) or a 23505 on the email unique index. Anything
// else is a real failure — the event must be nacked and retried, otherwise
// a transient DB error would silently leave the person without a login.
func isDuplicateUser(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}
	return isEmailConflict(err)
}

// isEmailConflict is the dangerous duplicate flavor: the email belongs to a
// DIFFERENT user id. Retrying cannot fix it, but it must be loudly visible —
// the new person has no login until an operator resolves the collision.
func isEmailConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func NewEventService(
	authRepo *repository.AuthRepository,
	eventRepo *repository.EventRepository,
	pool *pgxpool.Pool,
) *EventService {
	return &EventService{
		authRepo:  authRepo,
		eventRepo: eventRepo,
		pool:      pool,
	}
}

// HandleStudentCreated processes student.created event
func (s *EventService) HandleStudentCreated(ctx context.Context, event dto.StudentCreatedEvent) error {
	// Check idempotency
	processed, err := s.eventRepo.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("%w: failed to check event processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	if processed {
		logger.Info("event already processed, skipping",
			zap.String("event_id", event.EventID),
			zap.String("event_type", event.EventType),
		)
		return nil
	}

	// Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	// Parse user ID
	userID, err := uuid.Parse(event.Data.ID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Initial password = email (as per documentation)
	passwordHash, err := utils.HashPassword(event.Data.Email)
	if err != nil {
		return fmt.Errorf("%w: failed to hash password: %v", sharedErrors.ErrInternal, err)
	}

	// Create user in auth database
	queries := db.New(tx)
	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		ID:                  utils.UUIDToPgtype(userID),
		Email:               event.Data.Email,
		PasswordHash:        passwordHash,
		Role:                "student",
		Department:          utils.StringToPointer(event.Data.Department),
		IsActive:            utils.BoolPtr(true),
		TokenVersion:        utils.Int32Ptr(1),
		ForcePasswordChange: utils.BoolPtr(true),
	})
	if err != nil {
		if !isDuplicateUser(err) {
			// Real failure: do NOT mark the event processed — nack so the
			// broker redelivers and the user eventually gets a login.
			return fmt.Errorf("%w: failed to create user: %v", sharedErrors.ErrQueryFailed, err)
		}
		if isEmailConflict(err) {
			// Email owned by another user id: permanent conflict, retrying
			// is pointless — surface loudly for manual resolution.
			logger.Error("email already belongs to a different user — projection skipped",
				zap.Error(err),
				zap.String("user_id", userID.String()),
			)
		} else {
			logger.Warn("user already exists, skipping create",
				zap.Error(err),
				zap.String("user_id", userID.String()),
			)
		}
	} else {
		// Insert outbox event for notification (Welcome email)
		payloadBytes, _ := json.Marshal(buildUserRegisteredPayload(
			event.Data.ID,
			event.Data.Email,
			event.Data.FirstName,
			event.Data.LastName,
			"student",
		))
		_, err = queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType:  events.EventTypeUserRegistered,
			RoutingKey: events.RoutingKeyUserRegistered,
			Payload:    payloadBytes,
		})
		if err != nil {
			return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	// Mark event as processed
	err = queries.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   event.EventID,
		EventType: event.EventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event processed: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	logger.Info("student.created event processed",
		zap.String("event_id", event.EventID),
		zap.String("user_id", userID.String()),
		zap.String("email", event.Data.Email),
	)

	return nil
}

// HandleStaffCreated processes staff.created event
func (s *EventService) HandleStaffCreated(ctx context.Context, event dto.StaffCreatedEvent) error {
	// Check idempotency
	processed, err := s.eventRepo.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("%w: failed to check event processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	if processed {
		logger.Info("event already processed, skipping",
			zap.String("event_id", event.EventID),
			zap.String("event_type", event.EventType),
		)
		return nil
	}

	// Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	// Parse user ID
	userID, err := uuid.Parse(event.Data.ID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Initial password = email
	passwordHash, err := utils.HashPassword(event.Data.Email)
	if err != nil {
		return fmt.Errorf("%w: failed to hash password: %v", sharedErrors.ErrInternal, err)
	}

	// Create user in auth database
	queries := db.New(tx)
	var departmentPtr *string
	if event.Data.Department != "" {
		departmentPtr = utils.StringToPointer(event.Data.Department)
	}
	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		ID:                  utils.UUIDToPgtype(userID),
		Email:               event.Data.Email,
		PasswordHash:        passwordHash,
		Role:                event.Data.Role, // teacher or admin
		Department:          departmentPtr,
		IsActive:            utils.BoolPtr(true),
		TokenVersion:        utils.Int32Ptr(1),
		ForcePasswordChange: utils.BoolPtr(true),
	})
	if err != nil {
		if !isDuplicateUser(err) {
			// Real failure: do NOT mark the event processed — nack so the
			// broker redelivers and the staff member eventually gets a login.
			return fmt.Errorf("%w: failed to create user: %v", sharedErrors.ErrQueryFailed, err)
		}
		if isEmailConflict(err) {
			// Email owned by another user id: permanent conflict, retrying
			// is pointless — surface loudly for manual resolution.
			logger.Error("email already belongs to a different user — projection skipped",
				zap.Error(err),
				zap.String("user_id", userID.String()),
			)
		} else {
			logger.Warn("user already exists, skipping create",
				zap.Error(err),
				zap.String("user_id", userID.String()),
			)
		}
	} else {
		// Insert outbox event for notification (Welcome email)
		payloadBytes, _ := json.Marshal(buildUserRegisteredPayload(
			event.Data.ID,
			event.Data.Email,
			event.Data.FirstName,
			event.Data.LastName,
			event.Data.Role,
		))
		_, err = queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType:  events.EventTypeUserRegistered,
			RoutingKey: events.RoutingKeyUserRegistered,
			Payload:    payloadBytes,
		})
		if err != nil {
			return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	// Mark event as processed
	err = queries.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   event.EventID,
		EventType: event.EventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event processed: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	logger.Info("staff.created event processed",
		zap.String("event_id", event.EventID),
		zap.String("user_id", userID.String()),
		zap.String("email", event.Data.Email),
		zap.String("role", event.Data.Role),
	)

	return nil
}

// HandleUserUpdated processes student.updated and staff.updated events
func (s *EventService) HandleUserUpdated(ctx context.Context, event dto.UserUpdatedEvent) error {
	// Check idempotency
	processed, err := s.eventRepo.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("%w: failed to check event processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	if processed {
		logger.Info("event already processed, skipping",
			zap.String("event_id", event.EventID),
			zap.String("event_type", event.EventType),
		)
		return nil
	}

	// Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	// Parse user ID
	userID, err := uuid.Parse(event.Data.ID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	queries := db.New(tx)

	// Update user fields
	updateParams := db.UpdateUserParams{
		ID: utils.UUIDToPgtype(userID),
	}

	if email, ok := event.Data.ChangedFields["email"]; ok {
		updateParams.Email = utils.StringToPointer(email)
	}

	if department, ok := event.Data.ChangedFields["department"]; ok {
		updateParams.Department = utils.StringToPointer(department)
	}

	err = queries.UpdateUser(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("%w: failed to update user: %v", sharedErrors.ErrQueryFailed, err)
	}

	// If email changed, increment token version for security
	if email, ok := event.Data.ChangedFields["email"]; ok {
		_, err = queries.CheckEmailVersionSync(ctx, db.CheckEmailVersionSyncParams{
			ID:    utils.UUIDToPgtype(userID),
			Email: email,
		})
		if err != nil {
			// Token version bump is the thing that revokes sessions issued
			// for the old email. Failing silently would leave them valid, so
			// roll the whole event back and let the broker retry.
			return fmt.Errorf("%w: failed to sync token version on email change: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	// Mark event as processed
	err = queries.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   event.EventID,
		EventType: event.EventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event processed: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	logger.Info("user.updated event processed",
		zap.String("event_id", event.EventID),
		zap.String("user_id", userID.String()),
	)

	return nil
}

// HandleUserDeactivated processes student.deactivated and staff.deactivated events
func (s *EventService) HandleUserDeactivated(ctx context.Context, event dto.UserDeactivatedEvent) error {
	// Check idempotency
	processed, err := s.eventRepo.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("%w: failed to check event processed: %v", sharedErrors.ErrQueryFailed, err)
	}
	if processed {
		logger.Info("event already processed, skipping",
			zap.String("event_id", event.EventID),
			zap.String("event_type", event.EventType),
		)
		return nil
	}

	// Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	// Parse user ID
	userID, err := uuid.Parse(event.Data.ID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	queries := db.New(tx)

	// Deactivate user
	err = queries.DeactivateUser(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		return fmt.Errorf("%w: failed to deactivate user: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Delete all sessions
	err = queries.DeleteAllUserSessions(ctx, utils.UUIDToPgtype(userID))
	if err != nil {
		logger.Warn("failed to delete user sessions",
			zap.Error(err),
		)
		// Continue anyway
	}

	// Mark event as processed
	err = queries.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID:   event.EventID,
		EventType: event.EventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event processed: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	logger.Info("user.deactivated event processed",
		zap.String("event_id", event.EventID),
		zap.String("user_id", userID.String()),
	)

	return nil
}
