package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/grades/internal/dto"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Init("test")
}

func TestProcessAppeal_AdminAuthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &GradeHandler{}

	t.Run("non-admin role returns 403 Forbidden", func(t *testing.T) {
		r := gin.New()
		r.POST("/appeal", func(c *gin.Context) {
			c.Set("role", "teacher")
			handler.ProcessAppeal(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/appeal", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "FORBIDDEN", resp.Code)
		assert.Equal(t, "İtirazları yalnızca yöneticiler işleyebilir", resp.Error)
	})

	t.Run("missing role in context returns 403 Forbidden", func(t *testing.T) {
		r := gin.New()
		r.POST("/appeal", handler.ProcessAppeal)

		req := httptest.NewRequest(http.MethodPost, "/appeal", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestGradeHandler_HandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &GradeHandler{}

	t.Run("known AppError returns its status and code", func(t *testing.T) {
		r := gin.New()
		r.GET("/test-err", func(c *gin.Context) {
			h.handleError(c, sharedErrors.ErrNotFound)
		})

		req := httptest.NewRequest(http.MethodGet, "/test-err", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp dto.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, sharedErrors.ErrNotFound.Code, resp.Code)
	})

	t.Run("unexpected error returns 500 internal server error", func(t *testing.T) {
		r := gin.New()
		r.GET("/test-err", func(c *gin.Context) {
			h.handleError(c, errors.New("raw unhandled database connection reset"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test-err", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp dto.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, sharedErrors.ErrInternal.Code, resp.Code)
	})
}
