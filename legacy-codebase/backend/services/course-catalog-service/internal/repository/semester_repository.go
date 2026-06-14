package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/dto"
	catalogErrors "github.com/baaaki/mydreamcampus/course-catalog-service/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SemesterRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewSemesterRepository(pool *pgxpool.Pool) *SemesterRepository {
	return &SemesterRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// WithTx returns a new repository instance that uses the given transaction
func (r *SemesterRepository) WithTx(tx pgx.Tx) *SemesterRepository {
	return &SemesterRepository{
		queries: db.New(tx),
		pool:    r.pool,
	}
}

// GetSemesterCourseByID retrieves a semester course by ID and semester
func (r *SemesterRepository) GetSemesterCourseByID(ctx context.Context, id uuid.UUID, semester string) (db.SemesterCourse, error) {
	row, err := r.queries.GetSemesterCourseByID(ctx, db.GetSemesterCourseByIDParams{
		ID:       utils.UUIDToPgtype(id),
		Semester: semester,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.SemesterCourse{}, fmt.Errorf("%w: semester course with id %s not found", catalogErrors.ErrSemesterCourseNotFoundRepo, id)
		}
		return db.SemesterCourse{}, fmt.Errorf("%w: failed to get semester course: %v", sharedErrors.ErrQueryFailed, err)
	}

	return db.SemesterCourse{
		ID:                 row.ID,
		Semester:           row.Semester,
		CourseCode:         row.CourseCode,
		Credits:            row.Credits,
		ClassLevel:         row.ClassLevel,
		InstructorID:       row.InstructorID,
		InstructorFullname: row.InstructorFullname,
		ClassroomLocation:  row.ClassroomLocation,
		MaxCapacity:        row.MaxCapacity,
		AssessmentSchema:   row.AssessmentSchema,
		Prerequisites:      row.Prerequisites,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

// GetSemesterCourseBySemesterAndCode retrieves semester course by semester and course code
func (r *SemesterRepository) GetSemesterCourseBySemesterAndCode(ctx context.Context, semester, courseCode string) (db.SemesterCourse, error) {
	row, err := r.queries.GetSemesterCourseBySemesterAndCode(ctx, db.GetSemesterCourseBySemesterAndCodeParams{
		Semester:   semester,
		CourseCode: courseCode,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not found is valid for existence check - return empty course
			return db.SemesterCourse{}, nil
		}
		return db.SemesterCourse{}, fmt.Errorf("%w: failed to check course opened: %v", sharedErrors.ErrQueryFailed, err)
	}

	return db.SemesterCourse{
		ID:                 row.ID,
		Semester:           row.Semester,
		CourseCode:         row.CourseCode,
		Credits:            row.Credits,
		ClassLevel:         row.ClassLevel,
		InstructorID:       row.InstructorID,
		InstructorFullname: row.InstructorFullname,
		ClassroomLocation:  row.ClassroomLocation,
		MaxCapacity:        row.MaxCapacity,
		AssessmentSchema:   row.AssessmentSchema,
		Prerequisites:      row.Prerequisites,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

// ListSemesterCourses retrieves semester courses with filtering and pagination
func (r *SemesterRepository) ListSemesterCourses(ctx context.Context, params db.ListSemesterCoursesParams) ([]db.ListSemesterCoursesRow, error) {
	courses, err := r.queries.ListSemesterCourses(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list semester courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

// CountSemesterCourses counts total semester courses matching filters
func (r *SemesterRepository) CountSemesterCourses(ctx context.Context, params db.CountSemesterCoursesParams) (int64, error) {
	count, err := r.queries.CountSemesterCourses(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count semester courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return count, nil
}

// CreateSemesterCourse creates a new semester course
func (r *SemesterRepository) CreateSemesterCourse(ctx context.Context, params db.CreateSemesterCourseParams) (db.SemesterCourse, error) {
	row, err := r.queries.CreateSemesterCourse(ctx, params)
	if err != nil {
		// Check for unique constraint violation (course already opened for this semester)
		if isPgUniqueViolation(err) {
			return db.SemesterCourse{}, fmt.Errorf("%w: course already opened for this semester", catalogErrors.ErrCourseAlreadyOpenedRepo)
		}
		return db.SemesterCourse{}, fmt.Errorf("%w: failed to create semester course: %v", sharedErrors.ErrQueryFailed, err)
	}

	return db.SemesterCourse{
		ID:                 row.ID,
		Semester:           row.Semester,
		CourseCode:         row.CourseCode,
		Credits:            row.Credits,
		ClassLevel:         row.ClassLevel,
		InstructorID:       row.InstructorID,
		InstructorFullname: row.InstructorFullname,
		ClassroomLocation:  row.ClassroomLocation,
		MaxCapacity:        row.MaxCapacity,
		AssessmentSchema:   row.AssessmentSchema,
		Prerequisites:      row.Prerequisites,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

// DeleteSemesterCourse deletes a semester course
func (r *SemesterRepository) DeleteSemesterCourse(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteSemesterCourse(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		return fmt.Errorf("%w: failed to delete semester course: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// Helper: Convert assessment schema to JSONB
func AssessmentSchemaToJSON(assessmentSchema []dto.AssessmentItem) ([]byte, error) {
	return json.Marshal(assessmentSchema)
}

// Helper: Convert JSONB to assessment schema
func JSONToAssessmentSchema(data []byte) ([]dto.AssessmentItem, error) {
	var assessmentSchema []dto.AssessmentItem
	if len(data) == 0 || string(data) == "null" {
		return []dto.AssessmentItem{}, nil
	}
	if err := json.Unmarshal(data, &assessmentSchema); err != nil {
		return nil, err
	}
	return assessmentSchema, nil
}

// Helper: Check if error is a PostgreSQL unique constraint violation
func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// GetTeacherCourses retrieves all courses for a specific instructor
func (r *SemesterRepository) GetTeacherCourses(ctx context.Context, instructorID uuid.UUID, semester pgtype.Text) ([]db.GetTeacherCoursesRow, error) {
	courses, err := r.queries.GetTeacherCourses(ctx, db.GetTeacherCoursesParams{
		InstructorID: utils.UUIDToPgtype(instructorID),
		Semester:     semester,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get teacher courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}
