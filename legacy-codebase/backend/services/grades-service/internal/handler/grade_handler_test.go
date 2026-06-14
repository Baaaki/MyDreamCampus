package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/grades-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// SubmitScoreRequest carries instructor-controlled scoring input. Pin the
// gin binding rules — the score nullability is intentional (absent students
// have score=nil with is_absent=true), so missing/zero behaviours need a
// regression net.

func TestSubmitScoreRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	score := 87.5

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid score submission",
			body: dto.SubmitScoreRequest{
				RegistrationID: uuid.New(),
				Slug:           "midterm",
				Score:          &score,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "absent submission with nil score",
			body: dto.SubmitScoreRequest{
				RegistrationID: uuid.New(),
				Slug:           "midterm",
				Score:          nil,
				IsAbsent:       true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing registration_id",
			body: map[string]any{
				"slug":  "midterm",
				"score": 87.5,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty slug",
			body: dto.SubmitScoreRequest{
				RegistrationID: uuid.New(),
				Slug:           "",
				Score:          &score,
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
			router.POST("/grades/submit", func(c *gin.Context) {
				var req dto.SubmitScoreRequest
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

			req := httptest.NewRequest(http.MethodPost, "/grades/submit", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}
