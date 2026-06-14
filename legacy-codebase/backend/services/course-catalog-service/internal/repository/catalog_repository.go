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

type CatalogRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// WithTx returns a new repository instance that uses the given transaction
func (r *CatalogRepository) WithTx(tx pgx.Tx) *CatalogRepository {
	return &CatalogRepository{
		queries: db.New(tx),
		pool:    r.pool,
	}
}

// GetCourseByCourseCode retrieves a course by its course code
func (r *CatalogRepository) GetCourseByCourseCode(ctx context.Context, courseCode string) (db.CourseCatalog, error) {
	course, err := r.queries.GetCourseByCourseCode(ctx, courseCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.CourseCatalog{}, fmt.Errorf("%w: course with code %s not found", catalogErrors.ErrCourseNotFoundRepo, courseCode)
		}
		return db.CourseCatalog{}, fmt.Errorf("%w: failed to get course by code: %v", sharedErrors.ErrQueryFailed, err)
	}
	return course, nil
}

// GetCourseByID retrieves a course by ID
func (r *CatalogRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (db.CourseCatalog, error) {
	course, err := r.queries.GetCourseByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.CourseCatalog{}, fmt.Errorf("%w: course with id %s not found", catalogErrors.ErrCourseNotFoundRepo, id)
		}
		return db.CourseCatalog{}, fmt.Errorf("%w: failed to get course: %v", sharedErrors.ErrQueryFailed, err)
	}
	return course, nil
}

// ListCourses retrieves courses with filtering and pagination
func (r *CatalogRepository) ListCourses(ctx context.Context, params db.ListCoursesParams) ([]db.ListCoursesRow, error) {
	courses, err := r.queries.ListCourses(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

// CountCourses counts total courses matching filters
func (r *CatalogRepository) CountCourses(ctx context.Context, params db.CountCoursesParams) (int64, error) {
	count, err := r.queries.CountCourses(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count courses: %v", sharedErrors.ErrQueryFailed, err)
	}
	return count, nil
}

// GetCoursesByIDs retrieves multiple courses by their IDs (for prerequisite validation)
func (r *CatalogRepository) GetCoursesByIDs(ctx context.Context, ids []uuid.UUID) ([]db.GetCoursesByIDsRow, error) {
	// Convert uuid.UUID slice to pgtype.UUID slice
	pgtypeIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgtypeIDs[i] = utils.UUIDToPgtype(id)
	}

	courses, err := r.queries.GetCoursesByIDs(ctx, pgtypeIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get courses by IDs: %v", sharedErrors.ErrQueryFailed, err)
	}
	return courses, nil
}

// CreateCourse creates a new course
func (r *CatalogRepository) CreateCourse(ctx context.Context, params db.CreateCourseParams) (db.CourseCatalog, error) {
	course, err := r.queries.CreateCourse(ctx, params)
	if err != nil {
		// Check for duplicate course_code constraint violation
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) {
			if pgxErr.Code == "23505" { // unique_violation
				return db.CourseCatalog{}, fmt.Errorf("%w: course code already exists", catalogErrors.ErrCourseExistsRepo)
			}
		}
		return db.CourseCatalog{}, fmt.Errorf("%w: failed to create course: %v", sharedErrors.ErrQueryFailed, err)
	}
	return course, nil
}

// UpdateCourse updates an existing course
func (r *CatalogRepository) UpdateCourse(ctx context.Context, params db.UpdateCourseParams) (db.CourseCatalog, error) {
	course, err := r.queries.UpdateCourse(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.CourseCatalog{}, fmt.Errorf("%w: course not found for update", catalogErrors.ErrCourseNotFoundRepo)
		}
		return db.CourseCatalog{}, fmt.Errorf("%w: failed to update course: %v", sharedErrors.ErrQueryFailed, err)
	}
	return course, nil
}

// ==================== JSON Helper Functions ====================

// PrerequisitesToJSON converts prerequisites slice to JSONB bytes
func PrerequisitesToJSON(prerequisites []dto.Prerequisite) ([]byte, error) {
	if len(prerequisites) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(prerequisites)
}

// JSONToPrerequisites converts JSONB bytes to prerequisites slice
func JSONToPrerequisites(data []byte) ([]dto.Prerequisite, error) {
	var prerequisites []dto.Prerequisite
	if len(data) == 0 || string(data) == "null" {
		return []dto.Prerequisite{}, nil
	}
	if err := json.Unmarshal(data, &prerequisites); err != nil {
		return nil, err
	}
	return prerequisites, nil
}

// CoordinatorToJSON converts CourseCoordinator to JSONB bytes
func CoordinatorToJSON(coordinator *dto.CourseCoordinator) ([]byte, error) {
	if coordinator == nil {
		return nil, nil
	}
	return json.Marshal(coordinator)
}

// JSONToCoordinator converts JSONB bytes to CourseCoordinator
func JSONToCoordinator(data []byte) (*dto.CourseCoordinator, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var coordinator dto.CourseCoordinator
	if err := json.Unmarshal(data, &coordinator); err != nil {
		return nil, err
	}
	return &coordinator, nil
}

// WeeklyTopicsToJSON converts WeeklyTopic slice to JSONB bytes
func WeeklyTopicsToJSON(topics []dto.WeeklyTopic) ([]byte, error) {
	if len(topics) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(topics)
}

// JSONToWeeklyTopics converts JSONB bytes to WeeklyTopic slice
func JSONToWeeklyTopics(data []byte) ([]dto.WeeklyTopic, error) {
	var topics []dto.WeeklyTopic
	if len(data) == 0 || string(data) == "null" {
		return []dto.WeeklyTopic{}, nil
	}
	if err := json.Unmarshal(data, &topics); err != nil {
		return nil, err
	}
	return topics, nil
}

// StringSliceToJSON converts string slice to JSONB bytes
func StringSliceToJSON(items []string) ([]byte, error) {
	if len(items) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(items)
}

// JSONToStringSlice converts JSONB bytes to string slice
func JSONToStringSlice(data []byte) ([]string, error) {
	var items []string
	if len(data) == 0 || string(data) == "null" {
		return []string{}, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
