package semester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPChecker_IsActive_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/2026-spring/status"))
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"active":true}`))
	}))
	defer srv.Close()

	c := NewHTTPChecker(srv.URL)
	active, err := c.IsSemesterActive(context.Background(), "2026-spring")
	require.NoError(t, err)
	assert.True(t, active)
}

func TestHTTPChecker_IsActive_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"active":false}`))
	}))
	defer srv.Close()

	c := NewHTTPChecker(srv.URL)
	active, err := c.IsSemesterActive(context.Background(), "2025-fall")
	require.NoError(t, err)
	assert.False(t, active)
}

func TestHTTPChecker_NonOKReturnsFalseNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := NewHTTPChecker(srv.URL)
	active, err := c.IsSemesterActive(context.Background(), "missing")
	require.NoError(t, err, "non-OK status must NOT be treated as error")
	assert.False(t, active)
}

func TestHTTPChecker_BadJSONReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewHTTPChecker(srv.URL)
	_, err := c.IsSemesterActive(context.Background(), "x")
	assert.Error(t, err)
}

func TestHTTPChecker_TransportErrorReturnsError(t *testing.T) {
	c := NewHTTPChecker("http://127.0.0.1:1") // unreachable
	_, err := c.IsSemesterActive(context.Background(), "x")
	assert.Error(t, err)
}
