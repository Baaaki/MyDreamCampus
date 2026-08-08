package errors

import "errors"

// ============================================================================
// GENERIC REPOSITORY SENTINEL ERRORS
// These are pure sentinel errors (not AppError) for Repository ↔ Service communication
// They represent data layer failures independent of business domain
//
// USAGE PATTERN:
//   Repository layer: Return these sentinel errors
//   Service layer: Check with errors.Is() and convert to AppError
//
// WHY SENTINEL ERRORS?
//   - Repository doesn't know about HTTP (no status codes)
//   - Service layer translates to domain-specific AppError
//   - Keeps separation of concerns clean
// ============================================================================

var (
	// Generic "not found" errors - used when a resource doesn't exist
	ErrNotFoundRepo = errors.New("resource not found in repository")

	// Generic "already exists" errors - used for duplicate/uniqueness violations
	ErrAlreadyExistsRepo = errors.New("resource already exists in repository")

	// Database infrastructure errors
	ErrDatabaseConnection = errors.New("database connection failed")
	ErrTransactionFailed  = errors.New("database transaction failed")
	ErrQueryFailed        = errors.New("database query failed")
)
