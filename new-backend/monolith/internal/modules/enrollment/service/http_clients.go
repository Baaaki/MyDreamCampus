package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	catalogDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"
	studentDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/student/dto"
	studentErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/student/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/google/uuid"
)

// listPageSize matches the in-process adapter's "give me everything" limit.
const listPageSize = 1000

// HTTPStudentClient implements StudentClient over internal REST.
type HTTPStudentClient struct {
	base *client.Base
}

func NewHTTPStudentClient(base *client.Base) *HTTPStudentClient {
	return &HTTPStudentClient{base: base}
}

func (c *HTTPStudentClient) GetStudentByID(ctx context.Context, id uuid.UUID) (studentDTO.StudentResponse, error) {
	var resp studentDTO.StudentResponse
	if err := c.base.Get(ctx, "/internal/students/"+url.PathEscape(id.String()), &resp); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			// Same AppError the in-process path returns, so the handler still
			// answers 404 STUDENT_NOT_FOUND.
			return studentDTO.StudentResponse{}, studentErrors.ErrStudentNotFound
		}
		return studentDTO.StudentResponse{}, err
	}
	return resp, nil
}

func (c *HTTPStudentClient) GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]studentDTO.StudentResponse, error) {
	query := url.Values{"advisor_id": {advisorID.String()}}

	var resp studentDTO.MyAdviseesResponse
	if err := c.base.Get(ctx, "/internal/students?"+query.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Students, nil
}

// HTTPCourseCatalogClient implements CourseCatalogClient over internal REST.
type HTTPCourseCatalogClient struct {
	base *client.Base
}

func NewHTTPCourseCatalogClient(base *client.Base) *HTTPCourseCatalogClient {
	return &HTTPCourseCatalogClient{base: base}
}

func (c *HTTPCourseCatalogClient) GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]catalogDTO.SemesterCourseListItem, error) {
	query := url.Values{
		"semester":    {semester},
		"department":  {department},
		"class_level": {strconv.Itoa(int(classLevel))},
		"page":        {"1"},
		"limit":       {strconv.Itoa(listPageSize)},
	}

	var resp catalogDTO.ListSemesterCoursesResponse
	if err := c.base.Get(ctx, "/internal/semester-courses?"+query.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *HTTPCourseCatalogClient) GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]catalogDTO.SemesterCourseResponse, error) {
	query := url.Values{"semester": {semester}}.Encode()

	var res []catalogDTO.SemesterCourseResponse
	for _, id := range ids {
		var course catalogDTO.SemesterCourseResponse
		if err := c.base.Get(ctx, "/internal/semester-courses/"+url.PathEscape(id.String())+"?"+query, &course); err != nil {
			return nil, fmt.Errorf("course not found: %w", err)
		}
		res = append(res, course)
	}
	return res, nil
}

var (
	_ StudentClient       = (*HTTPStudentClient)(nil)
	_ CourseCatalogClient = (*HTTPCourseCatalogClient)(nil)
)
