package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// CreateCourseRequest binding rules: course code length, name/faculty/dept
// minimums. The catalog is a public-facing canonical source for every
// downstream service — typos here propagate everywhere.

func TestCreateCourseRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid",
			body: dto.CreateCourseRequest{
				CourseCode: "CS101",
				Name:       "Introduction to Computer Science",
				Faculty:    "Engineering",
				Department: "Computer Science",
				ClassLevel: 1,
				CourseType: "mandatory",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "course_code too short (1 char) — min=2",
			body: dto.CreateCourseRequest{
				CourseCode: "C",
				Name:       "Some name",
				Faculty:    "Engineering",
				Department: "Computer Science",
				ClassLevel: 1,
				CourseType: "mandatory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "name too short (2 char) — min=3",
			body: dto.CreateCourseRequest{
				CourseCode: "CS101",
				Name:       "AB",
				Faculty:    "Engineering",
				Department: "Computer Science",
				ClassLevel: 1,
				CourseType: "mandatory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "course_type outside enum",
			body: dto.CreateCourseRequest{
				CourseCode: "CS101",
				Name:       "Intro",
				Faculty:    "Engineering",
				Department: "Computer Science",
				ClassLevel: 1,
				CourseType: "free-elective",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "class_level above cap (7) is rejected — max=6",
			body: dto.CreateCourseRequest{
				CourseCode: "CS101",
				Name:       "Intro",
				Faculty:    "Engineering",
				Department: "Computer Science",
				ClassLevel: 7,
				CourseType: "mandatory",
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
			router.POST("/courses", func(c *gin.Context) {
				var req dto.CreateCourseRequest
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

			req := httptest.NewRequest(http.MethodPost, "/courses", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}
