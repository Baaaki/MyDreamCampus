package service

import (
	"context"
	"errors"
	"net/url"

	catalogErrors "github.com/baaaki/mydreamcampus/catalog/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/google/uuid"
)

// HTTPStaffClient reaches the staff service over internal REST. It is the
// same contract as InProcessStaffClient and maps the same errors — a 404
// must become ErrInstructorNotFound, or handlers would answer 500 where they
// used to answer 404.
type HTTPStaffClient struct {
	base *client.Base
}

func NewHTTPStaffClient(base *client.Base) *HTTPStaffClient {
	return &HTTPStaffClient{base: base}
}

// staffMember is the slice of staff's StaffResponse this module reads.
// Declared here rather than imported so the split does not leave catalog
// depending on staff's dto package.
type staffMember struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Department string `json:"department"`
	Status     string `json:"status"`
}

type staffListResponse struct {
	Data []staffMember `json:"data"`
}

// GetInstructor validates that the staff member exists, is active, and
// belongs to the requested department.
func (c *HTTPStaffClient) GetInstructor(ctx context.Context, instructorID uuid.UUID, department string) (*InstructorInfo, error) {
	var resp staffMember
	if err := c.base.Get(ctx, "/internal/staff/"+url.PathEscape(instructorID.String()), &resp); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return nil, catalogErrors.ErrInstructorNotFound
		}
		return nil, err
	}

	if resp.Status != "active" {
		return nil, catalogErrors.ErrInstructorNotActive
	}
	if resp.Department != department {
		return nil, catalogErrors.ErrInstructorNotInDepartment
	}
	id, err := uuid.Parse(resp.ID)
	if err != nil {
		return nil, catalogErrors.ErrInstructorNotFound
	}
	return &InstructorInfo{
		ID:         id,
		FirstName:  resp.FirstName,
		LastName:   resp.LastName,
		FullName:   resp.FirstName + " " + resp.LastName,
		Department: resp.Department,
		Status:     resp.Status,
	}, nil
}

// GetInstructorsByDepartment returns the active instructors of a department.
func (c *HTTPStaffClient) GetInstructorsByDepartment(ctx context.Context, department string) ([]InstructorInfo, error) {
	query := url.Values{"department": {department}, "role": {"instructor"}}

	var resp staffListResponse
	if err := c.base.Get(ctx, "/internal/staff?"+query.Encode(), &resp); err != nil {
		return nil, err
	}

	out := make([]InstructorInfo, 0, len(resp.Data))
	for _, s := range resp.Data {
		id, parseErr := uuid.Parse(s.ID)
		if parseErr != nil {
			continue
		}
		out = append(out, InstructorInfo{
			ID:         id,
			FirstName:  s.FirstName,
			LastName:   s.LastName,
			FullName:   s.FirstName + " " + s.LastName,
			Department: s.Department,
			Status:     s.Status,
		})
	}
	return out, nil
}

var _ StaffClient = (*HTTPStaffClient)(nil)
