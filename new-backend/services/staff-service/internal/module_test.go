package staff

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The directory holds every administrative employee's phone and office, so
// the guard is the feature: no role but admin may read or write it. No
// database is wired — a request that got past the guard would panic on the
// nil pool, which fails the test just as loudly.
func TestAdminStaffRoutes_NonAdmin_Rejected(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")
	platformMiddleware.SetBlacklistChecker(nil)

	r := gin.New()
	cfg := &config.Config{}
	cfg.Server.InternalSecret = "test-internal-secret-minimum-32-characters"
	New(cfg, nil).RegisterPublicRoutes(r)

	token := func(role string) string {
		tok, _, err := utils.GenerateAccessToken("user-1", role, "", 0)
		require.NoError(t, err)
		return "Bearer " + tok
	}

	cases := []struct {
		name, method, path, auth string
		want                     int
	}{
		{"list without token", http.MethodGet, "/api/admin-staff", "", http.StatusUnauthorized},
		{"list as student", http.MethodGet, "/api/admin-staff", token("student"), http.StatusForbidden},
		{"list as teacher", http.MethodGet, "/api/admin-staff", token("teacher"), http.StatusForbidden},
		{"read as teacher", http.MethodGet, "/api/admin-staff/0190a1b2-0000-7000-8000-000000000001", token("teacher"), http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.want, w.Code)
		})
	}
}
