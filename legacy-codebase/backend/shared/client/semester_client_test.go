package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = logger.Init("test")
	m.Run()
}

func TestGetSemesterInfo_FetchesAndCaches(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"name":"2026-spring","status":"active","hard_deadline":"2026-07-01T00:00:00Z","is_past_deadline":false}`))
	}))
	defer srv.Close()

	c := NewSemesterClient(srv.URL)

	info, err := c.GetSemesterInfo(context.Background(), "2026-spring")
	require.NoError(t, err)
	assert.Equal(t, "2026-spring", info.Name)
	assert.Equal(t, "active", info.Status)
	assert.False(t, info.IsPastDeadline)

	// Second call should hit cache
	info2, err := c.GetSemesterInfo(context.Background(), "2026-spring")
	require.NoError(t, err)
	assert.Equal(t, info, info2)
	assert.EqualValues(t, 1, calls.Load(), "second call must hit cache")
}

func TestGetSemesterInfo_NotFoundReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewSemesterClient(srv.URL)
	_, err := c.GetSemesterInfo(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetSemesterInfo_5xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := NewSemesterClient(srv.URL)
	_, err := c.GetSemesterInfo(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}

func TestGetSemesterInfo_ForwardsRequestID(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-ID")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"name":"x","status":"active","hard_deadline":"2026-07-01T00:00:00Z","is_past_deadline":false}`))
	}))
	defer srv.Close()

	ctx := logger.WithRequestIDValue(context.Background(), "trace-abc-123")
	c := NewSemesterClient(srv.URL)
	_, err := c.GetSemesterInfo(ctx, "x")
	require.NoError(t, err)
	assert.Equal(t, "trace-abc-123", got, "X-Request-ID must propagate to downstream calls")
}

func TestInvalidateCache_RefetchesOnNextCall(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"name":"y","status":"active","hard_deadline":"2026-07-01T00:00:00Z","is_past_deadline":false}`))
	}))
	defer srv.Close()

	c := NewSemesterClient(srv.URL)
	_, _ = c.GetSemesterInfo(context.Background(), "y")
	c.InvalidateCache("y")
	_, _ = c.GetSemesterInfo(context.Background(), "y")

	assert.EqualValues(t, 2, calls.Load(), "InvalidateCache must force a refetch")
}

func TestGetSemesterInfo_CacheExpiresAfterTTL(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"name":"z","status":"active","hard_deadline":"2026-07-01T00:00:00Z","is_past_deadline":false}`))
	}))
	defer srv.Close()

	c := NewSemesterClient(srv.URL)
	c.cacheTTL = 5 * time.Millisecond

	_, _ = c.GetSemesterInfo(context.Background(), "z")
	time.Sleep(15 * time.Millisecond)
	_, _ = c.GetSemesterInfo(context.Background(), "z")

	assert.EqualValues(t, 2, calls.Load(), "expired cache entry must trigger refetch")
}
