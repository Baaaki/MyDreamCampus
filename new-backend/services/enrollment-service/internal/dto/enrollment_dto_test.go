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

func TestCreateEnrollmentRequest_Validation(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		body := map[string]any{
			"semester":   "2026-2027-Fall",
			"course_ids": []string{uuid.NewString(), uuid.NewString()},
		}
		code, _ := validateBinding[CreateEnrollmentRequest](t, body)
		assert.Equal(t, 200, code)
	})

	t.Run("missing semester rejected", func(t *testing.T) {
		body := map[string]any{
			"course_ids": []string{uuid.NewString()},
		}
		code, _ := validateBinding[CreateEnrollmentRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("empty course_ids rejected", func(t *testing.T) {
		body := map[string]any{
			"semester":   "2026-2027-Fall",
			"course_ids": []string{},
		}
		code, _ := validateBinding[CreateEnrollmentRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("missing course_ids rejected", func(t *testing.T) {
		body := map[string]any{
			"semester": "2026-2027-Fall",
		}
		code, _ := validateBinding[CreateEnrollmentRequest](t, body)
		assert.Equal(t, 400, code)
	})
}

func TestApproveEnrollmentRequest_RequiresProgramID(t *testing.T) {
	code, _ := validateBinding[ApproveEnrollmentRequest](t, map[string]any{})
	assert.Equal(t, 400, code)

	code, _ = validateBinding[ApproveEnrollmentRequest](t,
		map[string]any{"program_id": uuid.NewString()})
	assert.Equal(t, 200, code)
}

func TestRejectEnrollmentRequest_RequiresReason(t *testing.T) {
	code, _ := validateBinding[RejectEnrollmentRequest](t,
		map[string]any{"program_id": uuid.NewString()})
	assert.Equal(t, 400, code, "rejection_reason required")

	code, _ = validateBinding[RejectEnrollmentRequest](t,
		map[string]any{
			"program_id":       uuid.NewString(),
			"rejection_reason": "Schedule conflicts with mandatory course",
		})
	assert.Equal(t, 200, code)
}

func TestEnrollmentProgramResponse_OmitsBlankFields(t *testing.T) {
	r := EnrollmentProgramResponse{
		ID: uuid.New(), StudentID: uuid.New(),
		Semester: "2026-Fall", Status: "pending",
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	assert.NotContains(t, str, "student_number")
	assert.NotContains(t, str, "student_name")
	assert.NotContains(t, str, "department")
}

func TestLatestRejectionResponse_NullableLatestRejection(t *testing.T) {
	// nil pointer must serialize as null, not omitted (required by frontend)
	r := LatestRejectionResponse{StudentID: uuid.New(), Semester: "x"}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"latest_rejection":null`)
}
