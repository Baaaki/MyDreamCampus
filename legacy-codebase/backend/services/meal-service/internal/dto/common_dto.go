package dto

// ErrorResponse represents error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// SuccessResponse represents generic success response
type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// ErrorResponseWrapper wraps error response
type ErrorResponseWrapper struct {
	Success bool          `json:"success"`
	Error   ErrorResponse `json:"error"`
}

// MessageResponse represents success message response
type MessageResponse struct {
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}
