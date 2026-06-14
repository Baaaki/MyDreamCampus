package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is an in-memory rate limit store with deterministic behavior.
type fakeStore struct {
	mu       sync.Mutex
	counts   map[string]int
	allow    bool
	err      error
	calls    int
}

func (f *fakeStore) CheckRateLimit(_ context.Context, key string, limit int, _ time.Duration) (bool, int, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return false, 0, 0, f.err
	}
	f.counts[key]++
	used := f.counts[key]
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	if used > limit {
		return false, 0, 60, nil
	}
	return f.allow, remaining, 0, nil
}

func newFakeStore(allow bool) *fakeStore {
	return &fakeStore{counts: map[string]int{}, allow: allow}
}

func TestIPRateLimit_PassesWhenUnconfigured(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	SetRateLimiter(nil)

	r := gin.New()
	r.GET("/", IPRateLimit(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestIPRateLimit_AllowsBelowLimit(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	store := newFakeStore(true)
	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test", IPLimit: 100, IPWindow: time.Minute,
	}))

	r := gin.New()
	r.GET("/", IPRateLimit(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	rem, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
	assert.Less(t, rem, 100)
}

func TestIPRateLimit_RejectsAboveLimit(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	store := newFakeStore(false) // store says deny
	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test", IPLimit: 1, IPWindow: time.Minute,
	}))

	r := gin.New()
	r.GET("/", IPRateLimit(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestIPRateLimit_FailOpenOnStoreError(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	store := newFakeStore(true)
	store.err = errors.New("store down")
	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test", IPLimit: 100, IPWindow: time.Minute,
	}))

	r := gin.New()
	r.GET("/", IPRateLimit(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "IP rate limit must fail-open by default")
}

func TestUserRateLimit_NoUserPassesThrough(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	store := newFakeStore(true)
	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test", UserLimit: 100, UserWindow: time.Minute,
	}))

	r := gin.New()
	r.GET("/", UserRateLimit(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 0, store.calls, "no user_id => store must not be called")
}

func TestUserRateLimit_KeyedByUser(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	store := newFakeStore(true)
	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test", UserLimit: 10, UserWindow: time.Minute,
	}))

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "u-42"); c.Next() })
	r.GET("/", UserRateLimit(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 1, store.calls)
	for k := range store.counts {
		assert.Contains(t, k, "u-42")
	}
}

func TestEndpointRateLimit_NoConfigPassesThrough(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	SetRateLimiter(NewRateLimiter(newFakeStore(true), RateLimitConfig{
		ServiceName:    "test",
		EndpointLimits: map[string]EndpointLimit{}, // no group => bypass
	}))

	r := gin.New()
	r.POST("/login", EndpointRateLimit("login"), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("POST", "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestEndpointRateLimit_FailClosedOnError(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	store := newFakeStore(true)
	store.err = errors.New("redis down")

	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test",
		EndpointLimits: map[string]EndpointLimit{
			"login": {Limit: 5, Window: time.Minute, FailClosed: true},
		},
	}))

	r := gin.New()
	r.POST("/login", EndpointRateLimit("login"), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("POST", "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code,
		"FailClosed: true must return 503 on Redis error")
}

func TestEndpointRateLimit_FailOpenWhenNotConfigured(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	store := newFakeStore(true)
	store.err = errors.New("redis down")

	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "test",
		EndpointLimits: map[string]EndpointLimit{
			"export": {Limit: 5, Window: time.Minute}, // FailClosed: false
		},
	}))

	r := gin.New()
	r.GET("/x", EndpointRateLimit("export"), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
