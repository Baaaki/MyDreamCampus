package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/staff-service/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
func (r *StaffRepository) CreateStaffWithEvent(ctx context.Context, params db.CreateStaffParams, eventPayload map[string]interface{}) (db.Staff, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create staff
	staff, err := qtx.CreateStaff(ctx, params)
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to create staff: %w", err)
	}

	// Create outbox event
	payload, _ := json.Marshal(eventPayload)
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  "staff.created",
		RoutingKey: "staff.created",
		Payload:    payload,
	})
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Staff{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return staff, nil
}

// GetStaffByID retrieves staff by ID
func (r *StaffRepository) GetStaffByID(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	staff, err := r.queries.GetStaffByID(ctx, utils.UUIDToPgtype(id))
	if err == pgx.ErrNoRows {
		return db.Staff{}, fmt.Errorf("staff not found")
	}
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to get staff: %w", err)
	}
	return staff, nil
}

// GetStaffByEmail retrieves staff by email
func (r *StaffRepository) GetStaffByEmail(ctx context.Context, email string) (db.Staff, error) {
	staff, err := r.queries.GetStaffByEmail(ctx, email)
	if err == pgx.ErrNoRows {
		return db.Staff{}, nil // Not found is not an error for existence check
	}
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to check staff existence: %w", err)
	}
	return staff, nil
}

// UpdateStaffWithEvent updates staff information with event
func (r *StaffRepository) UpdateStaffWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStaffParams, eventPayload map[string]interface{}) (db.Staff, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update staff
	staff, err := qtx.UpdateStaff(ctx, params)
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to update staff: %w", err)
	}

	// Create outbox event
	payload, _ := json.Marshal(eventPayload)
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  "staff.updated",
		RoutingKey: "staff.updated",
		Payload:    payload,
	})
	if err != nil {
		return db.Staff{}, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Staff{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return staff, nil
}

// SoftDeleteStaffWithEvent soft deletes a staff member with event
func (r *StaffRepository) SoftDeleteStaffWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]interface{}) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Soft delete staff
	err = qtx.SoftDeleteStaff(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("failed to delete staff: %w", err)
	}

	// Create outbox event
	payload, _ := json.Marshal(eventPayload)
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  "staff.deactivated",
		RoutingKey: "staff.deactivated",
		Payload:    payload,
	})
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListStaff lists staff with pagination
func (r *StaffRepository) ListStaff(ctx context.Context, limit, offset int32) ([]db.Staff, int64, error) {
	// Get total count
	total, err := r.queries.CountStaff(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count staff: %w", err)
	}

	// Get staff list
	staffList, err := r.queries.ListStaff(ctx, db.ListStaffParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list staff: %w", err)
	}

	return staffList, total, nil
}
