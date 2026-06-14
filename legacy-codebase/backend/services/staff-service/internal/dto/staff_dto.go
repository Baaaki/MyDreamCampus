package dto

import "time"

// CreateStaffRequest represents the request body for creating staff
type CreateStaffRequest struct {
	Email          string `json:"email" binding:"required,email"`
	FirstName      string `json:"first_name" binding:"required"`
	LastName       string `json:"last_name" binding:"required"`
	Role           string `json:"role" binding:"required,oneof=teacher"`
	Department     string `json:"department"`
	Phone          string `json:"phone"`
	OfficeLocation string `json:"office_location"`
}

// UpdateStaffRequest represents the request body for updating staff
type UpdateStaffRequest struct {
	Department     *string `json:"department"`
	Phone          *string `json:"phone"`
	OfficeLocation *string `json:"office_location"`
}

// StaffResponse represents staff response
type StaffResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Role           string    `json:"role"`
	Department     string    `json:"department,omitempty"`
	Phone          string    `json:"phone,omitempty"`
	OfficeLocation string    `json:"office_location,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// StaffListResponse represents paginated staff list
type StaffListResponse struct {
	Data       []StaffResponse    `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}
