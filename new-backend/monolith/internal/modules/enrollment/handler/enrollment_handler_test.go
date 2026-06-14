package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Pin the gin binding contract on CreateEnrollmentRequest. The deeper rules
// (max courses, dedup, dept/level) live in service.validateCourseSelection
// and are tested in service/validators_test.go — this file only covers the
// HTTP-layer rules.

func TestCreateEnrollmentRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid minimal",
			body: dto.CreateEnrollmentRequest{
				Semester:  "2025-2026-Fall",
				CourseIDs: []uuid.UUID{uuid.New()},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing semester",
			body: dto.CreateEnrollmentRequest{
				CourseIDs: []uuid.UUID{uuid.New()},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty course_ids fails min=1",
			body: dto.CreateEnrollmentRequest{
				Semester:  "2025-2026-Fall",
				CourseIDs: []uuid.UUID{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing course_ids field entirely",
			body: map[string]any{
				"semester": "2025-2026-Fall",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-JSON body",
			body:           "{not valid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/enrollment", func(c *gin.Context) {
				var req dto.CreateEnrollmentRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			var raw []byte
			if s, ok := tc.body.(string); ok {
				raw = []byte(s)
			} else {
				raw, _ = json.Marshal(tc.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/enrollment", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}
