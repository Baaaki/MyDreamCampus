package repository

import (
	"context"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistrationRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewRegistrationRepository(pool *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *RegistrationRepository) CreateRegistration(ctx context.Context, arg db.CreateRegistrationParams) (db.GradesStudentCourseRegistration, error) {
	return r.queries.CreateRegistration(ctx, arg)
}

// DeleteProgramRegistrations takes the student off the courses of a program
// that was withdrawn or replaced, and returns how many rows it removed.
func (r *RegistrationRepository) DeleteProgramRegistrations(ctx context.Context, programID, studentID uuid.UUID, courseIDs []uuid.UUID) (int64, error) {
	return r.queries.DeleteProgramRegistrations(ctx, db.DeleteProgramRegistrationsParams{
		StudentID: studentID,
		CourseIds: courseIDs,
		ProgramID: utils.UUIDToPgtype(programID),
	})
}

func (r *RegistrationRepository) StudentHasGradesInSemester(ctx context.Context, studentID uuid.UUID, semester string) (bool, error) {
	return r.queries.StudentHasGradesInSemester(ctx, db.StudentHasGradesInSemesterParams{
		StudentID: studentID,
		Semester:  semester,
	})
}

func (r *RegistrationRepository) GetRegistrationByID(ctx context.Context, id uuid.UUID) (db.GetRegistrationByIDRow, error) {
	return r.queries.GetRegistrationByID(ctx, id)
}

func (r *RegistrationRepository) GetRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) ([]db.GetRegistrationsByCourseRow, error) {
	return r.queries.GetRegistrationsByCourse(ctx, courseID)
}

func (r *RegistrationRepository) GetRegistrationsByIDs(ctx context.Context, ids []uuid.UUID) ([]db.GetRegistrationsByIDsRow, error) {
	return r.queries.GetRegistrationsByIDs(ctx, ids)
}

func (r *RegistrationRepository) MarkAttendanceFailed(ctx context.Context, arg db.MarkAttendanceFailedParams) error {
	return r.queries.MarkAttendanceFailed(ctx, arg)
}

func (r *RegistrationRepository) CountRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) (int64, error) {
	return r.queries.CountRegistrationsByCourse(ctx, courseID)
}

func (r *RegistrationRepository) CountEligibleRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) (int64, error) {
	return r.queries.CountEligibleRegistrationsByCourse(ctx, courseID)
}

func (r *RegistrationRepository) DeleteRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) error {
	return r.queries.DeleteRegistrationsByCourse(ctx, courseID)
}

func (r *RegistrationRepository) GetActiveRegistrationsByStudent(ctx context.Context, studentID uuid.UUID) ([]db.GetActiveRegistrationsByStudentRow, error) {
	return r.queries.GetActiveRegistrationsByStudent(ctx, studentID)
}

// WithTx returns a copy of the repository whose queries run inside tx —
// required by AutoFinalize so its multi-table writes commit atomically.
func (r *RegistrationRepository) WithTx(tx pgx.Tx) *RegistrationRepository {
	return &RegistrationRepository{queries: r.queries.WithTx(tx), pool: r.pool}
}
