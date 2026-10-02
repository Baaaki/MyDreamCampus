package service

import (
	"context"
	"errors"

	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/google/uuid"
)

// StudentClient defines the interface for communicating with the Student module
type StudentClient interface {
	GetStudentByID(ctx context.Context, id uuid.UUID) (contracts.StudentResponse, error)
	GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]contracts.StudentResponse, error)
}

// CourseCatalogClient defines the interface for communicating with the Course Catalog module
type CourseCatalogClient interface {
	GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]contracts.SemesterCourseListItem, error)
	GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]contracts.SemesterCourseResponse, error)
}

// upstreamError maps a failed call to student or catalog. A peer that cannot
// answer (down, 5xx, open breaker) is a 503 the caller may retry: the request
// itself was fine, so neither the 404 nor the 500 it used to surface as fits.
func upstreamError(err error) error {
	if errors.Is(err, client.ErrUnavailable) {
		return sharedErrors.Wrap(sharedErrors.ErrServiceUnavailable, err)
	}
	return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
}

// lookupStudent reads a student from the student service: a missing record is
// STUDENT_NOT_FOUND, anything else goes through upstreamError.
func lookupStudent(ctx context.Context, students StudentClient, id uuid.UUID) (contracts.StudentResponse, error) {
	student, err := students.GetStudentByID(ctx, id)
	if err != nil {
		if errors.Is(err, serviceErrors.ErrStudentNotFound) {
			return contracts.StudentResponse{}, serviceErrors.ErrStudentNotFound
		}
		return contracts.StudentResponse{}, upstreamError(err)
	}
	return student, nil
}
