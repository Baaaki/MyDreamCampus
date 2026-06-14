package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newVerifyOnlyHandler creates a handler that only exercises Verify(),
// which doesn't depend on the AuthService internals.
func newVerifyOnlyHandler() *AuthHandler {
	return &AuthHandler{
		authService: nil,
		config:      &config.Config{},
	}
}

func TestVerify_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newVerifyOnlyHandler()

	r := gin.New()
	r.GET("/verify", h.Verify)

	req := httptest.NewRequest("GET", "/verify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestVerify_AuthenticatedSetsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newVerifyOnlyHandler()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "u-42")
		c.Set("role", "teacher")
		c.Set("department", "CS")
		c.Next()
	})
	r.GET("/verify", h.Verify)

	req := httptest.NewRequest("GET", "/verify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "u-42", w.Header().Get("X-User-ID"))
	assert.Equal(t, "teacher", w.Header().Get("X-User-Role"))
	assert.Equal(t, "CS", w.Header().Get("X-User-Department"))
}

func TestVerify_OmitsDepartmentHeaderIfBlank(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newVerifyOnlyHandler()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "u-43")
		c.Set("role", "admin")
		// no department
		c.Next()
	})
	r.GET("/verify", h.Verify)

	req := httptest.NewRequest("GET", "/verify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "u-43", w.Header().Get("X-User-ID"))
	assert.Equal(t, "admin", w.Header().Get("X-User-Role"))
	assert.Empty(t, w.Header().Get("X-User-Department"),
		"Department header must be omitted when not set")
}

func TestVerify_BlocksWhenForcePasswordChange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newVerifyOnlyHandler()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "u-44")
		c.Set("role", "student")
		c.Set("force_password_change", true)
		c.Next()
	})
	r.GET("/verify", h.Verify)

	req := httptest.NewRequest("GET", "/verify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code)
	assert.Contains(t, w.Body.String(), "FORCE_PASSWORD_CHANGE")
}
