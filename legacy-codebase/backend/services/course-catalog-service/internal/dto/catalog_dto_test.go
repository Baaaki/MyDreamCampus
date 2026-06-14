package dto

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validateBinding[T any](t *testing.T, body any) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/", func(c *gin.Context) {
		var got T
		if err := c.ShouldBindJSON(&got); err != nil {
			c.AbortWithStatusJSON(400, gin.H{"err": err.Error()})
			return
		}
		c.Status(200)
	})

	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func TestCreateCourseRequest_Validation(t *testing.T) {
	valid := map[string]any{
		"course_code":      "CSE101",
		"name":             "Intro to CS",
		"faculty":          "Engineering",
		"department":       "Computer Science",
		"class_level":      1,
		"credits":          3,
		"course_type":      "mandatory",
	}

	t.Run("valid request", func(t *testing.T) {
		code, body := validateBinding[CreateCourseRequest](t, valid)
		assert.Equal(t, 200, code, body)
	})

	t.Run("course_code too short", func(t *testing.T) {
		body := copyMap(valid)
		body["course_code"] = "X"
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("course_type must be mandatory or elective", func(t *testing.T) {
		body := copyMap(valid)
		body["course_type"] = "optional"
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("class_level out of range", func(t *testing.T) {
		body := copyMap(valid)
		body["class_level"] = 8
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("credits cannot exceed 30", func(t *testing.T) {
		body := copyMap(valid)
		body["credits"] = 50
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("invalid course_category rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["course_category"] = "fakecategory"
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["status"] = "purgatory"
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("status omitted is ok", func(t *testing.T) {
		body := copyMap(valid)
		delete(body, "status")
		code, _ := validateBinding[CreateCourseRequest](t, body)
		assert.Equal(t, 200, code)
	})
}

func TestUpdateCourseRequest_PatchSemantics(t *testing.T) {
	t.Run("empty body is valid (full no-op patch)", func(t *testing.T) {
		code, _ := validateBinding[UpdateCourseRequest](t, map[string]any{})
		assert.Equal(t, 200, code)
	})

	t.Run("partial update only validates provided fields", func(t *testing.T) {
		code, _ := validateBinding[UpdateCourseRequest](t,
			map[string]any{"credits": 4, "status": "active"})
		assert.Equal(t, 200, code)
	})

	t.Run("invalid status still rejected", func(t *testing.T) {
		code, _ := validateBinding[UpdateCourseRequest](t,
			map[string]any{"status": "deleted"})
		assert.Equal(t, 400, code)
	})
}

func TestCourseResponse_OmitsOptionalFields(t *testing.T) {
	r := CourseResponse{CourseCode: "X", Name: "Y", Status: "active"}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	for _, omit := range []string{"offering_unit", "ects", "purpose", "description", "syllabus"} {
		assert.NotContains(t, str, omit, "must omit %s when empty", omit)
	}
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
