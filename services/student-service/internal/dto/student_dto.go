package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateStudentRequest represents the request body for creating a student
type CreateStudentRequest struct {
	StudentNumber  string    `json:"student_number" binding:"required"`
	FirstName      string    `json:"first_name" binding:"required"`
	LastName       string    `json:"last_name" binding:"required"`
	Email          string    `json:"email" binding:"required,email"`
	Faculty        string    `json:"faculty" binding:"required"`
	Department     string    `json:"department" binding:"required"`
	EnrollmentYear int       `json:"enrollment_year" binding:"required,min=1900,max=2100"`
	ClassLevel     int16     `json:"class_level" binding:"required,min=1,max=6"`
	AdvisorID      uuid.UUID `json:"advisor_id" binding:"required"`
}

// UpdateStudentRequest represents the request body for updating a student
type UpdateStudentRequest struct {
	ClassLevel *int16     `json:"class_level" binding:"omitempty,min=1,max=6"`
	AdvisorID  *uuid.UUID `json:"advisor_id"`
	Status     *string    `json:"status" binding:"omitempty,oneof=active graduated suspended withdrawn"`
}

// StudentResponse represents student response
type StudentResponse struct {
	ID             string         `json:"id"`
	StudentNumber  string         `json:"student_number"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	Email          string         `json:"email"`
	Faculty        string         `json:"faculty"`
	Department     string         `json:"department"`
	EnrollmentYear int            `json:"enrollment_year"`
	ClassLevel     int16          `json:"class_level"`
	AdvisorID      *string        `json:"advisor_id,omitempty"`
	Advisor        *AdvisorInfo   `json:"advisor,omitempty"`
	Status         string         `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// AdvisorInfo represents advisor basic information
type AdvisorInfo struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email,omitempty"`
}

// StudentListResponse represents paginated student list
type StudentListResponse struct {
	Data       []StudentResponse  `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// SearchStudentsRequest represents the request body for advanced search
type SearchStudentsRequest struct {
	Query      string            `json:"query"`
	Filters    StudentFilters    `json:"filters"`
	Sort       SortOptions       `json:"sort"`
	Pagination PaginationOptions `json:"pagination"`
}

// StudentFilters represents search filters
type StudentFilters struct {
	Department     []string    `json:"department"`
	ClassLevel     []int16     `json:"class_level"`
	Status         []string    `json:"status"`
	EnrollmentYear []int       `json:"enrollment_year"`
	AdvisorID      *uuid.UUID  `json:"advisor_id"`
}

// SortOptions represents sorting options
type SortOptions struct {
	Field string `json:"field" binding:"omitempty,oneof=student_number last_name enrollment_year class_level"`
	Order string `json:"order" binding:"omitempty,oneof=asc desc"`
}

// PaginationOptions represents cursor-based pagination options
type PaginationOptions struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchStudentsResponse represents search results
type SearchStudentsResponse struct {
	Data       []StudentResponse     `json:"data"`
	Pagination SearchPaginationResponse `json:"pagination"`
}

// SearchPaginationResponse represents cursor-based pagination metadata
type SearchPaginationResponse struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
	TotalCount int    `json:"total_count"`
}

// BulkAdvisorAssignRequest represents bulk advisor assignment request
type BulkAdvisorAssignRequest struct {
	StudentIDs []uuid.UUID `json:"student_ids" binding:"required,min=1"`
	AdvisorID  uuid.UUID   `json:"advisor_id" binding:"required"`
}

// BulkAdvisorAssignResponse represents bulk advisor assignment response
type BulkAdvisorAssignResponse struct {
	Message      string             `json:"message"`
	UpdatedCount int                `json:"updated_count"`
	Advisor      AdvisorInfo        `json:"advisor"`
	Students     []StudentBasicInfo `json:"students"`
}

// StudentBasicInfo represents minimal student info
type StudentBasicInfo struct {
	ID            string `json:"id"`
	StudentNumber string `json:"student_number"`
}

// MyAdviseesResponse represents teacher's advisees
type MyAdviseesResponse struct {
	Advisor    AdvisorInfo       `json:"advisor"`
	Students   []StudentResponse `json:"students"`
	TotalCount int               `json:"total_count"`
}

// OrphanedStudentsResponse represents orphaned students list
type OrphanedStudentsResponse struct {
	Data       []StudentResponse  `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}
