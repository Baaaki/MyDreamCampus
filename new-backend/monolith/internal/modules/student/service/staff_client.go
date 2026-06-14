package service

import (
	"context"
	"errors"
	"fmt"

	staffErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/errors"
	staffService "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/service"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/google/uuid"
)

// StaffClient calls into the staff module via in-process Go calls.
// Replaces the original microservice's HTTP client (staff-service /internal
// endpoints). Same interface (StaffServiceInterface in student_service.go)
// so the StudentService construction does not change.
type StaffClient struct {
	staff *staffService.StaffService
}

// NewStaffClient adapts the staff module's StaffService for the student
// service. Plan section 8 strategy 1 — cross-module reads via the public
// in-process Service handle.
func NewStaffClient(staff *staffService.StaffService) *StaffClient {
	return &StaffClient{staff: staff}
}

// AdvisorDetails contains the advisor information StudentService needs.
type AdvisorDetails struct {
	ID   string
	Name string
}

// GetAdvisorInfo validates the advisor and returns their full name.
// Returns "advisor not found" / "staff is not a teacher" / "advisor is
// not active" sentinel errors so callers can distinguish — same surface
// the HTTP client exposed before.
func (c *StaffClient) GetAdvisorInfo(ctx context.Context, advisorID uuid.UUID) (*AdvisorDetails, error) {
	resp, err := c.staff.GetStaffByID(ctx, advisorID.String())
	if err != nil {
		if errors.Is(err, staffErrors.ErrStaffNotFound) {
			return nil, fmt.Errorf("advisor not found")
		}
		if sharedErrors.Is(err, sharedErrors.ErrInvalidID) {
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
func (c *StaffClient) ValidateAdvisor(ctx context.Context, advisorID uuid.UUID) error {
	_, err := c.GetAdvisorInfo(ctx, advisorID)
	return err
}

// GetInstructorsByDepartment returns the active instructor IDs for a
// department. Backed by the staff module's GetInstructorsByDepartment.
func (c *StaffClient) GetInstructorsByDepartment(ctx context.Context, department string) ([]uuid.UUID, error) {
	list, err := c.staff.GetInstructorsByDepartment(ctx, department)
	if err != nil {
		return nil, fmt.Errorf("staff instructor lookup failed: %w", err)
	}
	out := make([]uuid.UUID, 0, len(list.Data))
	for _, s := range list.Data {
		id, parseErr := uuid.Parse(s.ID)
		if parseErr != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

// Compile-time check — drift between the staff module's API and
// StudentService surfaces at build time, not runtime.
var _ StaffServiceInterface = (*StaffClient)(nil)
