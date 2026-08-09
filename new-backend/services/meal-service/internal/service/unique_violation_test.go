package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// isUniqueViolation classifies pgx errors so that CreateBatchReservation /
// CreateReservation can convert duplicate-key races into a domain conflict
// error instead of a 500. A regression here means concurrent reservations
// for the same (student, date, meal_time) leak as internal errors instead
// of a meaningful 409, and the user retries on a flaky-looking failure.

func TestIsUniqueViolation_PgError23505_True(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	assert.True(t, isUniqueViolation(err))
}

func TestIsUniqueViolation_OtherPgError_False(t *testing.T) {
	cases := []string{
		"23502", // not_null_violation
		"23503", // foreign_key_violation
		"23514", // check_violation
		"08006", // connection_failure
		"",
	}
	for _, code := range cases {
		t.Run(code, func(t *testing.T) {
			err := &pgconn.PgError{Code: code}
			assert.False(t, isUniqueViolation(err), "code %q must not be classified as unique violation", code)
		})
	}
}

func TestIsUniqueViolation_PlainError_False(t *testing.T) {
	assert.False(t, isUniqueViolation(errors.New("some random error")))
	assert.False(t, isUniqueViolation(fmt.Errorf("wrapped: %w", errors.New("inner"))))
}

func TestIsUniqueViolation_Nil_False(t *testing.T) {
	// Caller passes nil only by mistake, but the helper must not panic.
	assert.False(t, isUniqueViolation(nil))
}

func TestIsUniqueViolation_WrappedPgError_True(t *testing.T) {
	// errors.As must unwrap — the helper relies on that for callers that wrap
	// pgx errors with extra context (logger, repo layer, etc.).
	wrapped := fmt.Errorf("repository: insert failed: %w", &pgconn.PgError{Code: "23505"})
	assert.True(t, isUniqueViolation(wrapped), "errors.As must unwrap a layered pgx error")
}
