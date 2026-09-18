package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is an in-memory rate limit store with deterministic behavior.
type fakeStore struct {
	mu     sync.Mutex
	counts map[string]int
	allow  bool
	err    error
	calls  int
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

// identityRouter mounts the global limiter with a one-request IP budget, so
// any second request charged to the same bucket is rejected.
func identityRouter(t *testing.T, store *fakeStore) *gin.Engine {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret(authTestSecret)
	SetRateLimiter(NewRateLimiter(store, RateLimitConfig{
		ServiceName: "public",
		IPLimit:     1, IPWindow: time.Minute,
		UserLimit: 5, UserWindow: time.Minute,
		EndpointLimits: map[string]EndpointLimit{"refresh": {Limit: 1, Window: time.Minute}},
	}))
	t.Cleanup(func() { SetRateLimiter(nil) })

	r := gin.New()
	r.Use(IPRateLimit())
	r.GET("/api/x", UserRateLimit(), func(c *gin.Context) { c.Status(200) })
	r.GET("/internal/students/1", func(c *gin.Context) { c.Status(200) })
	r.POST("/api/auth/refresh", EndpointRateLimit("refresh"), func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(200, string(body))
	})
	return r
}

func sameIPRequest(method, path, bearer string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "203.0.113.10:5000" // one campus NAT address for everyone
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return req
}

func TestIPRateLimit_UsersBehindOneIP_GetSeparateBuckets(t *testing.T) {
	r := identityRouter(t, newFakeStore(true))

	for _, user := range []string{"student-a", "student-b", "student-c"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, sameIPRequest("GET", "/api/x", issueToken(t, user, "student", 1), ""))
		assert.Equal(t, 200, w.Code, "%s must not be throttled by the others' traffic", user)
	}
}

func TestIPRateLimit_AnonymousRequests_ShareTheIPBucket(t *testing.T) {
	r := identityRouter(t, newFakeStore(true))

	first, second := httptest.NewRecorder(), httptest.NewRecorder()
	r.ServeHTTP(first, sameIPRequest("GET", "/api/x", "", ""))
	r.ServeHTTP(second, sameIPRequest("GET", "/api/x", "", ""))

	assert.Equal(t, 200, first.Code)
	assert.Equal(t, http.StatusTooManyRequests, second.Code)
}

func TestIPRateLimit_ForgedToken_FallsBackToIP(t *testing.T) {
	r := identityRouter(t, newFakeStore(true))
	forged := issueToken(t, "victim", "student", 1) + "tampered"

	first, second := httptest.NewRecorder(), httptest.NewRecorder()
	r.ServeHTTP(first, sameIPRequest("GET", "/api/x", forged, ""))
	r.ServeHTTP(second, sameIPRequest("GET", "/api/x", forged, ""))

	assert.Equal(t, http.StatusTooManyRequests, second.Code,
		"an unverifiable token must not buy a fresh bucket")
}

func TestIPRateLimit_InternalRoutes_NotCounted(t *testing.T) {
	store := newFakeStore(true)
	r := identityRouter(t, store)

	for range 3 {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, sameIPRequest("GET", "/internal/students/1", "", ""))
		assert.Equal(t, 200, w.Code)
	}
	assert.Equal(t, 0, store.calls)
}

func TestIPRateLimit_AuthenticatedRequest_ChargedOnce(t *testing.T) {
	store := newFakeStore(true)
	r := identityRouter(t, store)

	w := httptest.NewRecorder()
	req := sameIPRequest("GET", "/api/x", issueToken(t, "student-a", "student", 1), "")
	// JWTAuth would have set this; UserRateLimit keys on it.
	r.Use(func(c *gin.Context) { c.Set("user_id", "student-a") })
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 1, store.calls, "global and user limiters must not both charge the user")
}

func TestEndpointRateLimit_Refresh_KeyedByTokenOwnerAndBodyKept(t *testing.T) {
	r := identityRouter(t, newFakeStore(true))

	for _, user := range []string{"student-a", "student-b"} {
		refresh, _, err := utils.GenerateRefreshTokenWithSecret(user, 1, []byte(authTestSecret), 24)
		require.NoError(t, err)
		body := `{"refresh_token":"` + refresh + `"}`

		w := httptest.NewRecorder()
		r.ServeHTTP(w, sameIPRequest("POST", "/api/auth/refresh", "", body))

		assert.Equal(t, 200, w.Code, "%s shares the NAT address but not the refresh bucket", user)
		assert.Equal(t, body, w.Body.String(), "the handler must still see the body")
	}
}
