package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRoleContext(role any) gin.HandlerFunc {
	return func(c *gin.Context) {
		if role != nil {
			c.Set("role", role)
		}
		c.Next()
	}
}

func runWithRole(t *testing.T, role any, mw gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(setRoleContext(role))
	r.Use(mw)
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireRole_AllowsListedRole(t *testing.T) {
	w := runWithRole(t, "teacher", RequireRole("teacher", "admin"))
	assert.Equal(t, 200, w.Code)
}

func TestRequireRole_DeniesUnlistedRole(t *testing.T) {
	w := runWithRole(t, "student", RequireRole("teacher", "admin"))
	assert.Equal(t, 403, w.Code)
}

func TestRequireRole_NoRoleInContext(t *testing.T) {
	w := runWithRole(t, nil, RequireRole("admin"))
	assert.Equal(t, 403, w.Code,
		"missing role in context must result in forbidden")
}

func TestRequireAdmin(t *testing.T) {
	t.Run("admin allowed", func(t *testing.T) {
		w := runWithRole(t, "admin", RequireAdmin())
		assert.Equal(t, 200, w.Code)
	})
	t.Run("non-admin denied", func(t *testing.T) {
		w := runWithRole(t, "teacher", RequireAdmin())
		assert.Equal(t, 403, w.Code)
	})
}

func TestRequireTeacherOrAdmin(t *testing.T) {
	for _, role := range []string{"teacher", "admin"} {
		w := runWithRole(t, role, RequireTeacherOrAdmin())
		assert.Equal(t, 200, w.Code, "%s must be allowed", role)
	}

	w := runWithRole(t, "student", RequireTeacherOrAdmin())
	assert.Equal(t, 403, w.Code)
}

func TestRequireStudent(t *testing.T) {
	w := runWithRole(t, "student", RequireStudent())
	assert.Equal(t, 200, w.Code)

	w = runWithRole(t, "admin", RequireStudent())
	assert.Equal(t, 403, w.Code)
}
