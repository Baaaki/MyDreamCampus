package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
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

func runSelfOrRole(t *testing.T, role, userID, target string) int {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", role)
		c.Set("user_id", userID)
		c.Next()
	})
	r.GET("/staff/:id", RequireSelfOrRole("id", "admin"), func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/staff/"+target, nil))
	return w.Code
}

func TestRequireSelfOrRole_OwnID_Allows(t *testing.T) {
	id := "5f0c7b1e-2d3a-4b5c-8d9e-0f1a2b3c4d5e"
	assert.Equal(t, 200, runSelfOrRole(t, "teacher", id, id))
}

func TestRequireSelfOrRole_OwnIDDifferentCase_Allows(t *testing.T) {
	assert.Equal(t, 200, runSelfOrRole(t, "teacher",
		"5f0c7b1e-2d3a-4b5c-8d9e-0f1a2b3c4d5e", "5F0C7B1E-2D3A-4B5C-8D9E-0F1A2B3C4D5E"))
}

func TestRequireSelfOrRole_OtherID_Denies(t *testing.T) {
	assert.Equal(t, 403, runSelfOrRole(t, "teacher",
		"5f0c7b1e-2d3a-4b5c-8d9e-0f1a2b3c4d5e", "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"))
}

func TestRequireSelfOrRole_AllowedRole_Allows(t *testing.T) {
	assert.Equal(t, 200, runSelfOrRole(t, "admin",
		"5f0c7b1e-2d3a-4b5c-8d9e-0f1a2b3c4d5e", "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"))
}

func TestRequireSelfOrRole_MalformedTarget_Denies(t *testing.T) {
	assert.Equal(t, 403, runSelfOrRole(t, "student", "", ""+"not-a-uuid"))
}

func TestRequireSuperAdmin(t *testing.T) {
	runWithSuperAdmin := func(t *testing.T, isSuper any) *httptest.ResponseRecorder {
		t.Helper()
		require.NoError(t, logger.Init("test"))
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if isSuper != nil {
				c.Set("is_superadmin", isSuper)
			}
			c.Next()
		})
		r.Use(RequireSuperAdmin())
		r.GET("/", func(c *gin.Context) { c.Status(200) })

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("superadmin allowed", func(t *testing.T) {
		w := runWithSuperAdmin(t, true)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("false denied", func(t *testing.T) {
		w := runWithSuperAdmin(t, false)
		assert.Equal(t, 403, w.Code)
	})

	t.Run("missing denied", func(t *testing.T) {
		w := runWithSuperAdmin(t, nil)
		assert.Equal(t, 403, w.Code)
	})
}

