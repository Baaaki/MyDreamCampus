package dto

import (
	"time"

	"github.com/google/uuid"
)

// BulkImportResponse represents the response for bulk import initiation
type BulkImportResponse struct {
	JobID                string    `json:"job_id"`
	Status               string    `json:"status"`
	Message              string    `json:"message"`
	TotalRecords         int       `json:"total_records"`
	EstimatedCompletion  time.Time `json:"estimated_completion"`
}

// ImportJobResponse represents the import job details
type ImportJobResponse struct {
	JobID              string           `json:"job_id"`
	FileName           string           `json:"file_name"`
	Status             string           `json:"status"`
	TotalRecords       int              `json:"total_records"`
	ProcessedRecords   int              `json:"processed_records"`
	SuccessfulRecords  int              `json:"successful_records"`
	FailedRecords      int              `json:"failed_records"`
	ProgressPercentage int              `json:"progress_percentage,omitempty"`
	Errors             []ImportJobError `json:"errors,omitempty"`
	CreatedBy          string           `json:"created_by"`
	StartedAt          *time.Time       `json:"started_at,omitempty"`
	CompletedAt        *time.Time       `json:"completed_at,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
}

// ImportJobError represents an error during import
type ImportJobError struct {
	Row           int    `json:"row"`
	StudentNumber string `json:"student_number"`
	ErrorCode     string `json:"error_code"`
	Message       string `json:"message"`
}

// ImportJobListResponse represents paginated import job list
type ImportJobListResponse struct {
	Data       []ImportJobSummary `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// ImportJobSummary represents summary info for import job list
type ImportJobSummary struct {
	JobID             string     `json:"job_id"`
	FileName          string     `json:"file_name"`
	Status            string     `json:"status"`
	TotalRecords      int        `json:"total_records"`
	SuccessfulRecords int        `json:"successful_records"`
	FailedRecords     int        `json:"failed_records"`
	CreatedAt         time.Time  `json:"created_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// CSVStudentRecord represents a row in the CSV file
type CSVStudentRecord struct {
	StudentNumber  string `csv:"student_number"`
	FirstName      string `csv:"first_name"`
	LastName       string `csv:"last_name"`
	Email          string `csv:"email"`
	Faculty        string `csv:"faculty"`
	Department     string `csv:"department"`
	EnrollmentYear int    `csv:"enrollment_year"`
	ClassLevel     int16  `csv:"class_level"`
}

// ImportJobFilterQuery represents query parameters for filtering import jobs
type ImportJobFilterQuery struct {
	PaginationQuery
	Status string `form:"status" binding:"omitempty,oneof=pending processing completed failed"`
}

// StaffAdvisor represents advisor from staff service (for caching)
type StaffAdvisor struct {
	ID         uuid.UUID `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Email      string    `json:"email"`
	Department string    `json:"department"`
}
