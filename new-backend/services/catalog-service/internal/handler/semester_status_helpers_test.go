package handler

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/catalog/internal/service"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidSemesterName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid fall semester", "2025-2026-Fall", true},
		{"valid spring semester", "2025-2026-Spring", true},
		{"valid boundary year lower", "2000-2001-Fall", true},
		{"valid boundary year upper", "2099-2100-Spring", true},

		{"non-consecutive years gap", "2025-2027-Fall", false},
		{"same year", "2025-2025-Fall", false},
		{"reverse years", "2026-2025-Fall", false},
		{"invalid term summer", "2025-2026-Summer", false},
		{"lowercase term fall", "2025-2026-fall", false},
		{"missing term", "2025-2026", false},
		{"hyphen instead of slash", "2025/2026-Fall", false},
		{"empty string", "", false},
		{"gibberish", "random_semester_name", false},
		{"year below 2000", "1999-2000-Fall", false},
		{"year above 2100", "2101-2102-Fall", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidSemesterName(tt.input))
		})
	}
}

func TestPeriodTargets(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	periods := &semesterPeriods{
		Catalog:    &periodTimeRange{Start: start, End: end},
		Enrollment: &periodTimeRange{Start: start, End: end},
		Grading:    &periodTimeRange{Start: start, End: end},
		Attendance: &periodTimeRange{Start: start, End: end},
	}

	targets := periodTargets(periods)
	require.Len(t, targets, 4)

	assert.Equal(t, sharedRepo.PeriodTypeCatalog, targets[0].periodType)
	assert.Equal(t, sharedRepo.PeriodTypeEnrollment, targets[1].periodType)
	assert.Equal(t, sharedRepo.PeriodTypeGrading, targets[2].periodType)
	assert.Equal(t, sharedRepo.PeriodTypeAttendance, targets[3].periodType)

	for _, trg := range targets {
		assert.NotNil(t, trg.pr)
		assert.Equal(t, start, trg.pr.Start)
		assert.Equal(t, end, trg.pr.End)
	}
}

func TestAcceptPeriodTargets(t *testing.T) {
	now := time.Now().UTC()
	hardDeadline := now.Add(30 * 24 * time.Hour)

	t.Run("drops nil targets", func(t *testing.T) {
		targets := []periodTarget{
			{periodType: sharedRepo.PeriodTypeCatalog, pr: nil},
			{periodType: sharedRepo.PeriodTypeEnrollment, pr: &periodTimeRange{
				Start: now,
				End:   now.Add(10 * 24 * time.Hour),
			}},
		}

		accepted, errs := acceptPeriodTargets(targets, hardDeadline)
		require.Empty(t, errs)
		require.Len(t, accepted, 1)
		assert.Equal(t, sharedRepo.PeriodTypeEnrollment, accepted[0].periodType)
	})

	t.Run("accepts targets within hard deadline", func(t *testing.T) {
		targets := []periodTarget{
			{periodType: sharedRepo.PeriodTypeCatalog, pr: &periodTimeRange{
				Start: now,
				End:   hardDeadline,
			}},
			{periodType: sharedRepo.PeriodTypeEnrollment, pr: &periodTimeRange{
				Start: now,
				End:   hardDeadline.Add(-1 * time.Hour),
			}},
		}

		accepted, errs := acceptPeriodTargets(targets, hardDeadline)
		require.Empty(t, errs)
		require.Len(t, accepted, 2)
	})

	t.Run("rejects targets exceeding hard deadline", func(t *testing.T) {
		targets := []periodTarget{
			{periodType: sharedRepo.PeriodTypeCatalog, pr: &periodTimeRange{
				Start: now,
				End:   now.Add(5 * 24 * time.Hour),
			}},
			{periodType: sharedRepo.PeriodTypeEnrollment, pr: &periodTimeRange{
				Start: now,
				End:   hardDeadline.Add(1 * time.Hour),
			}},
			{periodType: sharedRepo.PeriodTypeGrading, pr: &periodTimeRange{
				Start: now,
				End:   hardDeadline.Add(24 * time.Hour),
			}},
		}

		accepted, errs := acceptPeriodTargets(targets, hardDeadline)
		require.Len(t, accepted, 1)
		assert.Equal(t, sharedRepo.PeriodTypeCatalog, accepted[0].periodType)

		require.Len(t, errs, 2)
		assert.Contains(t, errs[0], "enrollment: period end exceeds hard_deadline")
		assert.Contains(t, errs[1], "grading: period end exceeds hard_deadline")
	})
}

func TestToClosedDays(t *testing.T) {
	entries := []closedDayEntry{
		{Date: "2026-10-29", Reason: "Cumhuriyet Bayrami"},
		{Date: "2026-01-01", Reason: "Yilbasi"},
	}

	converted := toClosedDays(entries)
	require.Len(t, converted, 2)

	assert.Equal(t, service.ClosedDay{Date: "2026-10-29", Reason: "Cumhuriyet Bayrami"}, converted[0])
	assert.Equal(t, service.ClosedDay{Date: "2026-01-01", Reason: "Yilbasi"}, converted[1])

	assert.Empty(t, toClosedDays(nil))
	assert.Empty(t, toClosedDays([]closedDayEntry{}))
}
