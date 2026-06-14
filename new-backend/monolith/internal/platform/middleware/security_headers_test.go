package middleware

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_AlwaysSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Unsetenv("ENVIRONMENT")

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	h := w.Header()
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
	assert.NotEmpty(t, h.Get("Permissions-Policy"))
	assert.Empty(t, h.Get("Strict-Transport-Security"),
		"HSTS must NOT be set outside production")
}

func TestSecurityHeaders_HSTSOnlyInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENVIRONMENT", "production")

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=")
	assert.Contains(t, hsts, "includeSubDomains")
}
