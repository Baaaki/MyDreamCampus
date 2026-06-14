package service

import (
	"testing"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/dto"
	catalogErrors "github.com/baaaki/mydreamcampus/course-catalog-service/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidateAssessmentSchema(t *testing.T) {
	t.Run("rejects empty schema", func(t *testing.T) {
		err := ValidateAssessmentSchema(nil)
		assert.ErrorIs(t, err, catalogErrors.ErrInvalidAssessmentSchema)
	})

	t.Run("accepts a single 100-weight item", func(t *testing.T) {
		err := ValidateAssessmentSchema([]dto.AssessmentItem{
			{Slug: "final", Name: "Final", Weight: 100},
		})
		assert.NoError(t, err)
	})

	t.Run("accepts multiple items summing to 100", func(t *testing.T) {
		err := ValidateAssessmentSchema([]dto.AssessmentItem{
			{Slug: "midterm", Name: "Midterm", Weight: 40},
			{Slug: "final", Name: "Final", Weight: 60},
		})
		assert.NoError(t, err)
	})

	t.Run("rejects when total weight is not 100", func(t *testing.T) {
		err := ValidateAssessmentSchema([]dto.AssessmentItem{
			{Slug: "midterm", Name: "Midterm", Weight: 40},
			{Slug: "final", Name: "Final", Weight: 50},
		})
		assert.ErrorIs(t, err, catalogErrors.ErrAssessmentWeightNotHundred)
	})

	t.Run("rejects duplicate slugs", func(t *testing.T) {
		err := ValidateAssessmentSchema([]dto.AssessmentItem{
			{Slug: "exam", Name: "Midterm", Weight: 50},
			{Slug: "exam", Name: "Final", Weight: 50},
		})
		assert.ErrorIs(t, err, catalogErrors.ErrDuplicateAssessmentSlug)
	})

	t.Run("rejects invalid slug formats", func(t *testing.T) {
		invalid := []string{
			"Final",       // uppercase letter
			"1exam",       // starts with digit
			"_quiz",       // starts with underscore
			"midterm-1",   // hyphen not allowed
			"final exam",  // space not allowed
			"",            // empty
		}
		for _, slug := range invalid {
			err := ValidateAssessmentSchema([]dto.AssessmentItem{
				{Slug: slug, Name: "Item", Weight: 100},
			})
			assert.ErrorIs(t, err, catalogErrors.ErrInvalidAssessmentSchema, "slug=%q", slug)
		}
	})

	t.Run("accepts slugs with letters, digits and underscores", func(t *testing.T) {
		err := ValidateAssessmentSchema([]dto.AssessmentItem{
			{Slug: "midterm_1", Name: "M1", Weight: 30},
			{Slug: "quiz2", Name: "Q2", Weight: 20},
			{Slug: "f", Name: "F", Weight: 50},
		})
		assert.NoError(t, err)
	})

	t.Run("rejects empty or overlong name", func(t *testing.T) {
		long := make([]byte, 101)
		for i := range long {
			long[i] = 'a'
		}
		cases := []string{"", string(long)}
		for _, name := range cases {
			err := ValidateAssessmentSchema([]dto.AssessmentItem{
				{Slug: "x", Name: name, Weight: 100},
			})
			assert.ErrorIs(t, err, catalogErrors.ErrInvalidAssessmentSchema, "len(name)=%d", len(name))
		}
	})

	t.Run("rejects negative or above-100 weight", func(t *testing.T) {
		cases := []int16{-1, 101, 200}
		for _, w := range cases {
			err := ValidateAssessmentSchema([]dto.AssessmentItem{
				{Slug: "x", Name: "Item", Weight: w},
			})
			assert.ErrorIs(t, err, catalogErrors.ErrInvalidAssessmentSchema, "weight=%d", w)
		}
	})

	t.Run("accepts a 100-char name on the boundary", func(t *testing.T) {
		name := make([]byte, 100)
		for i := range name {
			name[i] = 'a'
		}
		err := ValidateAssessmentSchema([]dto.AssessmentItem{
			{Slug: "x", Name: string(name), Weight: 100},
		})
		assert.NoError(t, err)
	})
}

