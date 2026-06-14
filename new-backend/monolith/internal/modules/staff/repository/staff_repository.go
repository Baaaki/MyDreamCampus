package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/events"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/db"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StaffRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewStaffRepository(pool *pgxpool.Pool) *StaffRepository {
	return &StaffRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateStaffWithEvent creates staff and outbox event atomically
// If role is "teacher", also creates an empty teacher_profile record
func (r *StaffRepository) CreateStaffWithEvent(ctx context.Context, params db.CreateStaffParams, eventPayload map[string]any) (db.Staff, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create staff
	staff, err := qtx.CreateStaff(ctx, params)
	if err != nil {
		// Check for duplicate email constraint violation
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) {
			if pgxErr.Code == "23505" { // unique_violation
				return db.Staff{}, fmt.Errorf("%w: email already exists", serviceErrors.ErrStaffExistsRepo)
			}
		}
		return db.Staff{}, fmt.Errorf("%w: failed to create staff: %v", sharedErrors.ErrQueryFailed, err)
	}

	// If role is teacher, create empty teacher_profile record
	if params.Role == "teacher" {
		_, err = qtx.CreateTeacherProfile(ctx, db.CreateTeacherProfileParams{
			StaffID: staff.ID,
			// All other fields will be NULL/empty by default
		})
		if err != nil {
			return db.Staff{}, fmt.Errorf("%w: failed to create teacher profile: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	// Set the staff ID in event payload (was nil before creation)
	eventPayload["id"] = utils.PgtypeToUUIDString(staff.ID)

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  events.EventStaffCreated,
		RoutingKey: events.RoutingKeyStaffCreated,
		Payload:    payload,
	})
	if err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return staff, nil
}

// GetStaffByID retrieves staff by ID
func (r *StaffRepository) GetStaffByID(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	staff, err := r.queries.GetStaffByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Staff{}, fmt.Errorf("%w: staff with id %s not found", serviceErrors.ErrStaffNotFoundRepo, id)
		}
		return db.Staff{}, fmt.Errorf("%w: failed to get staff: %v", sharedErrors.ErrQueryFailed, err)
	}
	return staff, nil
}

// GetStaffByEmail retrieves staff by email
// Returns empty staff with nil error if not found (for existence checks)
func (r *StaffRepository) GetStaffByEmail(ctx context.Context, email string) (db.Staff, error) {
	staff, err := r.queries.GetStaffByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not found is not an error for existence check - return empty staff
			return db.Staff{}, nil
		}
		return db.Staff{}, fmt.Errorf("%w: failed to check staff existence by email: %v", sharedErrors.ErrQueryFailed, err)
	}
	return staff, nil
}

// UpdateStaffWithEvent updates staff information with event
func (r *StaffRepository) UpdateStaffWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStaffParams, eventPayload map[string]any) (db.Staff, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update staff
	staff, err := qtx.UpdateStaff(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Staff{}, fmt.Errorf("%w: staff with id %s not found for update", serviceErrors.ErrStaffNotFoundRepo, id)
		}
		return db.Staff{}, fmt.Errorf("%w: failed to update staff: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  events.EventStaffUpdated,
		RoutingKey: events.RoutingKeyStaffUpdated,
		Payload:    payload,
	})
	if err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Staff{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return staff, nil
}

// SoftDeleteStaffWithEvent soft deletes a staff member with event
func (r *StaffRepository) SoftDeleteStaffWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]any) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Soft delete staff
	err = qtx.SoftDeleteStaff(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: staff with id %s not found for deletion", serviceErrors.ErrStaffNotFoundRepo, id)
		}
		return fmt.Errorf("%w: failed to delete staff: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  events.EventStaffDeactivated,
		RoutingKey: events.RoutingKeyStaffDeactivated,
		Payload:    payload,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return nil
}

// ListStaff lists staff with pagination
func (r *StaffRepository) ListStaff(ctx context.Context, limit, offset int32) ([]db.Staff, int64, error) {
	// Get total count
	total, err := r.queries.CountStaff(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to count staff: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Get staff list
	staffList, err := r.queries.ListStaff(ctx, db.ListStaffParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to list staff: %v", sharedErrors.ErrQueryFailed, err)
	}

	return staffList, total, nil
}

// GetInstructorsByDepartment retrieves instructors by department
func (r *StaffRepository) GetInstructorsByDepartment(ctx context.Context, department string) ([]db.Staff, error) {
	instructors, err := r.queries.GetInstructorsByDepartment(ctx, utils.StringToPgText(department))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get instructors by department: %v", sharedErrors.ErrQueryFailed, err)
	}
	return instructors, nil
}
