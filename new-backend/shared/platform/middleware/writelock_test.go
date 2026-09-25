package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

type mockWriteLockStore struct {
	locked bool
	err    error
}

func (m *mockWriteLockStore) IsWriteLocked(_ context.Context) (bool, error) {
	return m.locked, m.err
}

func setupWriteLockRouter(t *testing.T, store WriteLockStore) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	SetWriteLockStore(store)
	t.Cleanup(func() { SetWriteLockStore(nil) })

	r := gin.New()
	r.Use(WriteLock())

	handler := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

	r.GET("/api/courses", handler)
	r.POST("/api/courses", handler)
	r.PUT("/api/courses/1", handler)
	r.PATCH("/api/courses/1", handler)
	r.DELETE("/api/courses/1", handler)

	// Exempt routes
	r.POST("/internal/sync", handler)
	r.POST("/api/auth/login", handler)
	r.POST("/api/auth/refresh", handler)
	r.POST("/api/auth/logout", handler)
	r.GET("/health", handler)
	r.GET("/ready", handler)

	return r
}

func makeTestToken(superAdmin bool) string {
	now := time.Now()
	claims := &utils.Claims{
		UserID:       "test-user-id",
		Role:         "admin",
		TokenVersion: 1,
		TokenType:    string(utils.AccessToken),
		JTI:          "test-jti",
		SuperAdmin:   superAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(utils.GetJWTSecret())
	return tokenString
}

func TestWriteLock_WhenUnlocked_AllowsWrites(t *testing.T) {
	r := setupWriteLockRouter(t, &mockWriteLockStore{locked: false})

	req := httptest.NewRequest(http.MethodPost, "/api/courses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("X-System-Editing"))
}

func TestWriteLock_WhenLocked_AllowsReadsWithHeader(t *testing.T) {
	r := setupWriteLockRouter(t, &mockWriteLockStore{locked: true})

	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1", w.Header().Get("X-System-Editing"))
}

func TestWriteLock_WhenLocked_BlocksPublicWritesWith503(t *testing.T) {
	r := setupWriteLockRouter(t, &mockWriteLockStore{locked: true})

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range methods {
		path := "/api/courses"
		if method != http.MethodPost {
			path = "/api/courses/1"
		}
		req := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code, "method %s should be 503", method)
		assert.Equal(t, "1", w.Header().Get("X-System-Editing"))
		assert.Contains(t, w.Body.String(), "SYSTEM_EDITING")
	}
}

func TestWriteLock_WhenLocked_AllowsExemptRoutes(t *testing.T) {
	r := setupWriteLockRouter(t, &mockWriteLockStore{locked: true})

	exempts := []string{
		"/internal/sync",
		"/api/auth/login",
		"/api/auth/refresh",
		"/api/auth/logout",
	}

	for _, path := range exempts {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "exempt path %s should be allowed", path)
	}
}

func TestWriteLock_WhenLocked_AllowsSuperAdminWrites(t *testing.T) {
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")

	superAdminToken := makeTestToken(true)
	regularToken := makeTestToken(false)

	r := setupWriteLockRouter(t, &mockWriteLockStore{locked: true})

	// Regular admin is blocked
	req := httptest.NewRequest(http.MethodPost, "/api/courses", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Super admin is permitted
	req2 := httptest.NewRequest(http.MethodPost, "/api/courses", nil)
	req2.Header.Set("Authorization", "Bearer "+superAdminToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestWriteLock_FailsOpenOnRedisError(t *testing.T) {
	r := setupWriteLockRouter(t, &mockWriteLockStore{locked: true, err: errors.New("redis timeout")})

	req := httptest.NewRequest(http.MethodPost, "/api/courses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Failing open means request is permitted
	assert.Equal(t, http.StatusOK, w.Code)
}
