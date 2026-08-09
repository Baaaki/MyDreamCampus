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

// catalogPageSize is the largest `limit` catalog's query binding accepts
// (max=100); asking for more is a 400, not a bigger page. The in-process
// adapter passed one "give me everything" limit straight to the service and
// never met that ceiling, so the HTTP client pages to return the same full
// list rather than silently stopping at the first hundred.
const catalogPageSize = 100

// maxCatalogPages bounds the walk. Unreachable in practice — it is 5000
// courses for a single department and class level — so hitting it means the
// provider is not shrinking its pages, and truncating there without a word
// is the very failure this paging exists to avoid.
const maxCatalogPages = 50

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
	var all []contracts.SemesterCourseListItem

	for page := 1; page <= maxCatalogPages; page++ {
		query := url.Values{
			"semester":    {semester},
			"department":  {department},
			"class_level": {strconv.Itoa(int(classLevel))},
			"page":        {strconv.Itoa(page)},
			"limit":       {strconv.Itoa(catalogPageSize)},
		}

		var resp semesterCourseList
		if err := c.base.Get(ctx, "/internal/semester-courses?"+query.Encode(), &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)

		// A short page is the last page.
		if len(resp.Data) < catalogPageSize {
			return all, nil
		}
	}

	return nil, fmt.Errorf("catalog available courses: more than %d pages for %s/%d",
		maxCatalogPages, department, classLevel)
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
