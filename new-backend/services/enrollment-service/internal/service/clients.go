package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/shared/contracts"
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
