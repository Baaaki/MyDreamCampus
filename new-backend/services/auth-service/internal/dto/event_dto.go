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

// UserUpdatedData represents the data payload for user updates.
// Supports both canonical `id` + `changed_fields` (mixed-type map, e.g.
// numeric class_level) and staff-service's `staff_id` + top-level fields.
type UserUpdatedData struct {
	ID            string         `json:"id"`
	StaffID       string         `json:"staff_id,omitempty"`
	ChangedFields map[string]any `json:"changed_fields,omitempty"`
	Email         *string        `json:"email,omitempty"`
	Department    *string        `json:"department,omitempty"`
}

// EffectiveID returns ID if set, otherwise falls back to StaffID.
func (d UserUpdatedData) EffectiveID() string {
	if d.ID != "" {
		return d.ID
	}
	return d.StaffID
}

// UserDeactivatedEvent represents student.deactivated and staff.deactivated events
type UserDeactivatedEvent struct {
	BaseEvent
	Data UserDeactivatedData `json:"data"`
}

// UserDeactivatedData represents the data payload for user deactivation.
// Supports both `id` (student.deactivated) and `staff_id` (staff.deactivated).
type UserDeactivatedData struct {
	ID        string    `json:"id"`
	StaffID   string    `json:"staff_id,omitempty"`
	IsActive  bool      `json:"is_active"`
	DeletedAt time.Time `json:"deleted_at"`
}

// EffectiveID returns ID if set, otherwise falls back to StaffID.
func (d UserDeactivatedData) EffectiveID() string {
	if d.ID != "" {
		return d.ID
	}
	return d.StaffID
}
