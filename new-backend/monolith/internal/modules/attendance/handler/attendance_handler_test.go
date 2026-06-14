package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// CreateSessionRequest carries instructor-controlled session bounds — week
// number, duration, session type. Pinning the binding rules here prevents an
// off-spec week=0 / duration=999 / session_type="" from leaking into the
// service layer where the DB constraints would catch it later (with worse UX).

func TestCreateSessionRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid theory session",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      5,
				DurationMinutes: 50,
				SessionType:     "theory",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "week 0 is rejected",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      0,
				DurationMinutes: 50,
				SessionType:     "theory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "week beyond cap (15) is rejected",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      15,
				DurationMinutes: 50,
				SessionType:     "theory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "duration below 5 minutes is rejected",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      5,
				DurationMinutes: 4,
				SessionType:     "theory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "duration above 120 minutes is rejected",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      5,
				DurationMinutes: 121,
				SessionType:     "theory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "session_type outside enum is rejected",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      5,
				DurationMinutes: 50,
				SessionType:     "tutorial",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "session_type 'lab' is accepted",
			body: dto.CreateSessionRequest{
				CourseID:        uuid.New(),
				WeekNumber:      5,
				DurationMinutes: 50,
				SessionType:     "lab",
			},
			expectedStatus: http.StatusOK,
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
			router.POST("/attendance/sessions", func(c *gin.Context) {
				var req dto.CreateSessionRequest
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

			req := httptest.NewRequest(http.MethodPost, "/attendance/sessions", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}
