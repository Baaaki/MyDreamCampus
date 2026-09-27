package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func makeDemoTestToken(role string, demo bool) string {
	now := time.Now()
	claims := &utils.Claims{
		UserID:       "demo-user-id",
		Role:         role,
		TokenVersion: 1,
		TokenType:    string(utils.AccessToken),
		JTI:          "demo-jti",
		Demo:         demo,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(utils.GetJWTSecret())
	return token
}

func setupDemoRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")

	r := gin.New()
	r.Use(DemoAdminNoDelete())
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.DELETE("/api/students/:id", ok)
	r.DELETE("/api/auth/sessions/:id", ok)
	r.PUT("/api/students/:id", ok)
	r.DELETE("/api/enrollment/student/programs", ok)
	return r
}

func demoRequest(r *gin.Engine, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDemoAdminNoDelete_DemoAdminDelete_Returns403(t *testing.T) {
	r := setupDemoRouter(t)

	w := demoRequest(r, http.MethodDelete, "/api/students/1", makeDemoTestToken("admin", true))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), DemoDeleteDisabledCode)
}

func TestDemoAdminNoDelete_CookieToken_Returns403(t *testing.T) {
	r := setupDemoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/students/1", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: makeDemoTestToken("admin", true)})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDemoAdminNoDelete_OtherRequests_PassThrough(t *testing.T) {
	r := setupDemoRouter(t)

	cases := []struct {
		name, method, path, token string
	}{
		{"demo admin edit", http.MethodPut, "/api/students/1", makeDemoTestToken("admin", true)},
		{"demo admin ends a session", http.MethodDelete, "/api/auth/sessions/1", makeDemoTestToken("admin", true)},
		{"regular admin delete", http.MethodDelete, "/api/students/1", makeDemoTestToken("admin", false)},
		{"demo student cancels own program", http.MethodDelete, "/api/enrollment/student/programs", makeDemoTestToken("student", true)},
		{"no token", http.MethodDelete, "/api/students/1", ""},
		{"invalid token", http.MethodDelete, "/api/students/1", "not-a-jwt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := demoRequest(r, tc.method, tc.path, tc.token)
			assert.Equal(t, http.StatusNoContent, w.Code)
		})
	}
}
