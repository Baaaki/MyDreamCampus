package contracts

import "time"

// StudentResponse is the student record as every other service sees it.
// Served by student on GET /internal/students(/:id), read by enrollment.
type StudentResponse struct {
	ID             string    `json:"id"`
	StudentNumber  string    `json:"student_number"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Email          string    `json:"email"`
	Faculty        string    `json:"faculty"`
	Department     string    `json:"department"`
	EnrollmentYear int       `json:"enrollment_year"`
	ClassLevel     int16     `json:"class_level"`
	AdvisorID      *string   `json:"advisor_id,omitempty"`
	AdvisorName    *string   `json:"advisor_name,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
