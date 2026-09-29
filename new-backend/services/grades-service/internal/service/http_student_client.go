package service

import (
	"context"
	"errors"
	"net/url"

	gradeErrors "github.com/baaaki/mydreamcampus/grades/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/google/uuid"
)

// HTTPStudentClient implements StudentClient over internal REST.
type HTTPStudentClient struct {
	base *client.Base
}

func NewHTTPStudentClient(base *client.Base) *HTTPStudentClient {
	return &HTTPStudentClient{base: base}
}

// GetStudentByID maps student's 404 to grades' own ErrStudentNotFound.
// Anything else keeps client.ErrUnavailable in the chain, which the caller
// must refuse on: a student service it cannot reach proves no advisor.
func (c *HTTPStudentClient) GetStudentByID(ctx context.Context, id uuid.UUID) (contracts.StudentResponse, error) {
	var resp contracts.StudentResponse
	if err := c.base.Get(ctx, "/internal/students/"+url.PathEscape(id.String()), &resp); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return contracts.StudentResponse{}, gradeErrors.ErrStudentNotFound
		}
		return contracts.StudentResponse{}, err
	}
	return resp, nil
}

var _ StudentClient = (*HTTPStudentClient)(nil)
