package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSemesterTestClient(t *testing.T, handler http.HandlerFunc) *HTTPSemesterClient {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewHTTPSemesterClient(client.NewBase(client.Config{
		Target:  "catalog-service",
		BaseURL: srv.URL,
		Secret:  "test-secret",
	}))
}

func TestHTTPSemesterClient_GetSemesterInfo_ParsesHardDeadline(t *testing.T) {
	var gotPath, gotSecret string
	c := newSemesterTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotSecret = r.URL.Path, r.Header.Get("X-Internal-Secret")
		_, _ = w.Write([]byte(`{"name":"2025-2026-Fall","status":"active","hard_deadline":"2026-02-01T00:00:00Z","is_past_deadline":false}`))
	})

	info, err := c.GetSemesterInfo(context.Background(), "2025-2026-Fall")
	require.NoError(t, err)

	assert.Equal(t, "active", info.Status)
	assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), info.HardDeadline.UTC())
	assert.False(t, info.IsPastDeadline)
	assert.Equal(t, "/internal/semesters/2025-2026-Fall/info", gotPath)
	assert.Equal(t, "test-secret", gotSecret)
}

func TestHTTPSemesterClient_GetSemesterInfo_NotFoundIsNotAnOutage(t *testing.T) {
	c := newSemesterTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetSemesterInfo(context.Background(), "2030-2031-Fall")
	require.Error(t, err)
	assert.NotErrorIs(t, err, client.ErrUnavailable, "a missing semester must stay degradable")
}

// The hard deadline binds admins too, so the caller has to tell an outage
// apart from a missing semester and refuse instead of skipping the check.
func TestHTTPSemesterClient_GetSemesterInfo_ServerErrorMarksUnavailable(t *testing.T) {
	c := newSemesterTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetSemesterInfo(context.Background(), "2025-2026-Fall")
	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrUnavailable)
}
