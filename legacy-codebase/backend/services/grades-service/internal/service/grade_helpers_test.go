package service

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/grades-service/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidSlug(t *testing.T) {
	schema := []AssessmentSchemaItem{
		{Slug: "midterm", Name: "Midterm", Weight: 40},
		{Slug: "final", Name: "Final", Weight: 60},
	}

	t.Run("returns true for known slug", func(t *testing.T) {
		assert.True(t, isValidSlug(schema, "midterm"))
		assert.True(t, isValidSlug(schema, "final"))
	})

	t.Run("returns false for unknown slug", func(t *testing.T) {
		assert.False(t, isValidSlug(schema, "quiz"))
	})

	t.Run("is case sensitive", func(t *testing.T) {
		assert.False(t, isValidSlug(schema, "Midterm"))
	})

	t.Run("returns false for empty schema", func(t *testing.T) {
		assert.False(t, isValidSlug(nil, "midterm"))
		assert.False(t, isValidSlug([]AssessmentSchemaItem{}, "midterm"))
	})

	t.Run("returns false for empty slug", func(t *testing.T) {
		// Empty string is never a valid slug — it can't appear in a schema
		// because ValidateAssessmentSchema (catalog side) rejects empty names.
		assert.False(t, isValidSlug(schema, ""))
	})
}

func TestBuildFinalizeRequestedEventParams(t *testing.T) {
	fixedTime := time.Date(2026, time.April, 27, 12, 0, 0, 0, time.UTC)
	clock.Set(fixedTime)
	t.Cleanup(clock.Reset)

	courseID := uuid.New()
	instructorID := uuid.New()

	params, err := buildFinalizeRequestedEventParams(courseID, instructorID, finalizeTriggerInstructor)
	require.NoError(t, err)
	require.NotNil(t, params)

	t.Run("uses the canonical event type and routing key", func(t *testing.T) {
		assert.Equal(t, "grade.finalize.requested", params.EventType)
		assert.Equal(t, "grade.finalize.requested", params.RoutingKey)
	})

	t.Run("payload round-trips into the published event shape", func(t *testing.T) {
		var event dto.GradeFinalizeRequestedEvent
		require.NoError(t, json.Unmarshal(params.Payload, &event))

		assert.Equal(t, "grade.finalize.requested", event.EventType)
		assert.True(t, event.Timestamp.Equal(fixedTime), "timestamp should come from injected clock")
		assert.Equal(t, courseID, event.Data.CourseID)
		assert.Equal(t, instructorID, event.Data.InstructorID)
		assert.Equal(t, finalizeTriggerInstructor, event.Data.TriggeredBy)
	})
}

func TestBuildGradeSubmittedEventParams(t *testing.T) {
	fixedTime := time.Date(2026, time.April, 27, 9, 30, 0, 0, time.UTC)
	clock.Set(fixedTime)
	t.Cleanup(clock.Reset)

	studentID := uuid.New()

	t.Run("returns nil params when score is nil", func(t *testing.T) {
		params, err := buildGradeSubmittedEventParams(studentID, "CS101", "midterm", nil)
		assert.NoError(t, err)
		assert.Nil(t, params, "nil score is an absence-only upsert and must not publish")
	})

	t.Run("emits event when score is present", func(t *testing.T) {
		score := 87.5
		params, err := buildGradeSubmittedEventParams(studentID, "CS101", "midterm", &score)
		require.NoError(t, err)
		require.NotNil(t, params)

		assert.Equal(t, "grade.submitted", params.EventType)
		assert.Equal(t, "grade.submitted", params.RoutingKey)

		var event dto.GradeSubmittedEvent
		require.NoError(t, json.Unmarshal(params.Payload, &event))

		assert.Equal(t, "grade.submitted", event.EventType)
		assert.True(t, event.Timestamp.Equal(fixedTime))
		assert.Equal(t, studentID, event.Data.StudentID)
		assert.Equal(t, "CS101", event.Data.CourseCode)
		assert.Equal(t, "midterm", event.Data.Slug)
		assert.InDelta(t, 87.5, event.Data.Score, 0.0001)
	})
}

