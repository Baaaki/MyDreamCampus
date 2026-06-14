package dto

import "time"

// BaseEvent represents the base structure for all events
type BaseEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

// StudentCreatedEvent represents the student.created event
type StudentCreatedEvent struct {
	BaseEvent
	Data StudentCreatedData `json:"data"`
}

// StudentCreatedData represents the data payload for student.created
type StudentCreatedData struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Department string `json:"department"`
}

// StaffCreatedEvent represents the staff.created event
type StaffCreatedEvent struct {
	BaseEvent
	Data StaffCreatedData `json:"data"`
}

// StaffCreatedData represents the data payload for staff.created
type StaffCreatedData struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Department string `json:"department"`
}

// UserUpdatedEvent represents student.updated and staff.updated events
type UserUpdatedEvent struct {
	BaseEvent
	Data UserUpdatedData `json:"data"`
}

// UserUpdatedData represents the data payload for user updates
type UserUpdatedData struct {
	ID            string            `json:"id"`
	ChangedFields map[string]string `json:"changed_fields"`
}

// UserDeactivatedEvent represents student.deactivated and staff.deactivated events
type UserDeactivatedEvent struct {
	BaseEvent
	Data UserDeactivatedData `json:"data"`
}

// UserDeactivatedData represents the data payload for user deactivation
type UserDeactivatedData struct {
	ID         string    `json:"id"`
	IsActive   bool      `json:"is_active"`
	DeletedAt  time.Time `json:"deleted_at"`
}
