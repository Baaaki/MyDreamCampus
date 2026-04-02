package dto

import (
	"time"

	"github.com/google/uuid"
)

// Event represents a generic event structure
type Event struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// StudentCreatedEvent represents student.created event payload
type StudentCreatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      StudentCreatedData `json:"data"`
}

// StudentCreatedData represents the data in student.created event
type StudentCreatedData struct {
	ID             uuid.UUID  `json:"id"`
	StudentNumber  string     `json:"student_number"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	Email          string     `json:"email"`
	Faculty        string     `json:"faculty"`
	Department     string     `json:"department"`
	EnrollmentYear int        `json:"enrollment_year"`
	ClassLevel     int16      `json:"class_level"`
	AdvisorID      *uuid.UUID `json:"advisor_id,omitempty"`
	Status         string     `json:"status"`
}

// StudentUpdatedEvent represents student.updated event payload
type StudentUpdatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      StudentUpdatedData `json:"data"`
}

// StudentUpdatedData represents the data in student.updated event
type StudentUpdatedData struct {
	ID            uuid.UUID              `json:"id"`
	StudentNumber string                 `json:"student_number"`
	ChangedFields map[string]interface{} `json:"changed_fields"`
}

// StudentDeactivatedEvent represents student.deactivated event payload
type StudentDeactivatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      StudentDeactivatedData `json:"data"`
}

// StudentDeactivatedData represents the data in student.deactivated event
type StudentDeactivatedData struct {
	ID            uuid.UUID `json:"id"`
	StudentNumber string    `json:"student_number"`
	IsActive      bool      `json:"is_active"`
	DeletedAt     time.Time `json:"deleted_at"`
}

// StaffDeactivatedEvent represents staff.deactivated event payload (inbound)
type StaffDeactivatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      StaffDeactivatedData `json:"data"`
}

// StaffDeactivatedData represents the data in staff.deactivated event
type StaffDeactivatedData struct {
	StaffID   uuid.UUID `json:"staff_id"`
	IsActive  bool      `json:"is_active"`
	DeletedAt time.Time `json:"deleted_at"`
}