func TestFormatSemesterDisplay(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"2024_fall", "2024-2025 Güz"},
		{"2025_fall", "2025-2026 Güz"},
		{"2024_spring", "2024 Bahar"},
		{"2024_summer", "2024 Yaz"},

		// Unknown season passes through verbatim (no Turkish translation).
		{"2024_winter", "2024 winter"},

		// Malformed inputs are returned as-is.
		{"2024", "2024"},
		{"foo_bar", "foo_bar"},
		{"2024_fall_extra", "2024_fall_extra"},
		{"", ""},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.want, formatSemesterDisplay(c.input))
		})
	}
}

func TestGradePointToFloat(t *testing.T) {
	t.Run("parses standard grade strings", func(t *testing.T) {
		assert.InDelta(t, 4.0, gradePointToFloat("4.00"), 0.0001)
		assert.InDelta(t, 2.5, gradePointToFloat("2.5"), 0.0001)
		assert.InDelta(t, 0.0, gradePointToFloat("0"), 0.0001)
	})

	t.Run("returns 0 on parse error", func(t *testing.T) {
		assert.Equal(t, 0.0, gradePointToFloat(""))
		assert.Equal(t, 0.0, gradePointToFloat("AA"))
		assert.Equal(t, 0.0, gradePointToFloat("not-a-number"))
	})
}

func TestParseInterfaceToFloat64(t *testing.T) {
	t.Run("nil yields zero", func(t *testing.T) {
		assert.Equal(t, 0.0, parseInterfaceToFloat64(nil))
	})

	t.Run("native numeric types pass through", func(t *testing.T) {
		assert.Equal(t, 1.5, parseInterfaceToFloat64(float64(1.5)))
		assert.Equal(t, 2.5, parseInterfaceToFloat64(float32(2.5)))
		assert.Equal(t, 3.0, parseInterfaceToFloat64(int64(3)))
		assert.Equal(t, 4.0, parseInterfaceToFloat64(int32(4)))
	})

	t.Run("string is parsed", func(t *testing.T) {
		assert.InDelta(t, 87.5, parseInterfaceToFloat64("87.5"), 0.0001)
	})

	t.Run("unparseable string yields 0", func(t *testing.T) {
		assert.Equal(t, 0.0, parseInterfaceToFloat64("not-a-number"))
	})

	t.Run("pgtype.Numeric round-trips", func(t *testing.T) {
		// A pgtype.Numeric built from a decimal literal must round-trip via
		// Float64Value(). 12.5 = 125 * 10^-1.
		num := pgtype.Numeric{
			Int:   big.NewInt(125),
			Exp:   -1,
			Valid: true,
		}
		got := parseInterfaceToFloat64(num)
		assert.InDelta(t, 12.5, got, 0.0001)
	})

	t.Run("default branch falls back to fmt.Sprintf parse", func(t *testing.T) {
		// A bool is neither numeric nor string — the default arm renders it via
		// fmt.Sprintf which yields "true"/"false", which then fails to parse
		// and returns 0. Documenting the behavior so a future tightening of the
		// type switch is intentional.
		assert.Equal(t, 0.0, parseInterfaceToFloat64(true))
	})
}

func TestParseInterfaceToInt64(t *testing.T) {
	t.Run("nil yields zero", func(t *testing.T) {
		assert.Equal(t, int64(0), parseInterfaceToInt64(nil))
	})

	t.Run("native int types pass through", func(t *testing.T) {
		assert.Equal(t, int64(7), parseInterfaceToInt64(int64(7)))
		assert.Equal(t, int64(8), parseInterfaceToInt64(int32(8)))
	})

	t.Run("float64 truncates toward zero", func(t *testing.T) {
		// pgx's jsonb decoder hands us float64 for whole-number JSON
		// fields. The truncation semantics here are part of the contract
		// (we never want to round up a count of completed sessions).
		assert.Equal(t, int64(9), parseInterfaceToInt64(float64(9.7)))
	})

	t.Run("string is parsed", func(t *testing.T) {
		assert.Equal(t, int64(123), parseInterfaceToInt64("123"))
	})

	t.Run("unparseable string yields 0 — never panics", func(t *testing.T) {
		// parseInterfaceToInt64 runs on values from JSONB blobs; any panic
		// here would crash the request handler. The contract is to return
		// zero on bad input rather than fail loud.
		assert.Equal(t, int64(0), parseInterfaceToInt64("nope"))
	})

	t.Run("pgtype.Numeric truncates", func(t *testing.T) {
		// 875/100 = 8.75 → int64(8). The same Numeric goes to 8.75 in
		// parseInterfaceToFloat64; the divergence is intentional.
		num := pgtype.Numeric{
			Int:   big.NewInt(875),
			Exp:   -2,
			Valid: true,
		}
		assert.Equal(t, int64(8), parseInterfaceToInt64(num))
	})

	t.Run("default branch falls back to fmt.Sprintf parse", func(t *testing.T) {
		// Mirrors the float64 helper — the default arm renders unsupported
		// types via fmt.Sprintf and tries to parse the result. Bool prints
		// as "true"/"false" which fail parsing → 0. Locking this in so a
		// future tightening of the type switch is a deliberate choice.
		assert.Equal(t, int64(0), parseInterfaceToInt64(true))
	})
}

