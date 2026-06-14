package dto

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginationQuery_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var got PaginationQuery
	r.GET("/", func(c *gin.Context) {
		_ = c.ShouldBindQuery(&got)
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 1, got.Page, "default page = 1")
	assert.Equal(t, 20, got.Limit, "default limit = 20")
}

func TestPaginationQuery_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		query string
		ok    bool
	}{
		{"?page=2&limit=50", true},
		{"?page=0&limit=20", false},   // page must be >= 1
		{"?page=1&limit=0", false},    // limit must be >= 1
		{"?page=1&limit=101", false},  // limit max = 100
		{"?page=-1&limit=20", false},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				var q PaginationQuery
				if err := c.ShouldBindQuery(&q); err != nil {
					c.AbortWithStatus(400)
					return
				}
				c.Status(200)
			})
			req := httptest.NewRequest("GET", "/"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if tt.ok {
				assert.Equal(t, 200, w.Code, "expected accept")
			} else {
				assert.Equal(t, 400, w.Code, "expected reject")
			}
		})
	}
}

func TestErrorResponse_OmitsBlankMessage(t *testing.T) {
	e := ErrorResponse{Code: "X", Error: "x"}
	data, err := json.Marshal(e)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"message"`)
}

func TestMessageResponse_OmitsBlankID(t *testing.T) {
	m := MessageResponse{Message: "ok"}
	data, err := json.Marshal(m)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"id"`)
}
