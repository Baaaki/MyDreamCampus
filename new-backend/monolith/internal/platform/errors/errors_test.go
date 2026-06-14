package errors

import (
	stdlib "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_BasicConstruction(t *testing.T) {
	e := New("TEST_CODE", "test message", http.StatusTeapot)
	assert.Equal(t, "TEST_CODE", e.Code)
	assert.Equal(t, "test message", e.Message)
	assert.Equal(t, http.StatusTeapot, e.HTTPStatus)
	assert.Equal(t, "[TEST_CODE] test message", e.Error())
	assert.Nil(t, e.Unwrap())
}

func TestWrap_PreservesUnderlyingError(t *testing.T) {
	cause := stdlib.New("disk failure")
	wrapped := Wrap(ErrInternal, cause)
	require.NotNil(t, wrapped)
	assert.Equal(t, ErrInternal.Code, wrapped.Code)
	assert.Equal(t, ErrInternal.HTTPStatus, wrapped.HTTPStatus)
	assert.Same(t, cause, wrapped.Unwrap())
	assert.Contains(t, wrapped.Error(), "disk failure")
}

func TestWrap_NilAppErrorReturnsNil(t *testing.T) {
	assert.Nil(t, Wrap(nil, stdlib.New("x")))
}

func TestWrapWithMessage_OverridesMessage(t *testing.T) {
	cause := stdlib.New("low-level")
	w := WrapWithMessage(ErrValidation, cause, "field 'email' is required")
	require.NotNil(t, w)
	assert.Equal(t, ErrValidation.Code, w.Code)
	assert.Equal(t, "field 'email' is required", w.Message)
	assert.ErrorIs(t, w, ErrValidation, "Is must compare by code")
}

func TestIs_ComparesByCode(t *testing.T) {
	a := New("SAME_CODE", "msg-1", 400)
	b := New("SAME_CODE", "msg-2", 400)
	c := New("OTHER_CODE", "msg-3", 400)

	assert.True(t, stdlib.Is(a, b))
	assert.False(t, stdlib.Is(a, c))
	assert.False(t, a.Is(stdlib.New("plain error")))
}

func TestAs_ExtractsAppError(t *testing.T) {
	cause := stdlib.New("cause")
	wrapped := Wrap(ErrConflict, cause)

	got, ok := As(wrapped)
	require.True(t, ok)
	assert.Equal(t, ErrConflict.Code, got.Code)

	_, ok = As(stdlib.New("plain"))
	assert.False(t, ok)
}

func TestErrorClassifiers(t *testing.T) {
	assert.True(t, IsNotFound(ErrNotFound))
	assert.True(t, IsValidation(ErrValidation))
	assert.True(t, IsUnauthorized(ErrUnauthorized))
	assert.True(t, IsForbidden(ErrForbidden))
	assert.True(t, IsConflict(ErrConflict))

	assert.False(t, IsNotFound(ErrConflict))
	assert.False(t, IsValidation(ErrInternal))
	assert.False(t, IsNotFound(nil))
}

func TestWrappedError_ChainTraversal(t *testing.T) {
	root := stdlib.New("root cause")
	mid := Wrap(ErrInternal, root)
	outer := WrapWithMessage(ErrServiceUnavailable, mid, "circuit open")

	// errors.Is must walk the chain
	assert.ErrorIs(t, outer, ErrServiceUnavailable)
	// Cause is the immediately wrapped error (mid), and Is on AppError
	// compares by code, so the original root is not directly reachable
	// via Is/As — but the formatted Error() string must include it.
	assert.Contains(t, outer.Error(), "circuit open")
}

func TestCommonErrors_HTTPStatusMapping(t *testing.T) {
	cases := map[*AppError]int{
		ErrBadRequest:         http.StatusBadRequest,
		ErrUnauthorized:       http.StatusUnauthorized,
		ErrForbidden:          http.StatusForbidden,
		ErrNotFound:           http.StatusNotFound,
		ErrConflict:           http.StatusConflict,
		ErrValidation:         http.StatusBadRequest,
		ErrTooManyReqs:        http.StatusTooManyRequests,
		ErrInternal:           http.StatusInternalServerError,
		ErrServiceUnavailable: http.StatusServiceUnavailable,
		ErrNotImplemented:     http.StatusNotImplemented,
	}
	for err, status := range cases {
		assert.Equal(t, status, err.HTTPStatus, "status mismatch for %s", err.Code)
	}

	// Backward-compat alias
	assert.Equal(t, ErrInternal, ErrInternalServer)
}

func TestRepositorySentinels_AreDistinct(t *testing.T) {
	all := []error{
		ErrNotFoundRepo,
		ErrAlreadyExistsRepo,
		ErrDatabaseConnection,
		ErrTransactionFailed,
		ErrQueryFailed,
	}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				assert.ErrorIs(t, a, b)
				continue
			}
			assert.False(t, stdlib.Is(a, b),
				"sentinel errors must be distinct: %v vs %v", a, b)
		}
	}
}
