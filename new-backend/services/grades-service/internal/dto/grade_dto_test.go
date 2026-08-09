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

func TestSubmitScoreRequest_RequiresIDsAndSlug(t *testing.T) {
	t.Run("missing registration_id rejected", func(t *testing.T) {
		code, _ := validateBinding[SubmitScoreRequest](t, map[string]any{"slug": "midterm"})
		assert.Equal(t, 400, code)
	})

	t.Run("missing slug rejected", func(t *testing.T) {
		code, _ := validateBinding[SubmitScoreRequest](t,
			map[string]any{"registration_id": uuid.NewString()})
		assert.Equal(t, 400, code)
	})

	t.Run("score nil + is_absent true is valid", func(t *testing.T) {
		code, _ := validateBinding[SubmitScoreRequest](t, map[string]any{
			"registration_id": uuid.NewString(),
			"slug":            "midterm",
			"is_absent":       true,
		})
		assert.Equal(t, 200, code)
	})

	t.Run("score provided is valid", func(t *testing.T) {
		code, _ := validateBinding[SubmitScoreRequest](t, map[string]any{
			"registration_id": uuid.NewString(),
			"slug":            "midterm",
			"score":           75.5,
		})
		assert.Equal(t, 200, code)
	})
}

func TestBulkSubmitScoresRequest_RequiresSlugAndDive(t *testing.T) {
	t.Run("missing slug rejected", func(t *testing.T) {
		code, _ := validateBinding[BulkSubmitScoresRequest](t, map[string]any{
			"scores": []map[string]any{{"registration_id": uuid.NewString()}},
		})
		assert.Equal(t, 400, code)
	})

	t.Run("scores missing required field rejected", func(t *testing.T) {
		code, _ := validateBinding[BulkSubmitScoresRequest](t, map[string]any{
			"slug":   "midterm",
			"scores": []map[string]any{{}}, // missing registration_id
		})
		assert.Equal(t, 400, code)
	})

	t.Run("happy path", func(t *testing.T) {
		code, _ := validateBinding[BulkSubmitScoresRequest](t, map[string]any{
			"slug": "final",
			"scores": []map[string]any{
				{"registration_id": uuid.NewString(), "score": 80.0},
				{"registration_id": uuid.NewString(), "is_absent": true},
			},
		})
		assert.Equal(t, 200, code)
	})
}

func TestAppealScoreRequest_Validation(t *testing.T) {
	valid := map[string]any{
		"student_id": uuid.NewString(),
		"course_id":  uuid.NewString(),
		"slug":       "final",
		"new_score":  85.0,
		"reason":     "Clerical error in question 3 grading",
	}

	t.Run("happy path", func(t *testing.T) {
		code, _ := validateBinding[AppealScoreRequest](t, valid)
		assert.Equal(t, 200, code)
	})

	t.Run("score out of range rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["new_score"] = 150.0
		code, _ := validateBinding[AppealScoreRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("reason too short rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["reason"] = "short"
		code, _ := validateBinding[AppealScoreRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("missing course_id rejected", func(t *testing.T) {
		body := copyMap(valid)
		delete(body, "course_id")
		code, _ := validateBinding[AppealScoreRequest](t, body)
		assert.Equal(t, 400, code)
	})
}

func TestScoreLockRequest_RequiresBothFields(t *testing.T) {
	code, _ := validateBinding[ScoreLockRequest](t, map[string]any{})
	assert.Equal(t, 400, code)

	code, _ = validateBinding[ScoreLockRequest](t,
		map[string]any{"registration_id": uuid.NewString(), "slug": "midterm"})
	assert.Equal(t, 200, code)
}

func TestSubmitScoreResponse_OmitsFinalizeWhenAbsent(t *testing.T) {
	r := SubmitScoreResponse{
		ID: uuid.New(), StudentNumber: "20240001",
		Slug: "midterm", AutoFinalized: false,
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	assert.NotContains(t, str, "finalize_result")
	assert.NotContains(t, str, "auto_finalized")
}

func TestClassStatistics_OmitsAttendanceFailedWhenZero(t *testing.T) {
	s := ClassStatistics{Mean: 75, StdDev: 10}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "attendance_failed_count")
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
