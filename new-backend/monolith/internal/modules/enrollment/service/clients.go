package service

import (
	"context"
	"fmt"

	catalogDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"
	catalogService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
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

// InProcessCourseCatalogClient implements CourseCatalogClient by directly calling the Course Catalog module's public service
type InProcessCourseCatalogClient struct {
	svc *catalogService.SemesterService
}

func NewInProcessCourseCatalogClient(svc *catalogService.SemesterService) *InProcessCourseCatalogClient {
	return &InProcessCourseCatalogClient{svc: svc}
}

func (c *InProcessCourseCatalogClient) GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]contracts.SemesterCourseListItem, error) {
	req := catalogDTO.ListSemesterCoursesRequest{
		Department: &department,
		ClassLevel: &classLevel,
	}
	req.Limit = 1000 // Get all
	req.Page = 1

	resp, err := c.svc.ListSemesterCourses(ctx, semester, req)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *InProcessCourseCatalogClient) GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]contracts.SemesterCourseResponse, error) {
	var res []contracts.SemesterCourseResponse
	for _, id := range ids {
		course, err := c.svc.GetSemesterCourseByID(ctx, semester, id.String())
		if err != nil {
			return nil, fmt.Errorf("course not found: %w", err)
		}
		res = append(res, course)
	}
	return res, nil
}

// Compile-time assertions — the in-process and HTTP clients must stay
// interchangeable.
var _ CourseCatalogClient = (*InProcessCourseCatalogClient)(nil)
