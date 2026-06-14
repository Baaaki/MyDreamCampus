package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/events"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/student-service/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// Helper functions to convert sqlc row types to db.Student
func createStudentRowToStudent(row db.CreateStudentRow) db.Student {
	return db.Student{
		ID:             row.ID,
		StudentNumber:  row.StudentNumber,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Email:          row.Email,
		Faculty:        row.Faculty,
		Department:     row.Department,
		EnrollmentYear: row.EnrollmentYear,
		ClassLevel:     row.ClassLevel,
		AdvisorID:      row.AdvisorID,
		AdvisorName:    row.AdvisorName,
		Status:         row.Status,
		IsActive:       row.IsActive,
		DeletedAt:      row.DeletedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func getStudentByIDRowToStudent(row db.GetStudentByIDRow) db.Student {
	return db.Student{
		ID:             row.ID,
		StudentNumber:  row.StudentNumber,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Email:          row.Email,
		Faculty:        row.Faculty,
		Department:     row.Department,
		EnrollmentYear: row.EnrollmentYear,
		ClassLevel:     row.ClassLevel,
		AdvisorID:      row.AdvisorID,
		AdvisorName:    row.AdvisorName,
		Status:         row.Status,
		IsActive:       row.IsActive,
		DeletedAt:      row.DeletedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func getStudentByEmailRowToStudent(row db.GetStudentByEmailRow) db.Student {
	return db.Student{
		ID:             row.ID,
		StudentNumber:  row.StudentNumber,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Email:          row.Email,
		Faculty:        row.Faculty,
		Department:     row.Department,
		EnrollmentYear: row.EnrollmentYear,
		ClassLevel:     row.ClassLevel,
		AdvisorID:      row.AdvisorID,
		AdvisorName:    row.AdvisorName,
		Status:         row.Status,
		IsActive:       row.IsActive,
		DeletedAt:      row.DeletedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func getStudentByNumberRowToStudent(row db.GetStudentByNumberRow) db.Student {
	return db.Student{
		ID:             row.ID,
		StudentNumber:  row.StudentNumber,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Email:          row.Email,
		Faculty:        row.Faculty,
		Department:     row.Department,
		EnrollmentYear: row.EnrollmentYear,
		ClassLevel:     row.ClassLevel,
		AdvisorID:      row.AdvisorID,
		AdvisorName:    row.AdvisorName,
		Status:         row.Status,
		IsActive:       row.IsActive,
		DeletedAt:      row.DeletedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func updateStudentRowToStudent(row db.UpdateStudentRow) db.Student {
	return db.Student{
		ID:             row.ID,
		StudentNumber:  row.StudentNumber,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Email:          row.Email,
		Faculty:        row.Faculty,
		Department:     row.Department,
		EnrollmentYear: row.EnrollmentYear,
		ClassLevel:     row.ClassLevel,
		AdvisorID:      row.AdvisorID,
		AdvisorName:    row.AdvisorName,
		Status:         row.Status,
		IsActive:       row.IsActive,
		DeletedAt:      row.DeletedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func listStudentsRowsToStudents(rows []db.ListStudentsRow) []db.Student {
	students := make([]db.Student, len(rows))
	for i, row := range rows {
		students[i] = db.Student{
			ID:             row.ID,
			StudentNumber:  row.StudentNumber,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Email:          row.Email,
			Faculty:        row.Faculty,
			Department:     row.Department,
			EnrollmentYear: row.EnrollmentYear,
			ClassLevel:     row.ClassLevel,
			AdvisorID:      row.AdvisorID,
			AdvisorName:    row.AdvisorName,
			Status:         row.Status,
			IsActive:       row.IsActive,
			DeletedAt:      row.DeletedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return students
}

func listStudentsByAdvisorRowsToStudents(rows []db.ListStudentsByAdvisorRow) []db.Student {
	students := make([]db.Student, len(rows))
	for i, row := range rows {
		students[i] = db.Student{
			ID:             row.ID,
			StudentNumber:  row.StudentNumber,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Email:          row.Email,
			Faculty:        row.Faculty,
			Department:     row.Department,
			EnrollmentYear: row.EnrollmentYear,
			ClassLevel:     row.ClassLevel,
			AdvisorID:      row.AdvisorID,
			AdvisorName:    row.AdvisorName,
			Status:         row.Status,
			IsActive:       row.IsActive,
			DeletedAt:      row.DeletedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return students
}

func listOrphanedStudentsRowsToStudents(rows []db.ListOrphanedStudentsRow) []db.Student {
	students := make([]db.Student, len(rows))
	for i, row := range rows {
		students[i] = db.Student{
			ID:             row.ID,
			StudentNumber:  row.StudentNumber,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Email:          row.Email,
			Faculty:        row.Faculty,
			Department:     row.Department,
			EnrollmentYear: row.EnrollmentYear,
			ClassLevel:     row.ClassLevel,
			AdvisorID:      row.AdvisorID,
			AdvisorName:    row.AdvisorName,
			Status:         row.Status,
			IsActive:       row.IsActive,
			DeletedAt:      row.DeletedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return students
}

func searchStudentsRowsToStudents(rows []db.SearchStudentsRow) []db.Student {
	students := make([]db.Student, len(rows))
	for i, row := range rows {
		students[i] = db.Student{
			ID:             row.ID,
			StudentNumber:  row.StudentNumber,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Email:          row.Email,
			Faculty:        row.Faculty,
			Department:     row.Department,
			EnrollmentYear: row.EnrollmentYear,
			ClassLevel:     row.ClassLevel,
			AdvisorID:      row.AdvisorID,
			AdvisorName:    row.AdvisorName,
			Status:         row.Status,
			IsActive:       row.IsActive,
			DeletedAt:      row.DeletedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return students
}

func listStudentsByDepartmentRowsToStudents(rows []db.ListStudentsByDepartmentRow) []db.Student {
	students := make([]db.Student, len(rows))
	for i, row := range rows {
		students[i] = db.Student{
			ID:             row.ID,
			StudentNumber:  row.StudentNumber,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Email:          row.Email,
			Faculty:        row.Faculty,
			Department:     row.Department,
			EnrollmentYear: row.EnrollmentYear,
			ClassLevel:     row.ClassLevel,
			AdvisorID:      row.AdvisorID,
			AdvisorName:    row.AdvisorName,
			Status:         row.Status,
			IsActive:       row.IsActive,
			DeletedAt:      row.DeletedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return students
}

// CreateStudentWithEvent creates student and outbox event atomically
func (r *StudentRepository) CreateStudentWithEvent(ctx context.Context, params db.CreateStudentParams, eventPayload map[string]any) (db.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create student
	student, err := qtx.CreateStudent(ctx, params)
	if err != nil {
		// Check for duplicate constraints
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) {
			if pgxErr.Code == "23505" { // unique_violation
				if pgxErr.ConstraintName == "students_student_number_key" {
					return db.Student{}, fmt.Errorf("%w: student number already exists", serviceErrors.ErrStudentNumberExistsRepo)
				}
				if pgxErr.ConstraintName == "students_email_key" {
					return db.Student{}, fmt.Errorf("%w: email already exists", serviceErrors.ErrStudentEmailExistsRepo)
				}
			}
		}
		return db.Student{}, fmt.Errorf("%w: failed to create student: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Set the student ID in event payload (was nil before creation)
	eventPayload["id"] = utils.PgtypeToUUIDString(student.ID)

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  events.EventStudentCreated,
		RoutingKey: events.RoutingKeyStudentCreated,
		Payload:    payload,
	})
	if err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return createStudentRowToStudent(student), nil
}

// GetStudentByID retrieves student by ID
func (r *StudentRepository) GetStudentByID(ctx context.Context, id uuid.UUID) (db.Student, error) {
	student, err := r.queries.GetStudentByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Student{}, fmt.Errorf("%w: student with id %s not found", serviceErrors.ErrStudentNotFoundRepo, id)
		}
		return db.Student{}, fmt.Errorf("%w: failed to get student: %v", sharedErrors.ErrQueryFailed, err)
	}
	return getStudentByIDRowToStudent(student), nil
}

// GetStudentByEmail retrieves student by email
// Returns empty student with nil error if not found (for existence checks)
func (r *StudentRepository) GetStudentByEmail(ctx context.Context, email string) (db.Student, error) {
	student, err := r.queries.GetStudentByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Student{}, nil // Not found is not an error for existence check
		}
		return db.Student{}, fmt.Errorf("%w: failed to check student existence by email: %v", sharedErrors.ErrQueryFailed, err)
	}
	return getStudentByEmailRowToStudent(student), nil
}

// GetStudentByNumber retrieves student by student number
// Returns empty student with nil error if not found (for existence checks)
func (r *StudentRepository) GetStudentByNumber(ctx context.Context, studentNumber string) (db.Student, error) {
	student, err := r.queries.GetStudentByNumber(ctx, studentNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Student{}, nil
		}
		return db.Student{}, fmt.Errorf("%w: failed to check student existence by number: %v", sharedErrors.ErrQueryFailed, err)
	}
	return getStudentByNumberRowToStudent(student), nil
}

// UpdateStudentWithEvent updates student information with event
func (r *StudentRepository) UpdateStudentWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStudentParams, eventPayload map[string]any) (db.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update student
	student, err := qtx.UpdateStudent(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Student{}, fmt.Errorf("%w: student with id %s not found for update", serviceErrors.ErrStudentNotFoundRepo, id)
		}
		return db.Student{}, fmt.Errorf("%w: failed to update student: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  events.EventStudentUpdated,
		RoutingKey: events.RoutingKeyStudentUpdated,
		Payload:    payload,
	})
	if err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Student{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return updateStudentRowToStudent(student), nil
}

// SoftDeleteStudentWithEvent soft deletes a student with event
func (r *StudentRepository) SoftDeleteStudentWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]any) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Soft delete student
	err = qtx.SoftDeleteStudent(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: student with id %s not found for deletion", serviceErrors.ErrStudentNotFoundRepo, id)
		}
		return fmt.Errorf("%w: failed to delete student: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:  events.EventStudentDeactivated,
		RoutingKey: events.RoutingKeyStudentDeactivated,
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

// ListStudentsFiltered lists students with filters, sorting, and pagination
func (r *StudentRepository) ListStudentsFiltered(ctx context.Context, params db.ListStudentsParams) ([]db.Student, error) {
	students, err := r.queries.ListStudents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list students: %v", sharedErrors.ErrQueryFailed, err)
	}
	return listStudentsRowsToStudents(students), nil
}

// CountStudents returns total count of active students
func (r *StudentRepository) CountStudents(ctx context.Context) (int64, error) {
	count, err := r.queries.CountStudents(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count students: %v", sharedErrors.ErrQueryFailed, err)
	}
	return count, nil
}

// ListStudentsByAdvisor lists students by advisor ID
func (r *StudentRepository) ListStudentsByAdvisor(ctx context.Context, advisorID uuid.UUID) ([]db.Student, error) {
	students, err := r.queries.ListStudentsByAdvisor(ctx, utils.UUIDToPgtype(advisorID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list students by advisor: %v", sharedErrors.ErrQueryFailed, err)
	}
	return listStudentsByAdvisorRowsToStudents(students), nil
}

// ListOrphanedStudents lists students without advisor
func (r *StudentRepository) ListOrphanedStudents(ctx context.Context, limit, offset int32) ([]db.Student, int64, error) {
	// Get total count
	total, err := r.queries.CountOrphanedStudents(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to count orphaned students: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Get orphaned student list
	students, err := r.queries.ListOrphanedStudents(ctx, db.ListOrphanedStudentsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to list orphaned students: %v", sharedErrors.ErrQueryFailed, err)
	}

	return listOrphanedStudentsRowsToStudents(students), total, nil
}

// BulkAssignAdvisor assigns advisor to multiple students with event
func (r *StudentRepository) BulkAssignAdvisor(ctx context.Context, studentIDs []uuid.UUID, advisorID uuid.UUID, advisorName string, eventPayloads []map[string]any) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
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
		Column1:     pgStudentIDs,
		AdvisorID:   utils.UUIDToPgtype(advisorID),
		AdvisorName: utils.StringToPgText(advisorName),
	})
	if err != nil {
		return fmt.Errorf("%w: failed to bulk assign advisor: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox events for each student
	for _, eventPayload := range eventPayloads {
		payload, err := json.Marshal(eventPayload)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal event payload: %v", sharedErrors.ErrQueryFailed, err)
		}
		_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType:  events.EventStudentUpdated,
			RoutingKey: events.RoutingKeyStudentUpdated,
			Payload:    payload,
		})
		if err != nil {
			return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return nil
}

// UnassignAdvisorByStaffID removes advisor assignment for all students of a staff member
func (r *StudentRepository) UnassignAdvisorByStaffID(ctx context.Context, staffID uuid.UUID) error {
	err := r.queries.UnassignAdvisorByStaffID(ctx, utils.UUIDToPgtype(staffID))
	if err != nil {
		return fmt.Errorf("%w: failed to unassign advisor: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// UnassignAdvisorByStaffIDWithEventMarking removes advisor and marks event as processed atomically
func (r *StudentRepository) UnassignAdvisorByStaffIDWithEventMarking(ctx context.Context, staffID uuid.UUID, eventID, eventType string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Unassign advisor
	err = qtx.UnassignAdvisorByStaffID(ctx, utils.UUIDToPgtype(staffID))
	if err != nil {
		return fmt.Errorf("%w: failed to unassign advisor: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Mark event as processed
	err = qtx.CreateProcessedEvent(ctx, db.CreateProcessedEventParams{
		EventID:   eventID,
		EventType: eventType,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to mark event as processed: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return nil
}

// SearchStudents performs advanced search with filters
func (r *StudentRepository) SearchStudents(ctx context.Context, params db.SearchStudentsParams) ([]db.Student, error) {
	students, err := r.queries.SearchStudents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to search students: %v", sharedErrors.ErrQueryFailed, err)
	}
	return searchStudentsRowsToStudents(students), nil
}

// ListStudentsByDepartment lists students by department
func (r *StudentRepository) ListStudentsByDepartment(ctx context.Context, department string) ([]db.Student, error) {
	students, err := r.queries.ListStudentsByDepartment(ctx, department)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list students by department: %v", sharedErrors.ErrQueryFailed, err)
	}
	return listStudentsByDepartmentRowsToStudents(students), nil
}
