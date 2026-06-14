package rules

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/stretchr/testify/assert"
)

func tp(t time.Time) *time.Time { return &t }

func TestCanOperateInSemester_HardDeadlineBlocksAll(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	hard := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	for _, admin := range []bool{false, true} {
		res := CanOperateInSemester(SemesterOperationParams{
			HardDeadline:  hard,
			IsAdminAction: admin,
		})
		assert.False(t, res.Allowed)
		assert.Equal(t, "semester_ended", res.Reason)
	}
}

func TestCanOperateInSemester_AdminBypassWithinHardDeadline(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	res := CanOperateInSemester(SemesterOperationParams{
		HardDeadline:  time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		IsAdminAction: true,
	})
	assert.True(t, res.Allowed)
	assert.Equal(t, "admin_bypass", res.Reason)
}

func TestCanOperateInSemester_NoPeriodAllowed(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	res := CanOperateInSemester(SemesterOperationParams{
		HardDeadline: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	assert.True(t, res.Allowed)
	assert.Equal(t, "no_period_defined", res.Reason)
}

func TestCanOperateInSemester_PeriodChecks(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	defer clock.Reset()

	hard := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("before period start", func(t *testing.T) {
		res := CanOperateInSemester(SemesterOperationParams{
			HardDeadline: hard,
			PeriodStart:  tp(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:    tp(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)),
		})
		assert.False(t, res.Allowed)
		assert.Equal(t, "period_not_started", res.Reason)
	})

	t.Run("after period end", func(t *testing.T) {
		res := CanOperateInSemester(SemesterOperationParams{
			HardDeadline: hard,
			PeriodStart:  tp(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:    tp(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)),
		})
		assert.False(t, res.Allowed)
		assert.Equal(t, "period_ended", res.Reason)
	})

	t.Run("within period", func(t *testing.T) {
		res := CanOperateInSemester(SemesterOperationParams{
			HardDeadline: hard,
			PeriodStart:  tp(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:    tp(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		})
		assert.True(t, res.Allowed)
		assert.Equal(t, "within_period", res.Reason)
	})
}

func TestCanEnrollInSemester_StrictPeriodLock(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	t.Run("nil period start = closed", func(t *testing.T) {
		res := CanEnrollInSemester(EnrollmentParams{})
		assert.False(t, res.Allowed)
		assert.Equal(t, "enrollment_not_configured", res.Reason)
	})

	t.Run("not started yet", func(t *testing.T) {
		res := CanEnrollInSemester(EnrollmentParams{
			PeriodStart: tp(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:   tp(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)),
		})
		assert.False(t, res.Allowed)
		assert.Equal(t, "enrollment_not_started", res.Reason)
	})

	t.Run("ended", func(t *testing.T) {
		res := CanEnrollInSemester(EnrollmentParams{
			PeriodStart: tp(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:   tp(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)),
		})
		assert.False(t, res.Allowed)
		assert.Equal(t, "enrollment_ended", res.Reason)
	})

	t.Run("within enrollment window", func(t *testing.T) {
		res := CanEnrollInSemester(EnrollmentParams{
			PeriodStart: tp(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:   tp(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		})
		assert.True(t, res.Allowed)
		assert.Equal(t, "within_enrollment_period", res.Reason)
	})
}
