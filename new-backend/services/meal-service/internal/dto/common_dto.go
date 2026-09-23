package dto

// ErrorResponse is the error body every service answers with: the Turkish
// message to show and the code clients branch on.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// SuccessResponse represents generic success response
type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// MessageResponse represents success message response
type MessageResponse struct {
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}
