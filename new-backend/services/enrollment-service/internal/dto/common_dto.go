package dto

// ScheduleSession represents a course schedule session
type ScheduleSession struct {
	DayOfWeek   string `json:"day_of_week"`
	SlotNumbers []int  `json:"slot_numbers"`
	SessionType string `json:"session_type"`
}

// CourseBasic represents basic course information
type CourseBasic struct {
	ID               string            `json:"id"`
	CourseCode       string            `json:"course_code"`
	CourseName       string            `json:"course_name"`
	Credits          int16             `json:"credits"`
	InstructorName   string            `json:"instructor,omitempty"`
	ScheduleSessions []ScheduleSession `json:"schedule_sessions"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page  int `json:"page" form:"page"`
	Limit int `json:"limit" form:"limit"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
