package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if err := logger.Init("test"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type probe struct {
	Name string `json:"name"`
}

func newTestBase(url string) *Base {
	return NewBase(Config{
		Target:  "staff-service",
		BaseURL: url,
		Secret:  "test-secret",
		Timeout: 2 * time.Second,
		Breaker: BreakerConfig{MaxRequests: 1, Timeout: 50 * time.Millisecond, ConsecutiveFailures: 3},
	})
}

func TestBase_Get_SuccessDecodesBodyAndSendsHeaders(t *testing.T) {
	var gotSecret, gotRequestID, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Internal-Secret")
		gotRequestID = r.Header.Get("X-Request-ID")
		gotPath = r.URL.RequestURI()
		_ = json.NewEncoder(w).Encode(probe{Name: "Ada"})
	}))
	defer srv.Close()

	ctx := logger.WithRequestIDValue(context.Background(), "req-42")
	var out probe
	require.NoError(t, newTestBase(srv.URL).Get(ctx, "/internal/staff/1?x=y", &out))

	assert.Equal(t, "Ada", out.Name)
	assert.Equal(t, "test-secret", gotSecret)
	assert.Equal(t, "req-42", gotRequestID)
	assert.Equal(t, "/internal/staff/1?x=y", gotPath)
}

func TestBase_Post_SendsJSONBody(t *testing.T) {
	var got probe
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(probe{Name: "echo"})
	}))
	defer srv.Close()

	var out probe
	require.NoError(t, newTestBase(srv.URL).Post(context.Background(), "/internal/payments/initiate", probe{Name: "Ada"}, &out))

	assert.Equal(t, "Ada", got.Name)
	assert.Equal(t, "echo", out.Name)
}

func TestBase_Get_NotFoundReturnsSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := newTestBase(srv.URL).Get(context.Background(), "/internal/staff/1", &probe{})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBase_Get_ServerErrorReturnsStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	err := newTestBase(srv.URL).Get(context.Background(), "/internal/staff/1", &probe{})

	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusInternalServerError, statusErr.StatusCode)
	assert.Contains(t, statusErr.Body, "boom")
}

func TestBase_Get_BadRequestReturnsStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	err := newTestBase(srv.URL).Get(context.Background(), "/internal/staff/1", &probe{})

	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
}

func TestBase_Get_ConsecutiveServerErrorsOpenBreaker(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	base := newTestBase(srv.URL)
	for range 3 {
		require.Error(t, base.Get(context.Background(), "/internal/staff/1", &probe{}))
	}

	err := base.Get(context.Background(), "/internal/staff/1", &probe{})
	appErr, ok := sharedErrors.As(err)
	require.True(t, ok, "open breaker must surface as AppError, got %v", err)
	assert.Equal(t, http.StatusServiceUnavailable, appErr.HTTPStatus)
	assert.Equal(t, 3, calls, "open breaker must not reach the target")
}

// A missing student is a business answer, not an outage. Counting 404s as
// failures would open the breaker during normal use and declare a healthy
// service dead.
func TestBase_Get_NotFoundDoesNotOpenBreaker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	base := newTestBase(srv.URL)
	for range 10 {
		assert.ErrorIs(t, base.Get(context.Background(), "/internal/staff/1", &probe{}), ErrNotFound)
	}
}

func TestBase_Get_HalfOpenClosesAfterRecovery(t *testing.T) {
	var healthy bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !healthy {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(probe{Name: "back"})
	}))
	defer srv.Close()

	base := newTestBase(srv.URL)
	for range 3 {
		require.Error(t, base.Get(context.Background(), "/internal/staff/1", &probe{}))
	}

	healthy = true
	time.Sleep(80 * time.Millisecond) // breaker timeout is 50ms in the test config

	var out probe
	require.NoError(t, base.Get(context.Background(), "/internal/staff/1", &out))
	assert.Equal(t, "back", out.Name)
}

func TestBase_Get_UnconfiguredBaseURLFails(t *testing.T) {
	base := NewBase(Config{Target: "staff-service", Secret: "s"})
	assert.Error(t, base.Get(context.Background(), "/internal/staff/1", &probe{}))
}
