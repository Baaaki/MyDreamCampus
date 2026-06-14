package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestScheduleSession_RequiresValidSessionType(t *testing.T) {
	t.Run("valid theory", func(t *testing.T) {
		body := map[string]any{
			"day_of_week":  "Monday",
			"slot_numbers": []int{1, 2},
			"session_type": "theory",
		}
		code, _ := validateBinding[ScheduleSession](t, body)
		assert.Equal(t, 200, code)
	})

	t.Run("valid lab", func(t *testing.T) {
		body := map[string]any{
			"day_of_week":  "Friday",
			"slot_numbers": []int{6, 7, 8},
			"session_type": "lab",
		}
		code, _ := validateBinding[ScheduleSession](t, body)
		assert.Equal(t, 200, code)
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		body := map[string]any{
			"day_of_week":  "Monday",
			"slot_numbers": []int{1},
			"session_type": "exam",
		}
		code, _ := validateBinding[ScheduleSession](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("missing session_type rejected", func(t *testing.T) {
		body := map[string]any{
			"day_of_week":  "Monday",
			"slot_numbers": []int{1},
		}
		code, _ := validateBinding[ScheduleSession](t, body)
		assert.Equal(t, 400, code)
	})
}

func TestCreateSemesterCourseRequest_Validation(t *testing.T) {
	valid := map[string]any{
		"course_code":         "CSE101",
		"class_level":         1,
		"instructor_id":       uuid.New().String(),
		"instructor_fullname": "Dr. Ada Lovelace",
		"classroom_location":  "Z-101",
		"max_capacity":        50,
		"assessment_schema": []map[string]any{
			{"slug": "midterm", "name": "Midterm", "weight": 40},
			{"slug": "final", "name": "Final", "weight": 60},
		},
		"schedule_sessions": []map[string]any{
			{"day_of_week": "Monday", "slot_numbers": []int{1, 2}, "session_type": "theory"},
		},
	}

	t.Run("happy path", func(t *testing.T) {
		code, body := validateBinding[CreateSemesterCourseRequest](t, valid)
		assert.Equal(t, 200, code, body)
	})

	t.Run("empty schedule_sessions rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["schedule_sessions"] = []map[string]any{}
		code, _ := validateBinding[CreateSemesterCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("empty assessment_schema rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["assessment_schema"] = []map[string]any{}
		code, _ := validateBinding[CreateSemesterCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("invalid instructor_id (not uuid) rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["instructor_id"] = "not-a-uuid"
		code, _ := validateBinding[CreateSemesterCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("max_capacity zero rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["max_capacity"] = 0
		code, _ := validateBinding[CreateSemesterCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("session_type validates inside dive", func(t *testing.T) {
		body := copyMap(valid)
		body["schedule_sessions"] = []map[string]any{
			{"day_of_week": "Monday", "slot_numbers": []int{1}, "session_type": "exam"},
		}
		code, _ := validateBinding[CreateSemesterCourseRequest](t, body)
		assert.Equal(t, 400, code)
	})
}
