package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFProtection_SkipsSafeMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFProtection())
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	r.HEAD("/", func(c *gin.Context) { c.Status(200) })

	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		req := httptest.NewRequest(method, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusForbidden, w.Code,
			"%s must not be CSRF-checked", method)
	}
}

func TestCSRFProtection_SkipsAuthorizationHeaderRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFProtection())
	r.POST("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "API/mobile clients with Authorization bypass CSRF")
}

func TestCSRFProtection_RejectsMissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFProtection())
	r.POST("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "CSRF")
}

func TestCSRFProtection_RejectsMismatchedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFProtection())
	r.POST("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "cookie-value"})
	req.Header.Set("X-CSRF-Token", "header-value-different")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFProtection_AcceptsMatchingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFProtection())
	r.POST("/", func(c *gin.Context) { c.Status(200) })

	const token = "abc123secrettoken"
	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	req.Header.Set("X-CSRF-Token", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSetCSRFToken_SetsCookieIfMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SetCSRFToken(false))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	var csrf *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrf = c
		}
	}
	if assert.NotNil(t, csrf, "csrf_token cookie must be set") {
		assert.NotEmpty(t, csrf.Value)
		assert.False(t, csrf.HttpOnly, "csrf_token must NOT be HttpOnly so JS can read it")
		assert.Len(t, csrf.Value, 64, "32 bytes hex-encoded = 64 chars")
	}
}

func TestSetCSRFToken_SkipsIfPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SetCSRFToken(false))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "existing-value"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	for _, c := range w.Result().Cookies() {
		assert.NotEqual(t, "csrf_token", c.Name,
			"existing CSRF cookie must not be overwritten")
	}
}
