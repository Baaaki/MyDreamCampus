package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/httpserver"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-minimum-32-characters-long-aaa"

// Only the student who owns a payment may read or confirm it. No database
// is wired — a request that got past the guard would panic on the nil pool,
// which fails the test just as loudly.
func TestPaymentRoutes_NonStudent_Rejected(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret(testSecret)
	platformMiddleware.SetBlacklistChecker(nil)

	cfg := &config.Config{}
	cfg.Server.InternalSecret = "test-internal-secret-minimum-32-characters"
	s := httpserver.NewServer(cfg, "payment")
	s.RegisterModules(New(cfg, logger.Log, nil))

	token := func(role string) string {
		tok, _, err := utils.GenerateAccessToken("0190a1b2-0000-7000-8000-00000000000a", role, "", 0)
		require.NoError(t, err)
		return "Bearer " + tok
	}
	const payment = "/api/payments/0190a1b2-0000-7000-8000-00000000000b"

	cases := []struct {
		name, method, path, auth string
		want                     int
	}{
		{"read without token", http.MethodGet, payment, "", http.StatusUnauthorized},
		{"read as teacher", http.MethodGet, payment, token("teacher"), http.StatusForbidden},
		{"read as admin", http.MethodGet, payment, token("admin"), http.StatusForbidden},
		{"confirm without token", http.MethodPost, payment + "/confirm", "", http.StatusUnauthorized},
		{"confirm as admin", http.MethodPost, payment + "/confirm", token("admin"), http.StatusForbidden},
		// The static status route shares its segment with :payment_id.
		{"time status as admin", http.MethodGet, "/api/payments/admin/time/status", token("admin"), http.StatusOK},
		{"internal without secret", http.MethodPost, "/internal/payments/initiate", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			w := httptest.NewRecorder()
			s.Engine().ServeHTTP(w, req)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}
