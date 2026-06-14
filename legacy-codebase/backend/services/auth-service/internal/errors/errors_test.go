package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/stretchr/testify/assert"
)

func TestAuthErrors_HTTPStatusCodes(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"INVALID_CREDENTIALS":              {ErrInvalidCredentials, http.StatusUnauthorized},
		"WEAK_PASSWORD":                    {ErrWeakPassword, http.StatusBadRequest},
		"INVALID_TOKEN":                    {ErrInvalidToken, http.StatusUnauthorized},
		"EXPIRED_TOKEN":                    {ErrExpiredToken, http.StatusUnauthorized},
		"TOKEN_REVOKED":                    {ErrTokenRevoked, http.StatusUnauthorized},
		"TOKEN_VERSION_MISMATCH":           {ErrTokenVersionMismatch, http.StatusUnauthorized},
		"ACCOUNT_LOCKED":                   {ErrAccountLocked, http.StatusTooManyRequests},
		"ACCOUNT_DEACTIVATED":              {ErrAccountDeactivated, http.StatusUnauthorized},
		"FORCE_PASSWORD_CHANGE":            {ErrForcePasswordChange, http.StatusForbidden},
		"CANNOT_TERMINATE_CURRENT_SESSION": {ErrCannotTerminateSession, http.StatusBadRequest},
		"RATE_LIMIT_EXCEEDED":              {ErrRateLimitExceeded, http.StatusTooManyRequests},
		"USER_NOT_FOUND":                   {ErrUserNotFound, http.StatusNotFound},
		"USER_EXISTS":                      {ErrUserExists, http.StatusConflict},
		"EMAIL_EXISTS":                     {ErrEmailExists, http.StatusConflict},
		"SESSION_NOT_FOUND":                {ErrSessionNotFound, http.StatusNotFound},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code, "code mismatch")
		assert.Equal(t, c.want, c.err.HTTPStatus, "http status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestRepositorySentinels_AliasShared(t *testing.T) {
	assert.Same(t, sharedErrors.ErrNotFoundRepo, ErrUserNotFoundRepo)
	assert.Same(t, sharedErrors.ErrAlreadyExistsRepo, ErrUserExistsRepo)
	assert.Same(t, sharedErrors.ErrNotFoundRepo, ErrSessionNotFoundRepo)
}

func TestAuthErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrInvalidCredentials, ErrWeakPassword,
		ErrInvalidToken, ErrExpiredToken, ErrTokenRevoked, ErrTokenVersionMismatch,
		ErrAccountLocked, ErrAccountDeactivated, ErrForcePasswordChange,
		ErrCannotTerminateSession, ErrRateLimitExceeded,
		ErrUserNotFound, ErrUserExists, ErrEmailExists, ErrSessionNotFound,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate error code %q", e.Code)
		seen[e.Code] = true
	}
}
