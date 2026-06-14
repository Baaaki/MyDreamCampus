package rules

import (
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
)

// PeriodCheckResult contains the result of a period check.
type PeriodCheckResult struct {
	Allowed           bool      `json:"allowed"`
	Reason            string    `json:"reason"`
	EffectiveDeadline time.Time `json:"effective_deadline"`
}

// IsWithinPeriod checks whether the current simulated/real time
// falls within the active academic period for the given deadline.
// periodEnd is the effective deadline (already resolved by the repository layer).
func IsWithinPeriod(periodStart, periodEnd time.Time) PeriodCheckResult {
	now := clock.Now()

	if now.Before(periodStart) {
		return PeriodCheckResult{
			Allowed:           false,
			Reason:            "period has not started yet",
			EffectiveDeadline: periodEnd,
		}
	}

	if now.After(periodEnd) {
		return PeriodCheckResult{
			Allowed:           false,
			Reason:            "period has ended",
			EffectiveDeadline: periodEnd,
		}
	}

	return PeriodCheckResult{
		Allowed:           true,
		Reason:            "within active period",
		EffectiveDeadline: periodEnd,
	}
}
