package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/student/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Mirrors the auth-service handler validation harness — exercises gin binding
// rules on CreateStudentRequest. Keeps the email format and required-field
// contract pinned so admin-driven onboarding flows fail fast on bad CSV rows
// rather than after a DB round-trip.

func TestCreateStudentRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid",
			body: dto.CreateStudentRequest{
				StudentNumber:  "20210001",
				FirstName:      "Ahmet",
				LastName:       "Yilmaz",
				Email:          "ahmet@university.edu.tr",
				Faculty:        "Engineering",
				Department:     "CS",
				EnrollmentYear: 2021,
				ClassLevel:     1,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing student_number",
			body: dto.CreateStudentRequest{
				FirstName:      "Ahmet",
				LastName:       "Yilmaz",
				Email:          "ahmet@university.edu.tr",
				Faculty:        "Engineering",
				Department:     "CS",
				EnrollmentYear: 2021,
				ClassLevel:     1,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			body: dto.CreateStudentRequest{
				StudentNumber:  "20210001",
				FirstName:      "Ahmet",
				LastName:       "Yilmaz",
				Email:          "not-an-email",
				Faculty:        "Engineering",
				Department:     "CS",
				EnrollmentYear: 2021,
				ClassLevel:     1,
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
			router.POST("/students", func(c *gin.Context) {
				var req dto.CreateStudentRequest
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

			req := httptest.NewRequest(http.MethodPost, "/students", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}

// The deny paths return before the service is touched, so a nil service is
// enough — reaching it would panic and fail the test.
func TestGetStudentByIDForCaller_DenyPaths_Return403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	self := "2b1f6c1e-8f0a-4a57-9a53-3f3b1f1d2a10"
	other := "7c3d9e2a-1b4f-4e6a-8d2c-5a6b7c8d9e0f"

	tests := []struct {
		name     string
		role     string
		targetID string
	}{
		{name: "student reading another student", role: "student", targetID: other},
		{name: "student with malformed id", role: "student", targetID: "not-a-uuid"},
		{name: "unknown role", role: "guest", targetID: self},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &StudentHandler{}
			router := gin.New()
			router.GET("/students/:id", func(c *gin.Context) {
				c.Set("user_id", self)
				c.Set("role", tc.role)
				h.GetStudentByIDForCaller(c)
			})

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/students/"+tc.targetID, nil))

			assert.Equal(t, http.StatusForbidden, resp.Code)
		})
	}
}
