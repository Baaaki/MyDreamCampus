package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	catalogService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	catalogDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"
	studentService "github.com/baaaki/mydreamcampus/monolith/internal/modules/student/service"
	studentDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/student/dto"
)

// StudentClient defines the interface for communicating with the Student module
type StudentClient interface {
	GetStudentByID(ctx context.Context, id uuid.UUID) (studentDTO.StudentResponse, error)
	GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]studentDTO.StudentResponse, error)
}

// CourseCatalogClient defines the interface for communicating with the Course Catalog module
type CourseCatalogClient interface {
	GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]catalogDTO.SemesterCourseListItem, error)
	GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]catalogDTO.SemesterCourseResponse, error)
}

// InProcessStudentClient implements StudentClient by directly calling the Student module's public service
type InProcessStudentClient struct {
	svc *studentService.StudentService
}

func NewInProcessStudentClient(svc *studentService.StudentService) *InProcessStudentClient {
	return &InProcessStudentClient{svc: svc}
}

func (c *InProcessStudentClient) GetStudentByID(ctx context.Context, id uuid.UUID) (studentDTO.StudentResponse, error) {
	return c.svc.GetStudentByID(ctx, id.String())
}

func (c *InProcessStudentClient) GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]studentDTO.StudentResponse, error) {
	resp, err := c.svc.ListStudentsByAdvisor(ctx, advisorID)
	if err != nil {
		return nil, err
	}
	return resp.Students, nil
}

// InProcessCourseCatalogClient implements CourseCatalogClient by directly calling the Course Catalog module's public service
type InProcessCourseCatalogClient struct {
	svc *catalogService.SemesterService
}

func NewInProcessCourseCatalogClient(svc *catalogService.SemesterService) *InProcessCourseCatalogClient {
	return &InProcessCourseCatalogClient{svc: svc}
}

func (c *InProcessCourseCatalogClient) GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]catalogDTO.SemesterCourseListItem, error) {
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

func (c *InProcessCourseCatalogClient) GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]catalogDTO.SemesterCourseResponse, error) {
	var res []catalogDTO.SemesterCourseResponse
	for _, id := range ids {
		course, err := c.svc.GetSemesterCourseByID(ctx, semester, id.String())
		if err != nil {
			return nil, fmt.Errorf("course not found: %w", err)
		}
		res = append(res, course)
	}
	return res, nil
}