func TestValidateSlotNumbers(t *testing.T) {
	t.Run("accepts the full valid range 1..9", func(t *testing.T) {
		err := ValidateSlotNumbers([]int16{1, 2, 3, 4, 5, 6, 7, 8, 9})
		assert.NoError(t, err)
	})

	t.Run("accepts an empty slice", func(t *testing.T) {
		// The function is purely a per-element guard — no minimum-length contract.
		// Documenting the current behavior so a future tightening is intentional.
		err := ValidateSlotNumbers(nil)
		assert.NoError(t, err)
	})

	t.Run("rejects 0 (below range)", func(t *testing.T) {
		err := ValidateSlotNumbers([]int16{0})
		assert.ErrorIs(t, err, catalogErrors.ErrInvalidSlotNumber)
	})

	t.Run("rejects 10 (above range)", func(t *testing.T) {
		err := ValidateSlotNumbers([]int16{10})
		assert.ErrorIs(t, err, catalogErrors.ErrInvalidSlotNumber)
	})

	t.Run("rejects negative slots", func(t *testing.T) {
		err := ValidateSlotNumbers([]int16{-1})
		assert.ErrorIs(t, err, catalogErrors.ErrInvalidSlotNumber)
	})

	t.Run("rejects when only one element is out of range", func(t *testing.T) {
		err := ValidateSlotNumbers([]int16{1, 2, 99, 4})
		assert.ErrorIs(t, err, catalogErrors.ErrInvalidSlotNumber)
	})
}

func TestValidateDayOfWeek(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"accepts monday", "monday", false},
		{"accepts tuesday", "tuesday", false},
		{"accepts wednesday", "wednesday", false},
		{"accepts thursday", "thursday", false},
		{"accepts friday", "friday", false},
		{"accepts saturday", "saturday", false},
		{"accepts sunday", "sunday", false},

		{"rejects capitalised day (case sensitive)", "Monday", true},
		{"rejects empty string", "", true},
		{"rejects non-day word", "yesterday", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateDayOfWeek(c.input)
			if c.wantErr {
				assert.ErrorIs(t, err, catalogErrors.ErrInvalidDayOfWeek)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateScheduleSessionTypes(t *testing.T) {
	t.Run("accepts theory-only when slot count matches theoreticalHours", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2, 3}, SessionType: "theory"},
		}
		assert.NoError(t, validateScheduleSessionTypes(sessions, 3, 0))
	})

	t.Run("accepts theory + lab when both totals match", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2}, SessionType: "theory"},
			{DayOfWeek: "tuesday", SlotNumbers: []int16{4, 5}, SessionType: "lab"},
		}
		assert.NoError(t, validateScheduleSessionTypes(sessions, 2, 2))
	})

	t.Run("rejects unknown session type", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "studio"},
		}
		err := validateScheduleSessionTypes(sessions, 1, 0)
		assert.ErrorIs(t, err, catalogErrors.ErrInvalidSessionType)
	})

	t.Run("rejects theory session when course has 0 theory hours", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "theory"},
		}
		err := validateScheduleSessionTypes(sessions, 0, 1)
		assert.ErrorIs(t, err, catalogErrors.ErrTheoryHoursZero)
	})

	t.Run("rejects lab session when course has 0 lab hours", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "lab"},
		}
		err := validateScheduleSessionTypes(sessions, 1, 0)
		assert.ErrorIs(t, err, catalogErrors.ErrLabHoursZero)
	})

	t.Run("rejects when theory slot total does not match", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2}, SessionType: "theory"},
		}
		err := validateScheduleSessionTypes(sessions, 3, 0) // expected 3 theory slots, got 2
		assert.ErrorIs(t, err, catalogErrors.ErrTheorySlotCountMismatch)
	})

	t.Run("rejects when lab slot total does not match", func(t *testing.T) {
		sessions := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "lab"},
		}
		err := validateScheduleSessionTypes(sessions, 0, 2) // expected 2 lab slots, got 1
		assert.ErrorIs(t, err, catalogErrors.ErrLabSlotCountMismatch)
	})

	t.Run("accepts no sessions when both hours are zero", func(t *testing.T) {
		// Edge case worth pinning: an all-zero course with no sessions is currently
		// accepted. Caller is expected to reject 0-hour courses upstream
		// (ErrCourseCreditsZero), but this helper itself is tolerant.
		assert.NoError(t, validateScheduleSessionTypes(nil, 0, 0))
	})
}

func TestIsValidSemesterFormat(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"2025-2026-Fall", true},
		{"2025-2026-Spring", true},
		{"2099-2100-Fall", true},
		{"2000-2001-Spring", true},

		// Year math: end must be exactly start+1.
		{"2025-2027-Fall", false},
		{"2025-2025-Fall", false},
		{"2026-2025-Fall", false},

		// Range bounds (start < 2000 or start > 2100).
		{"1999-2000-Fall", false},
		{"2101-2102-Fall", false},

		// Format violations.
		{"2025-2026-Summer", false},
		{"2025-2026-fall", false}, // case sensitive
		{"2025-2026", false},
		{"25-26-Fall", false},
		{"", false},
		{"random-junk", false},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.want, isValidSemesterFormat(c.input))
		})
	}
}
