package service

import (
	"encoding/json"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	"github.com/baaaki/mydreamcampus/catalog/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertMarshalsEmptyArray pins the wire shape clients rely on: a course
// without sessions sends [], never null.
func assertMarshalsEmptyArray(t *testing.T, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(raw))
}

func row(day, sessionType string, slot int16) db.GetScheduleSessionsByCourseIDRow {
	return db.GetScheduleSessionsByCourseIDRow{
		DayOfWeek:   db.DayOfWeekEnum(day),
		SlotNumber:  slot,
		SessionType: db.ScheduleSessionTypeEnum(sessionType),
	}
}

func multiRow(day, sessionType string, slot int16) db.GetScheduleSessionsByMultipleCourseIDsRow {
	return db.GetScheduleSessionsByMultipleCourseIDsRow{
		DayOfWeek:   db.DayOfWeekEnum(day),
		SlotNumber:  slot,
		SessionType: db.ScheduleSessionTypeEnum(sessionType),
	}
}

func TestGroupScheduleSessionsFromRows(t *testing.T) {
	t.Run("nil input marshals as empty array", func(t *testing.T) {
		assertMarshalsEmptyArray(t, groupScheduleSessionsFromRows(nil))
	})

	t.Run("single row produces one session with one slot", func(t *testing.T) {
		got := groupScheduleSessionsFromRows([]db.GetScheduleSessionsByCourseIDRow{
			row("monday", "theory", 1),
		})
		want := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "theory"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("rows with same day and type collapse into one session", func(t *testing.T) {
		got := groupScheduleSessionsFromRows([]db.GetScheduleSessionsByCourseIDRow{
			row("monday", "theory", 1),
			row("monday", "theory", 2),
			row("monday", "theory", 3),
		})
		want := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2, 3}, SessionType: "theory"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("same day with different session types stays separate", func(t *testing.T) {
		got := groupScheduleSessionsFromRows([]db.GetScheduleSessionsByCourseIDRow{
			row("monday", "theory", 1),
			row("monday", "theory", 2),
			row("monday", "lab", 3),
			row("monday", "lab", 4),
		})
		want := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2}, SessionType: "theory"},
			{DayOfWeek: "monday", SlotNumbers: []int16{3, 4}, SessionType: "lab"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("different days stay separate", func(t *testing.T) {
		got := groupScheduleSessionsFromRows([]db.GetScheduleSessionsByCourseIDRow{
			row("monday", "theory", 1),
			row("wednesday", "theory", 2),
			row("friday", "theory", 5),
		})
		assert.Len(t, got, 3)
		assert.Equal(t,
			[]dto.ScheduleSession{
				{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "theory"},
				{DayOfWeek: "wednesday", SlotNumbers: []int16{2}, SessionType: "theory"},
				{DayOfWeek: "friday", SlotNumbers: []int16{5}, SessionType: "theory"},
			},
			got,
		)
	})

	t.Run("week order, theory before lab, slots ascending, on every call", func(t *testing.T) {
		rows := []db.GetScheduleSessionsByCourseIDRow{
			row("sunday", "theory", 2),
			row("friday", "lab", 6),
			row("monday", "lab", 4),
			row("friday", "lab", 5),
			row("monday", "theory", 2),
			row("thursday", "theory", 1),
			row("monday", "theory", 1),
			row("tuesday", "theory", 3),
		}
		want := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2}, SessionType: "theory"},
			{DayOfWeek: "monday", SlotNumbers: []int16{4}, SessionType: "lab"},
			{DayOfWeek: "tuesday", SlotNumbers: []int16{3}, SessionType: "theory"},
			{DayOfWeek: "thursday", SlotNumbers: []int16{1}, SessionType: "theory"},
			{DayOfWeek: "friday", SlotNumbers: []int16{5, 6}, SessionType: "lab"},
			{DayOfWeek: "sunday", SlotNumbers: []int16{2}, SessionType: "theory"},
		}
		// The groups come out of a map; one lucky run proves nothing.
		for range 20 {
			shuffled := slices.Clone(rows)
			rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
			require.Equal(t, want, groupScheduleSessionsFromRows(shuffled))
		}
	})

	t.Run("preserves duplicate slot numbers (no dedup)", func(t *testing.T) {
		// Documenting the contract: the grouper does not dedupe inside a key.
		// The DB unique index is what guarantees this in practice; if the index
		// is ever loosened, the function will faithfully echo duplicates.
		got := groupScheduleSessionsFromRows([]db.GetScheduleSessionsByCourseIDRow{
			row("monday", "theory", 1),
			row("monday", "theory", 1),
		})
		assert.Equal(t,
			[]dto.ScheduleSession{
				{DayOfWeek: "monday", SlotNumbers: []int16{1, 1}, SessionType: "theory"},
			},
			got,
		)
	})
}

func TestGroupScheduleSessionsFromMultiRows(t *testing.T) {
	// Same contract as the by-course-id variant — the row type only differs
	// in the SQL it came from, but the grouping shape is identical. The cases
	// below mirror the by-id ones to lock both call sites.

	t.Run("nil input marshals as empty array", func(t *testing.T) {
		assertMarshalsEmptyArray(t, groupScheduleSessionsFromMultiRows(nil))
	})

	t.Run("groups by (day, type) across rows from multiple courses", func(t *testing.T) {
		got := groupScheduleSessionsFromMultiRows([]db.GetScheduleSessionsByMultipleCourseIDsRow{
			multiRow("monday", "theory", 1),
			multiRow("monday", "theory", 2),
			multiRow("tuesday", "lab", 4),
		})
		want := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1, 2}, SessionType: "theory"},
			{DayOfWeek: "tuesday", SlotNumbers: []int16{4}, SessionType: "lab"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("same day, different types stay separate", func(t *testing.T) {
		got := groupScheduleSessionsFromMultiRows([]db.GetScheduleSessionsByMultipleCourseIDsRow{
			multiRow("friday", "theory", 1),
			multiRow("friday", "lab", 2),
		})
		assert.Len(t, got, 2)
	})

	t.Run("sorted like the by-id variant, on every call", func(t *testing.T) {
		rows := []db.GetScheduleSessionsByMultipleCourseIDsRow{
			multiRow("wednesday", "theory", 4),
			multiRow("monday", "lab", 7),
			multiRow("wednesday", "theory", 3),
			multiRow("monday", "theory", 1),
		}
		want := []dto.ScheduleSession{
			{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "theory"},
			{DayOfWeek: "monday", SlotNumbers: []int16{7}, SessionType: "lab"},
			{DayOfWeek: "wednesday", SlotNumbers: []int16{3, 4}, SessionType: "theory"},
		}
		for range 20 {
			shuffled := slices.Clone(rows)
			rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
			require.Equal(t, want, groupScheduleSessionsFromMultiRows(shuffled))
		}
	})
}

func TestTeacherSchedule(t *testing.T) {
	t.Run("no sessions marshals as empty array", func(t *testing.T) {
		assertMarshalsEmptyArray(t, teacherSchedule(nil, "A-101"))
	})

	t.Run("one entry per slot in the course's room", func(t *testing.T) {
		got := teacherSchedule([]db.GetScheduleSessionsByMultipleCourseIDsRow{
			multiRow("monday", "theory", 1),
			multiRow("monday", "lab", 2),
		}, "A-101")
		assert.Equal(t, []dto.TeacherScheduleSession{
			{Day: "monday", Time: slotTimeRange(1), Room: "A-101", SessionType: "theory"},
			{Day: "monday", Time: slotTimeRange(2), Room: "A-101", SessionType: "lab"},
		}, got)
	})
}

func TestTeacherCourseItems(t *testing.T) {
	t.Run("no courses marshals as empty array", func(t *testing.T) {
		raw, err := json.Marshal(teacherCourseItems(nil, nil))
		require.NoError(t, err)
		assert.JSONEq(t, `[]`, string(raw))
	})

	t.Run("each course gets only its own sessions", func(t *testing.T) {
		withLessons, without := uuid.New(), uuid.New()
		session := multiRow("tuesday", "theory", 3)
		session.SemesterCourseID = utils.UUIDToPgtype(withLessons)

		got := teacherCourseItems([]db.GetTeacherCoursesRow{
			{ID: utils.UUIDToPgtype(withLessons), CourseCode: "BLM101", ClassroomLocation: "A-101"},
			{ID: utils.UUIDToPgtype(without), CourseCode: "BLM102", ClassroomLocation: "B-202"},
		}, []db.GetScheduleSessionsByMultipleCourseIDsRow{session})

		require.Len(t, got, 2)
		assert.Equal(t, []dto.TeacherScheduleSession{
			{Day: "tuesday", Time: slotTimeRange(3), Room: "A-101", SessionType: "theory"},
		}, got[0].Schedule)
		assertMarshalsEmptyArray(t, got[1].Schedule)
	})
}
