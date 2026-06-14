package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Mirrors the auth-service/internal/handler validation harness — exercises gin
// binding rules directly, no live repository. Keeps the role enum and email
// format pinned so a `binding:"required,oneof=teacher"` regression is caught
// in CI rather than at the first failing onboarding.

func TestCreateStaffRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid teacher",
			body: dto.CreateStaffRequest{
				Email:     "jane@university.edu.tr",
				FirstName: "Jane",
				LastName:  "Doe",
				Role:      "teacher",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid email format",
			body: dto.CreateStaffRequest{
				Email:     "not-an-email",
				FirstName: "Jane",
				LastName:  "Doe",
				Role:      "teacher",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "role outside enum is rejected",
			body: dto.CreateStaffRequest{
				Email:     "jane@university.edu.tr",
				FirstName: "Jane",
				LastName:  "Doe",
				Role:      "admin",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing first_name",
			body: dto.CreateStaffRequest{
				Email:    "jane@university.edu.tr",
				LastName: "Doe",
				Role:     "teacher",
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
			router.POST("/staff", func(c *gin.Context) {
				var req dto.CreateStaffRequest
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

			req := httptest.NewRequest(http.MethodPost, "/staff", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code,
				"binding rules on CreateStaffRequest are part of the API contract — see staff_dto.go")
		})
	}
}
