package errors

import (
	"errors"
	"fmt"
)

// ============================================================================
// CORE ERROR TYPES
// Modern Go error handling with wrapping support (Go 1.13+)
// Implements errors.Is(), errors.As(), and errors.Unwrap() patterns
// ============================================================================

// AppError represents an application-level error with HTTP status code
// Supports error wrapping for maintaining error context chain
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	cause      error  // Wrapped underlying error (private for encapsulation)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap implements errors.Unwrap() for error chain traversal
// This allows errors.Is() and errors.As() to work with wrapped errors
func (e *AppError) Unwrap() error {
	return e.cause
}

// Is implements custom error comparison for errors.Is()
// Compares error codes for semantic equality
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// ============================================================================
// CONSTRUCTOR FUNCTIONS
// ============================================================================

// New creates a new AppError without wrapping
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		cause:      nil,
	}
}

// Wrap creates a new AppError that wraps an underlying error
// This maintains the error context chain for debugging
//
// Example:
//   if err := db.Query(); err != nil {
//       return errors.Wrap(errors.ErrInternal, err)
//   }
func Wrap(appErr *AppError, cause error) *AppError {
	if appErr == nil {
		return nil
	}
	return &AppError{
		Code:       appErr.Code,
		Message:    appErr.Message,
		HTTPStatus: appErr.HTTPStatus,
		cause:      cause,
	}
}

// WrapWithMessage creates a new AppError with custom message that wraps an underlying error
func WrapWithMessage(appErr *AppError, cause error, message string) *AppError {
	if appErr == nil {
		return nil
	}
	return &AppError{
		Code:       appErr.Code,
		Message:    message,
		HTTPStatus: appErr.HTTPStatus,
		cause:      cause,
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// As is a convenience wrapper around errors.As for AppError type assertion
// Returns the AppError and true if the error chain contains an AppError
//
// Example:
//   if appErr, ok := errors.As(err); ok {
//       c.JSON(appErr.HTTPStatus, appErr)
//   }
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// Is checks if err is or wraps target using errors.Is
// This is a convenience wrapper for cleaner code
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// IsNotFound checks if the error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsValidation checks if the error is ErrValidation
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

// IsUnauthorized checks if the error is ErrUnauthorized
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks if the error is ErrForbidden
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsConflict checks if the error is ErrConflict
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}
