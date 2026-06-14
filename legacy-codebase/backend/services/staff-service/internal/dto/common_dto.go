package dto

// PaginationQuery represents pagination query parameters
type PaginationQuery struct {
	Page  int `form:"page,default=1" binding:"min=1"`
	Limit int `form:"limit,default=20" binding:"min=1,max=100"`
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
