package dto

import "github.com/google/uuid"

// PaginationQuery represents pagination query parameters with filters and sorting
type PaginationQuery struct {
	Page       int        `form:"page,default=1" binding:"min=1"`
	Limit      int        `form:"limit,default=20" binding:"min=1,max=100"`
	Department *string    `form:"department"`
	ClassLevel *int16     `form:"class_level" binding:"omitempty,min=1,max=6"`
	Status     *string    `form:"status" binding:"omitempty,oneof=active graduated suspended withdrawn"`
	AdvisorID  *uuid.UUID `form:"advisor_id"`
	SortBy     *string    `form:"sort_by" binding:"omitempty,oneof=student_number last_name enrollment_year class_level created_at"`
	SortOrder  *string    `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// MessageResponse represents success message response
type MessageResponse struct {
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}
