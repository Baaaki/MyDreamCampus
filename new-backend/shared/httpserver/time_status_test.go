package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeStatusTestSecret = "time-status-test-secret-at-least-32-bytes"

// groupUseModule mirrors staff, which calls Use on the module group itself.
type groupUseModule struct{ calls *int }

func (groupUseModule) Name() string { return "staff" }

func (m groupUseModule) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(func(c *gin.Context) { *m.calls++ })
	rg.GET("/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
}

func timeStatusServer(t *testing.T) (*Server, *int) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	utils.InitJWTSecret(timeStatusTestSecret)
	s := NewServer(&config.Config{}, "staff")
	calls := 0
	s.RegisterModules(groupUseModule{calls: &calls})
	return s, &calls
}

func getTimeStatus(t *testing.T, s *Server, role string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/staff/admin/time/status", nil)
	if role != "" {
		token, _, err := utils.GenerateAccessTokenWithSecret("u-1", role, "", 1, []byte(timeStatusTestSecret), 15)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.Engine().ServeHTTP(w, req)
	return w
}

func TestRegisterModules_TimeStatus_AdminOnly(t *testing.T) {
	s, _ := timeStatusServer(t)

	assert.Equal(t, http.StatusUnauthorized, getTimeStatus(t, s, "").Code)
	assert.Equal(t, http.StatusForbidden, getTimeStatus(t, s, "student").Code)

	w := getTimeStatus(t, s, "admin")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"service":"staff"`)
}

func TestRegisterModules_TimeStatus_SkipsModuleGroupMiddleware(t *testing.T) {
	s, calls := timeStatusServer(t)

	require.Equal(t, http.StatusOK, getTimeStatus(t, s, "admin").Code)
	assert.Equal(t, 0, *calls, "a module's group.Use must not run on the shared status route")

	req := httptest.NewRequest("GET", "/api/staff/123", nil)
	s.Engine().ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, 1, *calls, "the module's own routes keep its chain")
}
