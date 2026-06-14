package rules

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/stretchr/testify/assert"
)

func TestCanEditGrade_HardDeadlineBlocksEveryone(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	hard := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	cases := []bool{false, true} // both regular and admin
	for _, isAdmin := range cases {
		res := CanEditGrade(GradeEditParams{
			GlobalDeadline: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			HardDeadline:   &hard,
			IsAdminAction:  isAdmin,
		})
		assert.False(t, res.Allowed, "hard deadline must block admin=%v too", isAdmin)
		assert.Contains(t, res.Reason, "hard deadline")
	}
}

func TestCanEditGrade_AdminBypassesScoreLockAndPeriod(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	res := CanEditGrade(GradeEditParams{
		IsLocked:       true,
		GlobalDeadline: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), // long past
		IsAdminAction:  true,
	})
	assert.True(t, res.Allowed, "admin must bypass score lock + period")
	assert.Contains(t, res.Reason, "admin")
}

func TestCanEditGrade_LockedScoreRejectsNonAdmin(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	res := CanEditGrade(GradeEditParams{
		IsLocked:       true,
		GlobalDeadline: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	assert.False(t, res.Allowed)
	assert.Contains(t, res.Reason, "locked")
}

func TestCanEditGrade_PeriodEndedRejectsTeacher(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	res := CanEditGrade(GradeEditParams{
		GlobalDeadline: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), // ended
	})
	assert.False(t, res.Allowed)
	assert.Contains(t, res.Reason, "period")
}

func TestCanEditGrade_OverrideExtendsDeadline(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	override := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	res := CanEditGrade(GradeEditParams{
		GlobalDeadline:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), // ended
		OverrideDeadline: &override,                                    // extends
	})
	assert.True(t, res.Allowed, "override should extend deadline beyond global")
	assert.Equal(t, override, res.EffectiveDeadline)
}

func TestCanEditGrade_OverrideEarlierThanGlobalIgnored(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	override := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	res := CanEditGrade(GradeEditParams{
		GlobalDeadline:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		OverrideDeadline: &override,
	})
	assert.True(t, res.Allowed)
	assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), res.EffectiveDeadline,
		"earlier override must not shorten deadline")
}

func TestCanEditGrade_HappyPath(t *testing.T) {
	clock.Set(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	defer clock.Reset()

	res := CanEditGrade(GradeEditParams{
		GlobalDeadline: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	assert.True(t, res.Allowed)
	assert.Contains(t, res.Reason, "within")
}
