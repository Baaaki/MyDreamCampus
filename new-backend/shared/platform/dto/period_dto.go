package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreatePeriodRequest is the request body for POST /admin/periods.
type CreatePeriodRequest struct {
	Semester    string     `json:"semester" binding:"required"`
	PeriodStart time.Time  `json:"period_start" binding:"required"`
	PeriodEnd   time.Time  `json:"period_end" binding:"required"`
	CourseID    *uuid.UUID `json:"course_id,omitempty"` // NULL = global, non-nil = course-specific override
}

// UpdatePeriodRequest is the request body for PUT /admin/periods/:id.
type UpdatePeriodRequest struct {
	PeriodEnd *time.Time `json:"period_end,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
}

// PeriodResponse is the response body for period endpoints (with course_id, used by grades service).
type PeriodResponse struct {
	ID          uuid.UUID  `json:"id"`
	Semester    string     `json:"semester"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	CourseID    *uuid.UUID `json:"course_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SimpleCreatePeriodRequest is the request body for services without course-specific overrides.
type SimpleCreatePeriodRequest struct {
	Semester    string    `json:"semester" binding:"required"`
	PeriodStart time.Time `json:"period_start" binding:"required"`
	PeriodEnd   time.Time `json:"period_end" binding:"required"`
}

// SimplePeriodResponse is the response body for services without course-specific overrides.
type SimplePeriodResponse struct {
	ID          uuid.UUID `json:"id"`
	Semester    string    `json:"semester"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
