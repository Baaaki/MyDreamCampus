package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const authTestSecret = "test-secret-key-minimum-32-characters-long-aaa"

// fakeBlacklist is an in-memory implementation of TokenBlacklistChecker.
type fakeBlacklist struct {
	blacklisted   map[string]bool
	minVersion    map[string]int
	errOnCheck    bool
	errOnVersion  bool
}

func (f *fakeBlacklist) IsAccessTokenBlacklisted(_ context.Context, jti string) (bool, error) {
	if f.errOnCheck {
		return false, errors.New("redis down")
	}
	return f.blacklisted[jti], nil
}

func (f *fakeBlacklist) GetMinTokenVersion(_ context.Context, userID string) (int, error) {
	if f.errOnVersion {
		return 0, errors.New("redis down")
	}
	return f.minVersion[userID], nil
}

func setupAuthTest(t *testing.T, blacklist *fakeBlacklist, opts ...AuthOption) *gin.Engine {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", authTestSecret)
	if blacklist == nil {
		// untyped nil — middleware short-circuits the blacklist check
		SetBlacklistChecker(nil)
	} else {
		SetBlacklistChecker(blacklist)
	}

	r := gin.New()
	r.GET("/protected", JWTAuth(opts...), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		c.JSON(200, gin.H{"user": uid})
	})
	return r
}

func issueToken(t *testing.T, userID, role string, version int) string {
	t.Helper()
	tok, _, err := utils.GenerateAccessTokenWithSecret(userID, role, "", version, []byte(authTestSecret), 15)
	require.NoError(t, err)
	return tok
}

func TestJWTAuth_RejectsMissingToken(t *testing.T) {
	r := setupAuthTest(t, nil)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_RejectsInvalidToken(t *testing.T) {
	r := setupAuthTest(t, nil)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.real.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_AcceptsValidBearerToken(t *testing.T) {
	r := setupAuthTest(t, nil)
	tok := issueToken(t, "user-7", "student", 1)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "user-7")
}

func TestJWTAuth_FallsBackToCookie(t *testing.T) {
	r := setupAuthTest(t, nil)
	tok := issueToken(t, "user-cookie", "admin", 2)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: tok})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "user-cookie")
}

func TestJWTAuth_RejectsBlacklistedJTI(t *testing.T) {
	tok := issueToken(t, "user-x", "student", 1)
	claims, err := utils.ValidateTokenWithSecret(tok, []byte(authTestSecret))
	require.NoError(t, err)

	bl := &fakeBlacklist{
		blacklisted: map[string]bool{claims.JTI: true},
		minVersion:  map[string]int{},
	}
	r := setupAuthTest(t, bl)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "revoked")
}

func TestJWTAuth_RejectsTokenVersionTooOld(t *testing.T) {
	tok := issueToken(t, "user-old", "student", 1)
	bl := &fakeBlacklist{
		blacklisted: map[string]bool{},
		minVersion:  map[string]int{"user-old": 5},
	}
	r := setupAuthTest(t, bl)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_FailOpenOnRedisError(t *testing.T) {
	tok := issueToken(t, "user-fo", "student", 1)
	bl := &fakeBlacklist{
		blacklisted: map[string]bool{},
		minVersion:  map[string]int{},
		errOnCheck:  true,
		errOnVersion: true,
	}
	r := setupAuthTest(t, bl)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "default fail-open: request must succeed when Redis is down")
}

func TestJWTAuth_FailClosedOnRedisError(t *testing.T) {
	tok := issueToken(t, "user-fc", "student", 1)
	bl := &fakeBlacklist{
		blacklisted:  map[string]bool{},
		minVersion:   map[string]int{},
		errOnCheck:   true,
		errOnVersion: true,
	}
	r := setupAuthTest(t, bl, WithFailClosed())

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestOptionalJWTAuth_PassesWithoutToken(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", authTestSecret)

	r := gin.New()
	r.GET("/", OptionalJWTAuth(), func(c *gin.Context) {
		_, ok := c.Get("user_id")
		c.JSON(200, gin.H{"authed": ok})
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"authed":false`)
}

func TestOptionalJWTAuth_SetsClaimsWhenPresent(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", authTestSecret)

	r := gin.New()
	r.GET("/", OptionalJWTAuth(), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		c.JSON(200, gin.H{"user": uid})
	})

	tok := issueToken(t, "opt-1", "teacher", 3)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Body.String(), "opt-1")
}

func TestExtractUserFromHeaders(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/", ExtractUserFromHeaders(), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		role, _ := c.Get("role")
		dept, _ := c.Get("department")
		c.JSON(200, gin.H{"user": uid, "role": role, "dept": dept})
	})

	t.Run("rejects missing user header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("rejects missing role header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("accepts when both present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User-ID", "u-1")
		req.Header.Set("X-User-Role", "admin")
		req.Header.Set("X-User-Department", "CS")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), "u-1")
		assert.Contains(t, w.Body.String(), "admin")
		assert.Contains(t, w.Body.String(), "CS")
	})
}

func TestStripInternalHeaders_PanicsWithoutSecret(t *testing.T) {
	os.Unsetenv("INTERNAL_SERVICE_SECRET")
	assert.Panics(t, func() { _ = StripInternalHeaders() })
}

func TestStripInternalHeaders_StripsUntrustedHeaders(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_SERVICE_SECRET", "internal-shared-secret")

	r := gin.New()
	r.Use(StripInternalHeaders())
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"uid":  c.GetHeader("X-User-ID"),
			"role": c.GetHeader("X-User-Role"),
		})
	})

	t.Run("strips when no internal secret", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User-ID", "spoofed-1")
		req.Header.Set("X-User-Role", "admin")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.NotContains(t, w.Body.String(), "spoofed-1",
			"X-User-ID without internal secret must be stripped")
		assert.NotContains(t, w.Body.String(), "admin")
	})

	t.Run("preserves when internal secret matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Internal-Secret", "internal-shared-secret")
		req.Header.Set("X-User-ID", "trusted-1")
		req.Header.Set("X-User-Role", "admin")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "trusted-1")
		assert.Contains(t, w.Body.String(), "admin")
	})
}
