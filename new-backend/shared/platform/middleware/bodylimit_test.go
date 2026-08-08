package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func runWithBodyLimit(t *testing.T, limit int64, body string) (*httptest.ResponseRecorder, error) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodySizeLimit(limit))

	var readErr error
	r.POST("/", func(c *gin.Context) {
		_, readErr = io.ReadAll(c.Request.Body)
		if readErr != nil {
			c.Status(413)
			return
		}
		c.Status(200)
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, readErr
}

func TestBodySizeLimit_AllowsBodyWithinLimit(t *testing.T) {
	w, err := runWithBodyLimit(t, 64, strings.Repeat("a", 32))
	assert.NoError(t, err)
	assert.Equal(t, 200, w.Code)
}

func TestBodySizeLimit_RejectsOversizedBody(t *testing.T) {
	w, err := runWithBodyLimit(t, 64, strings.Repeat("a", 128))
	assert.Error(t, err, "reading past the limit must fail")
	assert.Equal(t, 413, w.Code)
}
