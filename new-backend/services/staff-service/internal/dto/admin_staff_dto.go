package dto

import "time"

// AdminStaffRequest is a whole administrative staff record. Create and update
// take the same shape: the profile editor always sends every field, so PUT
// replaces rather than patches.
type AdminStaffRequest struct {
	Email     string `json:"email" binding:"required,email,max=255"`
	Title     string `json:"title" binding:"max=50"`
	FirstName string `json:"first_name" binding:"required,max=100"`
	LastName  string `json:"last_name" binding:"required,max=100"`
	Faculty   string `json:"faculty" binding:"required,max=200"`
	// The editor renders this as <img src>; http_url keeps javascript: and
	// data: schemes out.
	ProfileImageURL  string   `json:"profile_image_url" binding:"omitempty,http_url,max=2048"`
	Department       string   `json:"department" binding:"max=100"`
	Phone            string   `json:"phone" binding:"max=20"`
	Position         string   `json:"position" binding:"required,max=150"`
	JobDescription   string   `json:"job_description" binding:"max=2000"`
	Responsibilities []string `json:"responsibilities" binding:"max=30,dive,max=300"`
	WorkingHours     string   `json:"working_hours" binding:"max=100"`
	OfficeLocation   string   `json:"office_location" binding:"max=200"`
	StartDate        string   `json:"start_date" binding:"omitempty,datetime=2006-01-02"`
}

// AdminStaffResponse is one administrative staff record.
type AdminStaffResponse struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Title            string    `json:"title"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Faculty          string    `json:"faculty"`
	Department       string    `json:"department"`
	Phone            string    `json:"phone"`
	ProfileImageURL  string    `json:"profile_image_url"`
	Position         string    `json:"position"`
	JobDescription   string    `json:"job_description"`
	Responsibilities []string  `json:"responsibilities"`
	WorkingHours     string    `json:"working_hours"`
	OfficeLocation   string    `json:"office_location"`
	StartDate        string    `json:"start_date,omitempty"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AdminStaffListResponse wraps a faculty's administrative staff. Unpaged: a
// faculty has a handful of them.
type AdminStaffListResponse struct {
	Data []AdminStaffResponse `json:"data"`
}
