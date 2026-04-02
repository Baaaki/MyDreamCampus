package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateStudentWithEvent creates student and outbox event atomically
func (r *StudentRepository) CreateStudentWithEvent(ctx context.Context, params db.CreateStudentParams, eventPayload map[string]interface{}) (db.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create student
	student, err := qtx.CreateStudent(ctx, params)
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to create student: %w", err)
	}

	// Create outbox event
	payload, _ := json.Marshal(eventPayload)
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  "student.created",
		RoutingKey: "student.created",
		Payload:    payload,
	})
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Student{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return student, nil
}

// GetStudentByID retrieves student by ID
func (r *StudentRepository) GetStudentByID(ctx context.Context, id uuid.UUID) (db.Student, error) {
	student, err := r.queries.GetStudentByID(ctx, utils.UUIDToPgtype(id))
	if err == pgx.ErrNoRows {
		return db.Student{}, fmt.Errorf("student not found")
	}
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to get student: %w", err)
	}
	return student, nil
}

// GetStudentByEmail retrieves student by email
func (r *StudentRepository) GetStudentByEmail(ctx context.Context, email string) (db.Student, error) {
	student, err := r.queries.GetStudentByEmail(ctx, email)
	if err == pgx.ErrNoRows {
		return db.Student{}, nil // Not found is not an error for existence check
	}
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to check student existence: %w", err)
	}
	return student, nil
}

// GetStudentByNumber retrieves student by student number
func (r *StudentRepository) GetStudentByNumber(ctx context.Context, studentNumber string) (db.Student, error) {
	student, err := r.queries.GetStudentByNumber(ctx, studentNumber)
	if err == pgx.ErrNoRows {
		return db.Student{}, nil
	}
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to check student existence: %w", err)
	}
	return student, nil
}

// UpdateStudentWithEvent updates student information with event
func (r *StudentRepository) UpdateStudentWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStudentParams, eventPayload map[string]interface{}) (db.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update student
	student, err := qtx.UpdateStudent(ctx, params)
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to update student: %w", err)
	}

	// Create outbox event
	payload, _ := json.Marshal(eventPayload)
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  "student.updated",
		RoutingKey: "student.updated",
		Payload:    payload,
	})
	if err != nil {
		return db.Student{}, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Student{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return student, nil
}

// SoftDeleteStudentWithEvent soft deletes a student with event
func (r *StudentRepository) SoftDeleteStudentWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]interface{}) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Soft delete student
	err = qtx.SoftDeleteStudent(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	// Create outbox event
	payload, _ := json.Marshal(eventPayload)
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  "student.deactivated",
		RoutingKey: "student.deactivated",
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

// ListStudentsFiltered lists students with filters, sorting, and pagination
func (r *StudentRepository) ListStudentsFiltered(ctx context.Context, params db.ListStudentsParams) ([]db.Student, error) {
	students, err := r.queries.ListStudents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list students: %w", err)
	}
	return students, nil
}

// CountStudents returns total count of active students
func (r *StudentRepository) CountStudents(ctx context.Context) (int64, error) {
	count, err := r.queries.CountStudents(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count students: %w", err)
	}
	return count, nil
}

// ListStudentsByAdvisor lists students by advisor ID
func (r *StudentRepository) ListStudentsByAdvisor(ctx context.Context, advisorID uuid.UUID) ([]db.Student, error) {
	students, err := r.queries.ListStudentsByAdvisor(ctx, utils.UUIDToPgtype(advisorID))
	if err != nil {
		return nil, fmt.Errorf("failed to list students by advisor: %w", err)
	}
	return students, nil
}

// ListOrphanedStudents lists students without advisor
func (r *StudentRepository) ListOrphanedStudents(ctx context.Context, limit, offset int32) ([]db.Student, int64, error) {
	// Get total count
	total, err := r.queries.CountOrphanedStudents(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orphaned students: %w", err)
	}

	// Get orphaned student list
	students, err := r.queries.ListOrphanedStudents(ctx, db.ListOrphanedStudentsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orphaned students: %w", err)
	}

	return students, total, nil
}

// BulkAssignAdvisor assigns advisor to multiple students with event
func (r *StudentRepository) BulkAssignAdvisor(ctx context.Context, studentIDs []uuid.UUID, advisorID uuid.UUID, eventPayloads []map[string]interface{}) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Convert UUIDs to pgtype
	pgStudentIDs := make([]pgtype.UUID, len(studentIDs))
	for i, id := range studentIDs {
		pgStudentIDs[i] = utils.UUIDToPgtype(id)
	}

	// Bulk assign advisor
	err = qtx.BulkAssignAdvisor(ctx, db.BulkAssignAdvisorParams{
		Column1:   pgStudentIDs,
		AdvisorID: utils.UUIDToPgtype(advisorID),
	})
	if err != nil {
		return fmt.Errorf("failed to bulk assign advisor: %w", err)
	}

	// Create outbox events for each student
	for _, eventPayload := range eventPayloads {
		payload, _ := json.Marshal(eventPayload)
		_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType:  "student.updated",
			RoutingKey: "student.updated",
			Payload:    payload,
		})
		if err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UnassignAdvisorByStaffID removes advisor assignment for all students of a staff member
func (r *StudentRepository) UnassignAdvisorByStaffID(ctx context.Context, staffID uuid.UUID) error {
	err := r.queries.UnassignAdvisorByStaffID(ctx, utils.UUIDToPgtype(staffID))
	if err != nil {
		return fmt.Errorf("failed to unassign advisor: %w", err)
	}
	return nil
}

// UnassignAdvisorByStaffIDWithEventMarking removes advisor and marks event as processed atomically
func (r *StudentRepository) UnassignAdvisorByStaffIDWithEventMarking(ctx context.Context, staffID uuid.UUID, eventID, eventType string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Unassign advisor
	err = qtx.UnassignAdvisorByStaffID(ctx, utils.UUIDToPgtype(staffID))
	if err != nil {
		return fmt.Errorf("failed to unassign advisor: %w", err)
	}

	// Mark event as processed (using raw SQL since we don't have processed_events in this repo)
	_, err = tx.Exec(ctx,
		"INSERT INTO processed_events (event_id, event_type) VALUES ($1, $2) ON CONFLICT (event_id) DO NOTHING",
		eventID, eventType,
	)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SearchStudents performs advanced search with filters
func (r *StudentRepository) SearchStudents(ctx context.Context, params db.SearchStudentsParams) ([]db.Student, error) {
	students, err := r.queries.SearchStudents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search students: %w", err)
	}
	return students, nil
}

// ListStudentsByDepartment lists students by department
func (r *StudentRepository) ListStudentsByDepartment(ctx context.Context, department string) ([]db.Student, error) {
	students, err := r.queries.ListStudentsByDepartment(ctx, department)
	if err != nil {
		return nil, fmt.Errorf("failed to list students by department: %w", err)
	}
	return students, nil
}
