package dto

import "github.com/google/uuid"

// StaffInstructor represents an instructor from Staff Service
type StaffInstructor struct {
	ID         uuid.UUID `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Department string    `json:"department"`
}

// StaffInstructorsResponse represents the response from Staff Service
type StaffInstructorsResponse struct {
	Data []StaffInstructor `json:"data"`
}
