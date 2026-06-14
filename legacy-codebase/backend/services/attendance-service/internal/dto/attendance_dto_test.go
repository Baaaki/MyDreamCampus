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

func TestCreateSessionRequest_Validation(t *testing.T) {
	valid := map[string]any{
		"course_id":        uuid.NewString(),
		"week_number":      5,
		"duration_minutes": 30,
		"session_type":     "theory",
	}

	t.Run("happy path", func(t *testing.T) {
		code, body := validateBinding[CreateSessionRequest](t, valid)
		assert.Equal(t, 200, code, body)
	})

	t.Run("week_number too small", func(t *testing.T) {
		body := copyMap(valid)
		body["week_number"] = 0
		code, _ := validateBinding[CreateSessionRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("week_number too large", func(t *testing.T) {
		body := copyMap(valid)
		body["week_number"] = 15
		code, _ := validateBinding[CreateSessionRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("duration too short", func(t *testing.T) {
		body := copyMap(valid)
		body["duration_minutes"] = 2
		code, _ := validateBinding[CreateSessionRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("duration too long", func(t *testing.T) {
		body := copyMap(valid)
		body["duration_minutes"] = 200
		code, _ := validateBinding[CreateSessionRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("session_type must be theory or lab", func(t *testing.T) {
		body := copyMap(valid)
		body["session_type"] = "exam"
		code, _ := validateBinding[CreateSessionRequest](t, body)
		assert.Equal(t, 400, code)

		for _, sType := range []string{"theory", "lab"} {
			body["session_type"] = sType
			code, _ := validateBinding[CreateSessionRequest](t, body)
			assert.Equal(t, 200, code, "session_type %q must be accepted", sType)
		}
	})
}

func TestManualAttendanceRequest_RequiresStudentID(t *testing.T) {
	code, _ := validateBinding[ManualAttendanceRequest](t,
		map[string]any{"note": "n/a"})
	assert.Equal(t, 400, code)

	code, _ = validateBinding[ManualAttendanceRequest](t,
		map[string]any{"student_id": uuid.NewString()})
	assert.Equal(t, 200, code)
}

func TestQRPayload_RoundTrip(t *testing.T) {
	p := QRPayload{SessionID: "s-1", Signature: "sig-abc"}
	data, err := json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"sid":"s-1"`)
	assert.Contains(t, string(data), `"sig":"sig-abc"`)

	var got QRPayload
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, p, got)
}

func TestSessionListItem_OmitsAllOptionals(t *testing.T) {
	// Used in list responses where some sessions have no records yet.
	r := SessionListItem{WeekNumber: 5, SessionType: "theory"}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	for _, omit := range []string{"session_id", "session_date", "present_count", "absent_count", "is_active", "status"} {
		assert.NotContains(t, str, omit, "must omit %s when nil", omit)
	}
}

func TestSessionTypeAttendance_PassedField(t *testing.T) {
	a := SessionTypeAttendance{PresentCount: 10, AbsentCount: 2, TotalSessions: 12, MinRequired: 9, Passed: true}
	data, err := json.Marshal(a)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"passed":true`)
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
