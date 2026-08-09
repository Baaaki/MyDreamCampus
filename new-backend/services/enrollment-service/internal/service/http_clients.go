package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	enrollmentErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/google/uuid"
)

// listPageSize matches the in-process adapter's "give me everything" limit.
const listPageSize = 1000

// adviseeList and semesterCourseList are the slices of the providers' paged
// responses this client reads. Declared locally so enrollment does not depend
// on student's or catalog's pagination DTOs.
type adviseeList struct {
	Students []contracts.StudentResponse `json:"students"`
}

type semesterCourseList struct {
	Data []contracts.SemesterCourseListItem `json:"data"`
}

// HTTPStudentClient implements StudentClient over internal REST.
type HTTPStudentClient struct {
	base *client.Base
}

func NewHTTPStudentClient(base *client.Base) *HTTPStudentClient {
	return &HTTPStudentClient{base: base}
}

func (c *HTTPStudentClient) GetStudentByID(ctx context.Context, id uuid.UUID) (contracts.StudentResponse, error) {
	var resp contracts.StudentResponse
	if err := c.base.Get(ctx, "/internal/students/"+url.PathEscape(id.String()), &resp); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			// Enrollment's own sentinel maps to the same 404 STUDENT_NOT_FOUND
			// the in-process path produced through student's AppError, so the
			// handler answer is unchanged.
			return contracts.StudentResponse{}, enrollmentErrors.ErrStudentNotFound
		}
		return contracts.StudentResponse{}, err
	}
	return resp, nil
}

func (c *HTTPStudentClient) GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]contracts.StudentResponse, error) {
	query := url.Values{"advisor_id": {advisorID.String()}}

	var resp adviseeList
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

func (c *HTTPCourseCatalogClient) GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]contracts.SemesterCourseListItem, error) {
	query := url.Values{
		"semester":    {semester},
		"department":  {department},
		"class_level": {strconv.Itoa(int(classLevel))},
		"page":        {"1"},
		"limit":       {strconv.Itoa(listPageSize)},
	}

	var resp semesterCourseList
	if err := c.base.Get(ctx, "/internal/semester-courses?"+query.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *HTTPCourseCatalogClient) GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]contracts.SemesterCourseResponse, error) {
	query := url.Values{"semester": {semester}}.Encode()

	var res []contracts.SemesterCourseResponse
	for _, id := range ids {
		var course contracts.SemesterCourseResponse
		if err := c.base.Get(ctx, "/internal/semester-courses/"+url.PathEscape(id.String())+"?"+query, &course); err != nil {
			return nil, fmt.Errorf("course not found: %w", err)
		}
		res = append(res, course)
	}
	return res, nil
}

var _ StudentClient = (*HTTPStudentClient)(nil)
var _ CourseCatalogClient = (*HTTPCourseCatalogClient)(nil)
