package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application-level error with HTTP status code
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New creates a new AppError
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Common HTTP errors
var (
	ErrNotFound           = New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrUnauthorized       = New("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden          = New("FORBIDDEN", "Forbidden", http.StatusForbidden)
	ErrValidation         = New("VALIDATION_ERROR", "Validation failed", http.StatusBadRequest)
	ErrInternalServer     = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrConflict           = New("CONFLICT", "Resource conflict", http.StatusConflict)
	ErrBadRequest         = New("BAD_REQUEST", "Bad request", http.StatusBadRequest)
	ErrServiceUnavailable = New("SERVICE_UNAVAILABLE", "Service unavailable", http.StatusServiceUnavailable)
)

// Staff Service specific errors
var (
	ErrEmailExists       = New("EMAIL_EXISTS", "Email already exists", http.StatusConflict)
	ErrStaffNotFound     = New("STAFF_NOT_FOUND", "Staff not found", http.StatusNotFound)
	ErrCannotCreateAdmin = New("CANNOT_CREATE_ADMIN", "Admin cannot be created via API", http.StatusBadRequest)
	ErrInvalidRole       = New("INVALID_ROLE", "Invalid role specified", http.StatusBadRequest)
	ErrInvalidID         = New("INVALID_ID", "Invalid ID format", http.StatusBadRequest)
	ErrStaffExists       = New("STAFF_EXISTS", "Staff already exists", http.StatusConflict)
	ErrInternal          = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
)

// Student Service specific errors
var (
	ErrStudentNotFound      = New("STUDENT_NOT_FOUND", "Student not found", http.StatusNotFound)
	ErrStudentNumberExists  = New("STUDENT_NUMBER_EXISTS", "Student number already exists", http.StatusConflict)
	ErrAdvisorNotFound      = New("ADVISOR_NOT_FOUND", "Advisor not found", http.StatusNotFound)
	ErrInvalidCSVFormat     = New("INVALID_CSV_FORMAT", "Invalid CSV format", http.StatusBadRequest)
	ErrStaffServiceUnavail  = New("STAFF_SERVICE_UNAVAILABLE", "Staff service is unavailable", http.StatusServiceUnavailable)
)

// Auth Service specific errors
var (
	ErrInvalidCredentials       = New("INVALID_CREDENTIALS", "Invalid email or password", http.StatusUnauthorized)
	ErrTokenRevoked             = New("TOKEN_REVOKED", "Token has been revoked", http.StatusUnauthorized)
	ErrWeakPassword             = New("WEAK_PASSWORD", "Password does not meet security requirements", http.StatusBadRequest)
	ErrAccountLocked            = New("ACCOUNT_LOCKED", "Account is temporarily locked due to multiple failed login attempts", http.StatusTooManyRequests)
	ErrAccountDeactivated       = New("ACCOUNT_DEACTIVATED", "Account has been deactivated", http.StatusUnauthorized)
	ErrRateLimitExceeded        = New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests)
	ErrInvalidToken             = New("INVALID_TOKEN", "Invalid token", http.StatusUnauthorized)
	ErrExpiredToken             = New("EXPIRED_TOKEN", "Token has expired", http.StatusUnauthorized)
	ErrCannotTerminateSession   = New("CANNOT_TERMINATE_CURRENT_SESSION", "Cannot terminate current session, use logout instead", http.StatusBadRequest)
	ErrForcePasswordChange      = New("FORCE_PASSWORD_CHANGE", "Password change required", http.StatusForbidden)
)

// IsAppError checks if an error is an AppError
func IsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}
