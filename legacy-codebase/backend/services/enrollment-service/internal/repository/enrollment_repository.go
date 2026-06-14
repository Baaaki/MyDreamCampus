package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type EnrollmentRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) *EnrollmentRepository {
	return &EnrollmentRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateProgramWithCoursesAndEvent creates enrollment program, courses and outbox event atomically
func (r *EnrollmentRepository) CreateProgramWithCoursesAndEvent(
	ctx context.Context,
	programParams db.CreateEnrollmentProgramParams,
	courseIDs []uuid.UUID,
	eventPayload map[string]any,
) (db.EnrollmentProgram, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Convert []uuid.UUID to []pgtype.UUID for query
	pgCourseIDs := make([]pgtype.UUID, len(courseIDs))
	for i, id := range courseIDs {
		pgCourseIDs[i] = utils.UUIDToPgtype(id)
	}

	// Lock courses and check capacity (SELECT FOR UPDATE)
	courses, err := qtx.GetCoursesForCapacityCheck(ctx, pgCourseIDs)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to lock courses: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Check capacity for each course
	for _, course := range courses {
		if course.CurrentEnrollment.Int16 >= course.MaxCapacity {
			// Course is full, return conflict error (transaction will rollback)
			return db.EnrollmentProgram{}, fmt.Errorf("%w: course capacity is full", sharedErrors.ErrConflict)
		}
	}

	// Create enrollment program
	program, err := qtx.CreateEnrollmentProgram(ctx, programParams)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to create enrollment program: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Add courses to program
	for _, courseID := range courseIDs {
		_, err := qtx.CreateEnrollmentProgramCourse(ctx, db.CreateEnrollmentProgramCourseParams{
			ProgramID: program.ID,
			CourseID:  utils.UUIDToPgtype(courseID),
		})
		if err != nil {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to add course to program: %v", sharedErrors.ErrQueryFailed, err)
		}

		// Increment enrollment count (returns 0 rows if capacity is full)
		rowsAffected, err := qtx.IncrementEnrollment(ctx, utils.UUIDToPgtype(courseID))
		if err != nil {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to increment enrollment: %v", sharedErrors.ErrQueryFailed, err)
		}
		if rowsAffected == 0 {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: course capacity is full", sharedErrors.ErrConflict)
		}
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to marshal submitted event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:   events.EventEnrollmentProgramSubmitted,
		AggregateID: program.ID,
		Payload:     payload,
	})
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return program, nil
}

func (r *EnrollmentRepository) GetEnrollmentProgramByID(ctx context.Context, id uuid.UUID) (db.EnrollmentProgram, error) {
	program, err := r.queries.GetEnrollmentProgramByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: enrollment program not found", sharedErrors.ErrNotFound)
		}
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to get enrollment program: %v", sharedErrors.ErrQueryFailed, err)
	}
	return program, nil
}

func (r *EnrollmentRepository) GetEnrollmentProgramByStudentAndSemester(ctx context.Context, studentID uuid.UUID, semester string) (db.EnrollmentProgram, error) {
	program, err := r.queries.GetEnrollmentProgramByStudentAndSemester(ctx, db.GetEnrollmentProgramByStudentAndSemesterParams{
		StudentID: utils.UUIDToPgtype(studentID),
		Semester:  semester,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: enrollment program not found", sharedErrors.ErrNotFound)
		}
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to get enrollment program: %v", sharedErrors.ErrQueryFailed, err)
	}
	return program, nil
}

