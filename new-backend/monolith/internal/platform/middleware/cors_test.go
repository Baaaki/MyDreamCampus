package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newCorsRouter(t *testing.T, mw gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	r.POST("/", func(c *gin.Context) { c.Status(200) })
	return r
}

func TestCORS_AllowsListedOrigin(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	r := newCorsRouter(t, CORS())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, w.Header().Get("Vary"), "Origin")
}

func TestCORS_RejectsUnknownOrigin(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	r := newCorsRouter(t, CORS())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"),
		"untrusted origin must NOT be reflected")
}

func TestCORS_PreflightReturns204(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	r := newCorsRouter(t, CORS())

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)
}

func TestCORS_DevDefaultsForLocalhost(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	r := newCorsRouter(t, CORS())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PanicsInProductionWithoutAllowlist(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	assert.Panics(t, func() {
		_ = CORS()
	}, "missing allowlist in prod must panic to surface misconfiguration")
}

func TestCORSWithOrigins_ExplicitList(t *testing.T) {
	r := newCorsRouter(t, CORSWithOrigins([]string{"https://only.example.com"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://only.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "https://only.example.com", w.Header().Get("Access-Control-Allow-Origin"))

	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://other.example.com")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSForMobile_NoOriginAllowsThrough(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	r := newCorsRouter(t, CORSForMobile())

	// Native mobile clients send no Origin header
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORSForMobile_DevAllowsExpAndLocalhost(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	r := newCorsRouter(t, CORSForMobile())

	for _, origin := range []string{
		"http://localhost:8081",
		"exp://192.168.1.10:8081",
	} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"),
			"dev should allow %s", origin)
	}
}

func TestCORSForMobile_ProdRequiresAllowlist(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://campus.example.com")

	r := newCorsRouter(t, CORSForMobile())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"),
		"localhost must NOT be allowed in production")

	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://campus.example.com")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "https://campus.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}
