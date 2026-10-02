package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc64"
	"slices"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/events"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

type ProgramCourseSnapshot struct {
	CourseID    uuid.UUID
	CourseCode  string
	CourseName  string
	Credits     int16
	MaxCapacity int16
}

var courseLockTable = crc64.MakeTable(crc64.ISO)

// courseLockIDs returns the seat-lock key of every course, ascending and
// without repeats. Taken in request order, two students who pick {A, B} and
// {B, A} each hold one lock and wait for the other's until Postgres breaks
// the deadlock by failing one of them; one global order cannot form a cycle.
func courseLockIDs(courses []ProgramCourseSnapshot) []int64 {
	ids := make([]int64, 0, len(courses))
	for _, course := range courses {
		// The uint64→int64 wraparound is fine: the checksum is only a
		// deterministic lock key, never interpreted as a quantity.
		ids = append(ids, int64(crc64.Checksum(course.CourseID[:], courseLockTable))) // #nosec G115
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

// CreateProgramWithCoursesAndEvent creates enrollment program, courses and outbox event atomically
func (r *EnrollmentRepository) CreateProgramWithCoursesAndEvent(
	ctx context.Context,
	programParams db.CreateEnrollmentProgramParams,
	courses []ProgramCourseSnapshot,
	eventPayload map[string]any,
) (db.EnrollmentProgram, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)

	// Every course is locked before any is counted: a seat counted while
	// another submission can still take it is no seat at all.
	for _, lockID := range courseLockIDs(courses) {
		if err := qtx.LockCourseSeats(ctx, lockID); err != nil {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to acquire advisory lock: %v", sharedErrors.ErrQueryFailed, err)
		}
	}
	for _, course := range courses {
		count, err := qtx.CountActiveCourseEnrollments(ctx, utils.UUIDToPgtype(course.CourseID))
		if err != nil {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to count course enrollments: %v", sharedErrors.ErrQueryFailed, err)
		}
		if count >= int64(course.MaxCapacity) {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: course capacity is full", sharedErrors.ErrConflict)
		}
	}

	// Create enrollment program
	program, err := qtx.CreateEnrollmentProgram(ctx, programParams)
	if err != nil {
		if isUniqueViolation(err) {
			// UNIQUE(student_id, semester): the same student's other
			// submission committed while this one waited on the index.
			return db.EnrollmentProgram{}, serviceErrors.ErrSubmissionConflict
		}
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to create enrollment program: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Add courses to program
	for _, course := range courses {
		_, err := qtx.CreateEnrollmentProgramCourse(ctx, db.CreateEnrollmentProgramCourseParams{
			ProgramID:  program.ID,
			CourseID:   utils.UUIDToPgtype(course.CourseID),
			CourseCode: course.CourseCode,
			CourseName: course.CourseName,
			Credits:    course.Credits,
		})
		if err != nil {
			return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to add course to program: %v", sharedErrors.ErrQueryFailed, err)
		}
	}

	// Create outbox event
	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to marshal submitted event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     events.EventEnrollmentProgramSubmitted,
		RoutingKey:    events.RoutingKeyEnrollmentProgramSubmitted,
		Payload:       payload,
	})
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return program, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" // unique_violation
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

func (r *EnrollmentRepository) GetCoursesByProgramID(ctx context.Context, programID uuid.UUID) ([]db.EnrollmentProgramCourse, error) {
	courses, err := r.queries.GetCoursesByProgramID(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get program courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

func (r *EnrollmentRepository) GetPendingProgramsByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]db.EnrollmentProgram, error) {
	pgStudentIDs := make([]pgtype.UUID, len(studentIDs))
	for i, id := range studentIDs {
		pgStudentIDs[i] = utils.UUIDToPgtype(id)
	}

	programs, err := r.queries.GetPendingProgramsByStudentIDs(ctx, pgStudentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pending programs: %v", sharedErrors.ErrQueryFailed, err)
	}
	return programs, nil
}

func (r *EnrollmentRepository) ApproveProgramWithEvent(
	ctx context.Context,
	programID uuid.UUID,
	eventPayload map[string]any,
) (db.EnrollmentProgram, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)

	// Row lock + status check: of two concurrent approvals only the first
	// finds the program pending, so the approved event is written once.
	if _, err := qtx.LockPendingProgram(ctx, utils.UUIDToPgtype(programID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.EnrollmentProgram{}, serviceErrors.ErrProgramNotPending
		}
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to lock program: %v", sharedErrors.ErrQueryFailed, err)
	}

	program, err := qtx.UpdateProgramStatus(ctx, db.UpdateProgramStatusParams{
		ID: utils.UUIDToPgtype(programID),
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumApproved,
			Valid:                true,
		},
	})
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to approve program: %v", sharedErrors.ErrQueryFailed, err)
	}

	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to marshal approved event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     "enrollment.program.approved",
		RoutingKey:    "enrollment.program.approved",
		Payload:       payload,
	})
	if err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.EnrollmentProgram{}, fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	return program, nil
}

func (r *EnrollmentRepository) RejectProgramWithEventAndLog(
	ctx context.Context,
	programID uuid.UUID,
	rejectionLogParams db.CreateRejectionLogParams,
	eventPayload map[string]any,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)

	_, err = qtx.LockPendingProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: program not found or not pending: %v", sharedErrors.ErrNotFound, err)
	}

	_, err = qtx.CreateRejectionLog(ctx, rejectionLogParams)
	if err != nil {
		return fmt.Errorf("%w: failed to create rejection log: %v", sharedErrors.ErrQueryFailed, err)
	}

	err = qtx.DeleteEnrollmentProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete program: %v", sharedErrors.ErrQueryFailed, err)
	}

	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal rejected event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     events.EventEnrollmentProgramRejected,
		RoutingKey:    events.RoutingKeyEnrollmentProgramRejected,
		Payload:       payload,
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

func (r *EnrollmentRepository) CancelProgramWithEvent(
	ctx context.Context,
	programID uuid.UUID,
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
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)

	_, err = qtx.LockPendingProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: program not found or not pending: %v", sharedErrors.ErrNotFound, err)
	}

	err = qtx.DeleteEnrollmentProgram(ctx, utils.UUIDToPgtype(programID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete program: %v", sharedErrors.ErrQueryFailed, err)
	}

	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal cancelled event payload: %v", sharedErrors.ErrInternal, err)
	}
	_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     events.EventEnrollmentProgramCancelled,
		RoutingKey:    events.RoutingKeyEnrollmentProgramCancelled,
		Payload:       payload,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to create outbox event: %v", sharedErrors.ErrQueryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", sharedErrors.ErrTransactionFailed, err)
	}

	log.Info("Program cancelled successfully", zap.String("program_id", programID.String()))
	return nil
}

func (r *EnrollmentRepository) CountEnrollmentForCourse(ctx context.Context, courseID uuid.UUID) (int, error) {
	count, err := r.queries.CountActiveCourseEnrollments(ctx, utils.UUIDToPgtype(courseID))
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count course enrollments: %v", sharedErrors.ErrQueryFailed, err)
	}
	return int(count), nil
}