func (r *EnrollmentRepository) GetEnrollmentProgramsByStudent(ctx context.Context, studentID uuid.UUID, semester *string, status *string) ([]db.EnrollmentProgram, error) {
	var semesterParam string
	var statusParam string

	if semester != nil {
		semesterParam = *semester
	}

	if status != nil {
		statusParam = *status
	}

	programs, err := r.queries.GetEnrollmentProgramsByStudent(ctx, db.GetEnrollmentProgramsByStudentParams{
		StudentID: utils.UUIDToPgtype(studentID),
		Column2:   semesterParam,
		Column3:   statusParam,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get enrollment programs: %v", sharedErrors.ErrQueryFailed, err)
	}
	return programs, nil
}

func (r *EnrollmentRepository) GetCoursesByProgramID(ctx context.Context, programID uuid.UUID) ([]db.GetCoursesByProgramIDRow, error) {
	courses, err := r.queries.GetCoursesByProgramID(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get program courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

func (r *EnrollmentRepository) GetPendingProgramsByAdvisor(ctx context.Context, advisorID uuid.UUID) ([]db.GetPendingProgramsByAdvisorRow, error) {
	programs, err := r.queries.GetPendingProgramsByAdvisor(ctx, utils.UUIDToPgtypeNullable(advisorID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending programs: %v", sharedErrors.ErrQueryFailed, err)
	}
	return programs, nil
}

// ApproveProgramWithEvent approves program and creates outbox event
func (r *EnrollmentRepository) ApproveProgramWithEvent(
	ctx context.Context,
	programID uuid.UUID,
	eventPayload map[string]any,
) (db.EnrollmentProgram, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update program status to approved
	program, err := qtx.UpdateProgramStatus(ctx, db.UpdateProgramStatusParams{
		ID: utils.UUIDToPgtype(programID),
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentStatusEnumApproved,
			Valid:                true,
		},
	})
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to approve program: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to marshal approved event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:   events.EventEnrollmentProgramApproved,
		AggregateID: utils.UUIDToPgtype(programID),
		Payload:     payload,
	})
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return program, nil
}

// RejectProgramWithEventAndLog rejects program, creates rejection log, decrements enrollments, creates outbox event
func (r *EnrollmentRepository) RejectProgramWithEventAndLog(
	ctx context.Context,
	programID uuid.UUID,
	rejectionLogParams db.CreateRejectionLogParams,
	courseIDs []uuid.UUID,
	eventPayload map[string]any,
) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("repository", "EnrollmentRepository"),
		zap.String("method", "RejectProgramWithEventAndLog"),
	)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Lock the program row — prevents concurrent reject/cancel causing double decrement
	_, err = qtx.LockPendingProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: program not found or not pending: %v", sharedErrors.ErrNotFound, err)
	}

	// Create rejection log
	_, err = qtx.CreateRejectionLog(ctx, rejectionLogParams)
	if err != nil {
		return fmt.Errorf("%w: failed to create rejection log: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Decrement enrollment counts
	for _, courseID := range courseIDs {
		rowsAffected, err := qtx.DecrementEnrollment(ctx, utils.UUIDToPgtype(courseID))
		if err != nil {
			return fmt.Errorf("%w: failed to decrement enrollment: %v", sharedErrors.ErrQueryFailed, err)
		}
		if rowsAffected == 0 {
			log.Warn("decrement enrollment affected 0 rows — possible counter inconsistency",
				zap.String("course_id", courseID.String()),
				zap.String("program_id", programID.String()),
				zap.String("context", "reject"),
			)
		}
	}

	// Delete program
	err = qtx.DeleteEnrollmentProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete program: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal rejected event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:   events.EventEnrollmentProgramRejected,
		AggregateID: utils.UUIDToPgtype(programID),
		Payload:     payload,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return nil
}

func (r *EnrollmentRepository) GetLatestRejectionByStudentAndSemester(ctx context.Context, studentID uuid.UUID, semester string) (db.EnrollmentRejectionLog, error) {
	rejection, err := r.queries.GetLatestRejectionByStudentAndSemester(ctx, db.GetLatestRejectionByStudentAndSemesterParams{
		StudentID: utils.UUIDToPgtype(studentID),
		Semester:  semester,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.EnrollmentRejectionLog{}, fmt.Errorf("%w: no rejection found", sharedErrors.ErrNotFound)
		}
		return db.EnrollmentRejectionLog{}, fmt.Errorf("%w: failed to get latest rejection: %v", sharedErrors.ErrQueryFailed, err)
	}
	return rejection, nil
}

func (r *EnrollmentRepository) GetRejectionsByStudentAndSemester(ctx context.Context, studentID uuid.UUID, semester *string) ([]db.EnrollmentRejectionLog, error) {
	var semesterParam string
	if semester != nil {
		semesterParam = *semester
	}

	rejections, err := r.queries.GetRejectionsByStudentAndSemester(ctx, db.GetRejectionsByStudentAndSemesterParams{
		StudentID: utils.UUIDToPgtype(studentID),
		Column2:   semesterParam,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get rejections: %v", sharedErrors.ErrQueryFailed, err)
	}
	return rejections, nil
}

func (r *EnrollmentRepository) CountRejectionsByStudentAndSemester(ctx context.Context, studentID uuid.UUID, semester string) (int64, error) {
	count, err := r.queries.CountRejectionsByStudentAndSemester(ctx, db.CountRejectionsByStudentAndSemesterParams{
		StudentID: utils.UUIDToPgtype(studentID),
		Semester:  semester,
	})
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count rejections: %v", sharedErrors.ErrQueryFailed, err)
	}
	return count, nil
}

func (r *EnrollmentRepository) CheckPrerequisitePassed(ctx context.Context, studentID uuid.UUID, courseCode string) (bool, error) {
	result, err := r.queries.CheckPrerequisitePassed(ctx, db.CheckPrerequisitePassedParams{
		StudentID:  utils.UUIDToPgtype(studentID),
		CourseCode: courseCode,
	})
	if err != nil {
		return false, fmt.Errorf("%w: failed to check prerequisite: %v", sharedErrors.ErrQueryFailed, err)
	}
	return result, nil
}

func (r *EnrollmentRepository) UpsertPassedPrerequisite(ctx context.Context, params db.UpsertPassedPrerequisiteParams) error {
	_, err := r.queries.UpsertPassedPrerequisite(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: failed to upsert passed prerequisite: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// CancelProgramWithEvent cancels program (decrements enrollments, deletes program, creates event)
func (r *EnrollmentRepository) CancelProgramWithEvent(
	ctx context.Context,
	programID uuid.UUID,
	courseIDs []uuid.UUID,
	eventPayload map[string]any,
) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("repository", "EnrollmentRepository"),
		zap.String("method", "CancelProgramWithEvent"),
	)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Lock the program row — ensures only one concurrent request can cancel
	_, err = qtx.LockPendingProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		// Program already cancelled/approved by another request
		return fmt.Errorf("%w: program not found or not pending: %v", sharedErrors.ErrNotFound, err)
	}

	// Decrement enrollment counts
	for _, courseID := range courseIDs {
		rowsAffected, err := qtx.DecrementEnrollment(ctx, utils.UUIDToPgtype(courseID))
		if err != nil {
			return fmt.Errorf("%w: failed to decrement enrollment: %v", sharedErrors.ErrQueryFailed, err)
		}
		if rowsAffected == 0 {
			log.Warn("decrement enrollment affected 0 rows — possible counter inconsistency",
				zap.String("course_id", courseID.String()),
				zap.String("program_id", programID.String()),
				zap.String("context", "cancel"),
			)
		}
	}

	// Delete program (CASCADE deletes program_courses)
	err = qtx.DeleteEnrollmentProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete program: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal cancelled event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType:   events.EventEnrollmentProgramCancelled,
		AggregateID: utils.UUIDToPgtype(programID),
		Payload:     payload,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return nil
}
