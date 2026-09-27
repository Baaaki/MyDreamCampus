package service

import (
	"testing"

	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollmentService_CheckScheduleConflict(t *testing.T) {
	svc := &EnrollmentService{}

	t.Run("empty courses list", func(t *testing.T) {
		err := svc.checkScheduleConflict(nil)
		assert.NoError(t, err)
	})

	t.Run("no schedule sessions in courses", func(t *testing.T) {
		courses := []contracts.SemesterCourseResponse{
			{CourseCode: "CS101"},
			{CourseCode: "MATH101"},
		}
		err := svc.checkScheduleConflict(courses)
		assert.NoError(t, err)
	})

	t.Run("disjoint days and slots", func(t *testing.T) {
		courses := []contracts.SemesterCourseResponse{
			{
				CourseCode: "CS101",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Monday",
						SlotNumbers: []int16{1, 2, 3},
					},
				},
			},
			{
				CourseCode: "MATH101",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Tuesday",
						SlotNumbers: []int16{1, 2, 3},
					},
				},
			},
		}
		err := svc.checkScheduleConflict(courses)
		assert.NoError(t, err)
	})

	t.Run("same day but different non-overlapping slots", func(t *testing.T) {
		courses := []contracts.SemesterCourseResponse{
			{
				CourseCode: "CS101",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Monday",
						SlotNumbers: []int16{1, 2},
					},
				},
			},
			{
				CourseCode: "CS102",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Monday",
						SlotNumbers: []int16{3, 4},
					},
				},
			},
		}
		err := svc.checkScheduleConflict(courses)
		assert.NoError(t, err)
	})

	t.Run("conflict on same day and same slot between two courses", func(t *testing.T) {
		courses := []contracts.SemesterCourseResponse{
			{
				CourseCode: "CS101",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Wednesday",
						SlotNumbers: []int16{2, 3},
					},
				},
			},
			{
				CourseCode: "PHYS101",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Wednesday",
						SlotNumbers: []int16{3, 4},
					},
				},
			},
		}
		err := svc.checkScheduleConflict(courses)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "schedule conflict on Wednesday slot 3")
	})

	t.Run("conflict within same course theory and lab session overlap", func(t *testing.T) {
		courses := []contracts.SemesterCourseResponse{
			{
				CourseCode: "CS101",
				ScheduleSessions: []contracts.ScheduleSession{
					{
						DayOfWeek:   "Friday",
						SessionType: "theory",
						SlotNumbers: []int16{1, 2},
					},
					{
						DayOfWeek:   "Friday",
						SessionType: "lab",
						SlotNumbers: []int16{2, 3}, // Overlaps on slot 2
					},
				},
			},
		}
		err := svc.checkScheduleConflict(courses)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "schedule conflict on Friday slot 2")
	})
}
