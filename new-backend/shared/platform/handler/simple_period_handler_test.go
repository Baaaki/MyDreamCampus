package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Init("test")
	gin.SetMode(gin.TestMode)
}

func TestSimplePeriodHandler_Validation(t *testing.T) {
	h := NewSimplePeriodHandler(nil, nil, nil)
	r := gin.New()
	r.POST("/periods", h.CreatePeriod)

	t.Run("invalid json body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/periods", bytes.NewBufferString("{broken json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
	})

	t.Run("period_end before period_start", func(t *testing.T) {
		body := dto.SimpleCreatePeriodRequest{
			Semester:    "2025-2026-Fall",
			PeriodStart: time.Now().Add(24 * time.Hour),
			PeriodEnd:   time.Now(),
		}
		raw, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/periods", bytes.NewBuffer(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp["code"])
		assert.Contains(t, resp["error"], "period_end must be after period_start")
	})

	t.Run("invalid period_type validation helper", func(t *testing.T) {
		assert.True(t, repository.IsValidPeriodType(repository.PeriodTypeCatalog))
		assert.True(t, repository.IsValidPeriodType(repository.PeriodTypeEnrollment))
		assert.True(t, repository.IsValidPeriodType(repository.PeriodTypeGrading))
		assert.True(t, repository.IsValidPeriodType(repository.PeriodTypeAttendance))
		assert.False(t, repository.IsValidPeriodType("unknown_type"))
		assert.False(t, repository.IsValidPeriodType(""))
	})
}
