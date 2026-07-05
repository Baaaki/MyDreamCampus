package repository

import (
	"context"
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/db"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PassedPrerequisitesRepository is the local projection of grades'
// grade.student.prerequisite.passed events. Enrollment validates against
// this table instead of asking the grades module, so the check survives a
// future split into separate services.
type PassedPrerequisitesRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewPassedPrerequisitesRepository(pool *pgxpool.Pool) *PassedPrerequisitesRepository {
	return &PassedPrerequisitesRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *PassedPrerequisitesRepository) UpsertPassedPrerequisite(ctx context.Context, studentID, courseID uuid.UUID, courseCode, semester, gradePoint string) error {
	err := r.queries.UpsertPassedPrerequisite(ctx, db.UpsertPassedPrerequisiteParams{
		StudentID:  utils.UUIDToPgtype(studentID),
		CourseID:   utils.UUIDToPgtype(courseID),
		CourseCode: courseCode,
		Semester:   semester,
		GradePoint: gradePoint,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to upsert passed prerequisite: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

func (r *PassedPrerequisitesRepository) HasPassedPrerequisite(ctx context.Context, studentID uuid.UUID, courseCode string) (bool, error) {
	passed, err := r.queries.HasPassedPrerequisite(ctx, db.HasPassedPrerequisiteParams{
		StudentID:  utils.UUIDToPgtype(studentID),
		CourseCode: courseCode,
	})
	if err != nil {
		return false, fmt.Errorf("%w: failed to check passed prerequisite: %v", sharedErrors.ErrQueryFailed, err)
	}
	return passed, nil
}
