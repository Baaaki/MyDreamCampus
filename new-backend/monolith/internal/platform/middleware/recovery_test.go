package middleware

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecovery_DoesNotLeakPanicDetails(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Recovery())
	r.GET("/boom", func(c *gin.Context) {
		panic("secret-internal-detail-12345")
	})

	req := httptest.NewRequest("GET", "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "secret-internal-detail-12345",
		"panic value MUST stay in logs only")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Internal server error", resp["message"])
	assert.Contains(t, resp, "error")
	assert.Contains(t, resp, "request_id")
}

func TestRecovery_PassesThroughNormalRequests(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Recovery())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestRecovery_IncludesRequestIDFromContext(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := logger.WithRequestIDValue(c.Request.Context(), "trace-xyz")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Use(Recovery())
	r.GET("/boom", func(c *gin.Context) {
		// sanity: request id is reachable
		assert.Equal(t, "trace-xyz", logger.GetRequestID(c.Request.Context()))
		panic(context.Canceled)
	})

	req := httptest.NewRequest("GET", "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "trace-xyz", resp["request_id"])
}
