package service

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/auth-service/internal/db"
	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/auth-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type EventService struct {
	authRepo  *repository.AuthRepository
	eventRepo *repository.EventRepository
	pool      *pgxpool.Pool
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
		// ON CONFLICT DO NOTHING - idempotent
		logger.Warn("failed to create user (might already exist)",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
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
		logger.Warn("failed to create user (might already exist)",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
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
			logger.Warn("failed to sync token version on email change",
				zap.Error(err),
			)
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
