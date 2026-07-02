package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runWithInternalSecret(t *testing.T, secret, header string) *httptest.ResponseRecorder {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireInternalSecret(secret))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/", nil)
	if header != "" {
		req.Header.Set("X-Internal-Secret", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireInternalSecret_AllowsMatchingSecret(t *testing.T) {
	w := runWithInternalSecret(t, "s3cret", "s3cret")
	assert.Equal(t, 200, w.Code)
}

func TestRequireInternalSecret_DeniesWrongSecret(t *testing.T) {
	w := runWithInternalSecret(t, "s3cret", "wrong")
	assert.Equal(t, 401, w.Code)
}

func TestRequireInternalSecret_DeniesMissingHeader(t *testing.T) {
	w := runWithInternalSecret(t, "s3cret", "")
	assert.Equal(t, 401, w.Code)
}

func TestRequireInternalSecret_PanicsOnEmptySecret(t *testing.T) {
	assert.Panics(t, func() { RequireInternalSecret("") },
		"empty secret must fail at startup, not leave the route open")
}
