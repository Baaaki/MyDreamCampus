package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLivenessHandler_Always200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", LivenessHandler("test-service"))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "alive", body["status"])
	assert.Equal(t, "test-service", body["service"])
}

func TestReadinessHandler_AllChecksPass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ready", ReadinessHandler("svc", map[string]HealthCheck{
		"db":    func(_ context.Context) error { return nil },
		"redis": func(_ context.Context) error { return nil },
	}))

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ready", body["status"])
	checks := body["checks"].(map[string]any)
	assert.Equal(t, "up", checks["db"])
	assert.Equal(t, "up", checks["redis"])
}

func TestReadinessHandler_OneFailure_503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ready", ReadinessHandler("svc", map[string]HealthCheck{
		"db":       func(_ context.Context) error { return nil },
		"rabbitmq": func(_ context.Context) error { return errors.New("connection refused") },
	}))

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 503, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])
	checks := body["checks"].(map[string]any)
	assert.Equal(t, "up", checks["db"])
	assert.Contains(t, checks["rabbitmq"], "down:")
	assert.Contains(t, checks["rabbitmq"], "connection refused")
}

func TestReadinessHandler_NoChecksAlwaysReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ready", ReadinessHandler("svc", map[string]HealthCheck{}))

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