func TestDecodeScoresJSON(t *testing.T) {
	t.Run("nil raw returns empty (non-nil) map", func(t *testing.T) {
		out, err := decodeScoresJSON(nil)
		require.NoError(t, err)
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})

	t.Run("empty bytes returns empty map", func(t *testing.T) {
		out, err := decodeScoresJSON([]byte{})
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("'null' literal returns empty map", func(t *testing.T) {
		out, err := decodeScoresJSON([]byte("null"))
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("decodes []byte JSONB into ScoreDetail map", func(t *testing.T) {
		raw := []byte(`{"midterm":{"score":"85.00","is_absent":false,"is_locked":true}}`)
		out, err := decodeScoresJSON(raw)
		require.NoError(t, err)
		require.Contains(t, out, "midterm")

		got := out["midterm"]
		require.NotNil(t, got.Score)
		assert.InDelta(t, 85.0, *got.Score, 0.0001)
		assert.False(t, got.IsAbsent)
		assert.True(t, got.IsLocked)
	})

	t.Run("decodes string input identically to []byte", func(t *testing.T) {
		raw := `{"final":{"score":"90","is_absent":false,"is_locked":false}}`
		out, err := decodeScoresJSON(raw)
		require.NoError(t, err)
		require.Contains(t, out, "final")
		require.NotNil(t, out["final"].Score)
		assert.InDelta(t, 90.0, *out["final"].Score, 0.0001)
	})

	t.Run("decodes map[string]any (pgx default jsonb shape)", func(t *testing.T) {
		raw := map[string]any{
			"quiz1": map[string]any{
				"score":     "70",
				"is_absent": false,
				"is_locked": false,
			},
		}
		out, err := decodeScoresJSON(raw)
		require.NoError(t, err)
		require.Contains(t, out, "quiz1")
		require.NotNil(t, out["quiz1"].Score)
		assert.InDelta(t, 70.0, *out["quiz1"].Score, 0.0001)
	})

	t.Run("absence (no score field) leaves Score nil", func(t *testing.T) {
		raw := []byte(`{"midterm":{"is_absent":true,"is_locked":false}}`)
		out, err := decodeScoresJSON(raw)
		require.NoError(t, err)
		require.Contains(t, out, "midterm")

		got := out["midterm"]
		assert.Nil(t, got.Score, "missing score field must produce nil pointer")
		assert.True(t, got.IsAbsent)
	})

	t.Run("rejects unexpected raw types", func(t *testing.T) {
		_, err := decodeScoresJSON(123)
		assert.Error(t, err)
	})

	t.Run("propagates JSON parse errors", func(t *testing.T) {
		_, err := decodeScoresJSON([]byte("{not-json"))
		assert.Error(t, err)
	})

	// Sanity: ScoreDetail conforms to the dto contract used elsewhere — guard
	// against a refactor that drops a field.
	t.Run("returned values are dto.ScoreDetail", func(t *testing.T) {
		raw := []byte(`{"x":{"score":"50","is_absent":false,"is_locked":true}}`)
		out, err := decodeScoresJSON(raw)
		require.NoError(t, err)
		var _ dto.ScoreDetail = out["x"]
	})
}

