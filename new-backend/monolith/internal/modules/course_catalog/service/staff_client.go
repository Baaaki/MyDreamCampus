package service

import (
	"context"
	"errors"

	catalogErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/errors"
	staffErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/errors"
	staffService "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/service"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/google/uuid"
)

// StaffClient is the contract the catalog services use to look up staff
// info. The HTTP-based implementation is gone; the in-process adapter
// (InProcessStaffClient) calls the staff module directly.
type StaffClient interface {
	GetInstructor(ctx context.Context, instructorID uuid.UUID, department string) (*InstructorInfo, error)
	GetInstructorsByDepartment(ctx context.Context, department string) ([]InstructorInfo, error)
}

// InstructorInfo is the read-only projection of a staff member that the
// catalog services consume — full name pre-computed for display.
type InstructorInfo struct {
	ID         uuid.UUID `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	FullName   string    `json:"-"`
	Department string    `json:"department"`
	Status     string    `json:"status"`
}

// InProcessStaffClient adapts the staff module's StaffService to the
// StaffClient interface. Replaces the HTTP-based HTTPStaffClient now
// that staff lives in the same binary (plan section 8 strategy 1).
type InProcessStaffClient struct {
	staff *staffService.StaffService
}

func NewInProcessStaffClient(staff *staffService.StaffService) *InProcessStaffClient {
	return &InProcessStaffClient{staff: staff}
}

// GetInstructor validates that the staff member exists, is active, and
// belongs to the requested department. Errors are mapped to the catalog
// module's sentinels so existing handler code keeps working.
func (c *InProcessStaffClient) GetInstructor(ctx context.Context, instructorID uuid.UUID, department string) (*InstructorInfo, error) {
	resp, err := c.staff.GetStaffByID(ctx, instructorID.String())
	if err != nil {
		if errors.Is(err, staffErrors.ErrStaffNotFound) {
			return nil, catalogErrors.ErrInstructorNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrInvalidID) {
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

// GetInstructorsByDepartment returns the active instructors for a
// department, with FullName pre-computed.
func (c *InProcessStaffClient) GetInstructorsByDepartment(ctx context.Context, department string) ([]InstructorInfo, error) {
	list, err := c.staff.GetInstructorsByDepartment(ctx, department)
	if err != nil {
		return nil, err
	}
	out := make([]InstructorInfo, 0, len(list.Data))
	for _, s := range list.Data {
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

// Compile-time assertion — drift between staff module and catalog
// expectations surfaces at build time.
var _ StaffClient = (*InProcessStaffClient)(nil)
