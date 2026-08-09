package dto

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validateBinding[T any](t *testing.T, body any) error {
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

	if w.Code == 200 {
		return nil
	}
	return assert.AnError
}

func TestCreateStudentRequest_Validation(t *testing.T) {
	valid := map[string]any{
		"student_number":  "20240001",
		"first_name":      "Ada",
		"last_name":       "Lovelace",
		"email":           "ada@uni.tr",
		"faculty":         "Engineering",
		"department":      "CS",
		"enrollment_year": 2024,
		"class_level":     1,
	}

	t.Run("valid request", func(t *testing.T) {
		assert.NoError(t, validateBinding[CreateStudentRequest](t, valid))
	})

	missing := []string{"student_number", "first_name", "last_name", "email", "faculty", "department"}
	for _, field := range missing {
		t.Run("missing "+field, func(t *testing.T) {
			body := copyMap(valid)
			delete(body, field)
			assert.Error(t, validateBinding[CreateStudentRequest](t, body))
		})
	}

	t.Run("invalid email", func(t *testing.T) {
		body := copyMap(valid)
		body["email"] = "not-an-email"
		assert.Error(t, validateBinding[CreateStudentRequest](t, body))
	})

	t.Run("enrollment_year too small", func(t *testing.T) {
		body := copyMap(valid)
		body["enrollment_year"] = 1800
		assert.Error(t, validateBinding[CreateStudentRequest](t, body))
	})

	t.Run("class_level out of range", func(t *testing.T) {
		body := copyMap(valid)
		body["class_level"] = 10
		assert.Error(t, validateBinding[CreateStudentRequest](t, body))
	})
}

func TestUpdateStudentRequest_OptionalFields(t *testing.T) {
	t.Run("empty patch is valid", func(t *testing.T) {
		assert.NoError(t, validateBinding[UpdateStudentRequest](t, map[string]any{}))
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		assert.Error(t, validateBinding[UpdateStudentRequest](t,
			map[string]any{"status": "rejected"}))
	})

	t.Run("valid statuses accepted", func(t *testing.T) {
		for _, s := range []string{"active", "graduated", "suspended", "withdrawn"} {
			assert.NoError(t,
				validateBinding[UpdateStudentRequest](t, map[string]any{"status": s}),
				"status %q must be accepted", s)
		}
	})

	t.Run("class_level out of range rejected", func(t *testing.T) {
		assert.Error(t, validateBinding[UpdateStudentRequest](t,
			map[string]any{"class_level": 0}))
		assert.Error(t, validateBinding[UpdateStudentRequest](t,
			map[string]any{"class_level": 7}))
	})
}

func TestSortOptions_AllowedFieldsAndOrders(t *testing.T) {
	t.Run("invalid field rejected", func(t *testing.T) {
		assert.Error(t, validateBinding[SortOptions](t,
			map[string]any{"field": "ssn"}))
	})

	t.Run("invalid order rejected", func(t *testing.T) {
		assert.Error(t, validateBinding[SortOptions](t,
			map[string]any{"field": "last_name", "order": "sideways"}))
	})

	t.Run("valid combination accepted", func(t *testing.T) {
		assert.NoError(t, validateBinding[SortOptions](t,
			map[string]any{"field": "enrollment_year", "order": "desc"}))
	})
}

func TestPaginationOptions_LimitBounds(t *testing.T) {
	t.Run("limit > 100 rejected", func(t *testing.T) {
		assert.Error(t, validateBinding[PaginationOptions](t,
			map[string]any{"limit": 500}))
	})
	t.Run("limit zero accepted (omitempty)", func(t *testing.T) {
		assert.NoError(t, validateBinding[PaginationOptions](t,
			map[string]any{"limit": 0}))
	})
}

func TestBulkAdvisorAssignRequest_Validation(t *testing.T) {
	t.Run("requires at least one student id", func(t *testing.T) {
		assert.Error(t, validateBinding[BulkAdvisorAssignRequest](t,
			map[string]any{
				"student_ids": []uuid.UUID{},
				"advisor_id":  uuid.New(),
			}))
	})

	t.Run("requires advisor id", func(t *testing.T) {
		assert.Error(t, validateBinding[BulkAdvisorAssignRequest](t,
			map[string]any{"student_ids": []uuid.UUID{uuid.New()}}))
	})

	t.Run("happy path", func(t *testing.T) {
		assert.NoError(t, validateBinding[BulkAdvisorAssignRequest](t,
			map[string]any{
				"student_ids": []uuid.UUID{uuid.New(), uuid.New()},
				"advisor_id":  uuid.New(),
			}))
	})
}

func TestStudentResponse_OmitsAdvisorWhenAbsent(t *testing.T) {
	r := StudentResponse{ID: "1", StudentNumber: "1", Status: "active"}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "advisor_id")
	assert.NotContains(t, string(data), "advisor_name")
}

func TestSearchPaginationResponse_OmitsBlankCursor(t *testing.T) {
	r := SearchPaginationResponse{HasMore: false, TotalCount: 0}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "next_cursor")
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
