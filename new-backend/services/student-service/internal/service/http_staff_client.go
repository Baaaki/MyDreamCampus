package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/google/uuid"
)

// HTTPStaffClient reaches the staff service over internal REST. Error
// surface matches InProcessStaffClient exactly — same three rejection
// reasons, same wording — so StudentService behaves identically either way.
type HTTPStaffClient struct {
	base *client.Base
}

func NewHTTPStaffClient(base *client.Base) *HTTPStaffClient {
	return &HTTPStaffClient{base: base}
}

// staffMember is the slice of staff's StaffResponse the student module
// reads. Kept local so the split does not leave student importing staff's
// dto package.
type staffMember struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

type staffListResponse struct {
	Data []staffMember `json:"data"`
}

// GetAdvisorInfo validates the advisor and returns their full name.
func (c *HTTPStaffClient) GetAdvisorInfo(ctx context.Context, advisorID uuid.UUID) (*AdvisorDetails, error) {
	var resp staffMember
	if err := c.base.Get(ctx, "/internal/staff/"+url.PathEscape(advisorID.String()), &resp); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return nil, fmt.Errorf("advisor not found")
		}
		return nil, fmt.Errorf("staff lookup failed: %w", err)
	}
	if resp.Role != "teacher" {
		return nil, fmt.Errorf("staff is not a teacher")
	}
	if resp.Status != "active" {
		return nil, fmt.Errorf("advisor is not active")
	}
	return &AdvisorDetails{
		ID:   resp.ID,
		Name: resp.FirstName + " " + resp.LastName,
	}, nil
}

// ValidateAdvisor preserves the legacy alias used by some callers.
func (c *HTTPStaffClient) ValidateAdvisor(ctx context.Context, advisorID uuid.UUID) error {
	_, err := c.GetAdvisorInfo(ctx, advisorID)
	return err
}

// GetInstructorsByDepartment returns the active instructor IDs of a department.
func (c *HTTPStaffClient) GetInstructorsByDepartment(ctx context.Context, department string) ([]uuid.UUID, error) {
	query := url.Values{"department": {department}, "role": {"instructor"}}

	var resp staffListResponse
	if err := c.base.Get(ctx, "/internal/staff?"+query.Encode(), &resp); err != nil {
		return nil, fmt.Errorf("staff instructor lookup failed: %w", err)
	}

	out := make([]uuid.UUID, 0, len(resp.Data))
	for _, s := range resp.Data {
		id, parseErr := uuid.Parse(s.ID)
		if parseErr != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

var _ StaffServiceInterface = (*HTTPStaffClient)(nil)
