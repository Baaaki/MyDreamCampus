package errors

import "net/http"

// ============================================================================
// COMMON HTTP ERRORS
// These errors are used across ALL services for standard HTTP responses
// Service-specific errors should be defined in service/internal/errors/
// ============================================================================

var (
	// 4xx Client Errors
	ErrBadRequest    = New("BAD_REQUEST", "Bad request", http.StatusBadRequest)
	ErrUnauthorized  = New("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden     = New("FORBIDDEN", "Forbidden", http.StatusForbidden)
	ErrNotFound      = New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrConflict      = New("CONFLICT", "Resource conflict", http.StatusConflict)
	ErrValidation    = New("VALIDATION_ERROR", "Validation failed", http.StatusBadRequest)
	ErrInvalidID     = New("INVALID_ID", "Invalid ID format", http.StatusBadRequest)
	ErrTooManyReqs   = New("TOO_MANY_REQUESTS", "Too many requests", http.StatusTooManyRequests)

	// 5xx Server Errors
	ErrInternal           = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrServiceUnavailable = New("SERVICE_UNAVAILABLE", "Service unavailable", http.StatusServiceUnavailable)
	ErrNotImplemented     = New("NOT_IMPLEMENTED", "Not implemented", http.StatusNotImplemented)

	// Deprecated aliases (kept for backward compatibility, will be removed in future)
	ErrInternalServer = ErrInternal // Use ErrInternal instead
)
