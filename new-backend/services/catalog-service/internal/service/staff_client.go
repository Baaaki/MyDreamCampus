package service

import (
	"context"

	"github.com/google/uuid"
)

// StaffClient is the contract the catalog services use to look up staff
// info, implemented by HTTPStaffClient against the staff service.
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
