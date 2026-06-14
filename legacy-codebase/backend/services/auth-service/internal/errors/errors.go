package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
)

// ============================================================================
// AUTH SERVICE SPECIFIC ERRORS
// These errors are specific to authentication and authorization domain
// They should NOT be moved to shared/errors as they represent business logic
// unique to the auth service
// ============================================================================

var (
	// Authentication errors
	ErrInvalidCredentials = sharedErrors.New("INVALID_CREDENTIALS", "Invalid email or password", http.StatusUnauthorized)
	ErrWeakPassword       = sharedErrors.New("WEAK_PASSWORD", "Password does not meet security requirements", http.StatusBadRequest)

	// Token errors
	ErrInvalidToken         = sharedErrors.New("INVALID_TOKEN", "Invalid token", http.StatusUnauthorized)
	ErrExpiredToken         = sharedErrors.New("EXPIRED_TOKEN", "Token has expired", http.StatusUnauthorized)
	ErrTokenRevoked         = sharedErrors.New("TOKEN_REVOKED", "Token has been revoked", http.StatusUnauthorized)
	ErrTokenVersionMismatch = sharedErrors.New("TOKEN_VERSION_MISMATCH", "Token version mismatch, please login again", http.StatusUnauthorized)

	// Account status errors
	ErrAccountLocked      = sharedErrors.New("ACCOUNT_LOCKED", "Account is temporarily locked due to multiple failed login attempts", http.StatusTooManyRequests)
	ErrAccountDeactivated = sharedErrors.New("ACCOUNT_DEACTIVATED", "Account has been deactivated", http.StatusUnauthorized)
	ErrForcePasswordChange = sharedErrors.New("FORCE_PASSWORD_CHANGE", "Password change required", http.StatusForbidden)

	// Session errors
	ErrCannotTerminateSession = sharedErrors.New("CANNOT_TERMINATE_CURRENT_SESSION", "Cannot terminate current session, use logout instead", http.StatusBadRequest)

	// Rate limiting
	ErrRateLimitExceeded = sharedErrors.New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests)

	// User errors
	ErrUserNotFound = sharedErrors.New("USER_NOT_FOUND", "User not found", http.StatusNotFound)
	ErrUserExists   = sharedErrors.New("USER_EXISTS", "User already exists", http.StatusConflict)
	ErrEmailExists  = sharedErrors.New("EMAIL_EXISTS", "Email already exists", http.StatusConflict)

	// Session errors
	ErrSessionNotFound = sharedErrors.New("SESSION_NOT_FOUND", "Session not found", http.StatusNotFound)

	// Repository-specific sentinel errors (for internal use)
	ErrUserNotFoundRepo    = sharedErrors.ErrNotFoundRepo
	ErrUserExistsRepo      = sharedErrors.ErrAlreadyExistsRepo
	ErrSessionNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
