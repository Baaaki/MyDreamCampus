package rules

import (
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
)

// GradeEditParams contains the inputs for the CanEditGrade decision.
type GradeEditParams struct {
	IsLocked         bool       // Current lock status of the score
	GlobalDeadline   time.Time  // Global grading period end date
	OverrideDeadline *time.Time // Course-specific override (nullable)
	IsAdminAction    bool       // Whether the caller is an admin
	HardDeadline     *time.Time // Semester hard deadline — absolute lock (nil = no hard deadline)
}

// GradeEditResult contains the decision and reasoning.
type GradeEditResult struct {
	Allowed           bool      `json:"allowed"`
	Reason            string    `json:"reason"`
	EffectiveDeadline time.Time `json:"effective_deadline"`
}

// CanEditGrade determines whether a grade can be modified based on the 4-layer lock model.
//
// IMPORTANT: hard_deadline check MUST come before admin bypass.
// Previous behavior allowed admin to bypass everything -- this was a security gap
// where admin could modify grades even after semester completion.
// See: docs/semester-wizard-plan.md "3 Katmanli Enforcement"
//
// Layer 0: Semester hard_deadline — absolute lock for everyone (including admin)
// Layer 1: Score-level lock (is_locked) — only admin can bypass
// Layer 2: Time-based deadline — effective deadline = max(global, override)
// Layer 3: Admin override — admins bypass score lock + period checks (but NOT hard_deadline)
func CanEditGrade(params GradeEditParams) GradeEditResult {
	// Calculate effective deadline
	effectiveDeadline := params.GlobalDeadline
	if params.OverrideDeadline != nil && params.OverrideDeadline.After(params.GlobalDeadline) {
		effectiveDeadline = *params.OverrideDeadline
	}

	now := clock.Now()

	// Layer 0: Semester hard_deadline — absolute lock, nobody can bypass
	if params.HardDeadline != nil && now.After(*params.HardDeadline) {
		return GradeEditResult{
			Allowed:           false,
			Reason:            "semester hard deadline has passed — no modifications allowed",
			EffectiveDeadline: effectiveDeadline,
		}
	}

	// Layer 3 (early): Admins bypass score lock + period checks
	if params.IsAdminAction {
		return GradeEditResult{
			Allowed:           true,
			Reason:            "admin action — score lock and period checks bypassed",
			EffectiveDeadline: effectiveDeadline,
		}
	}

	// Layer 1: Score-level lock
	if params.IsLocked {
		return GradeEditResult{
			Allowed:           false,
			Reason:            "score is locked — admin must unlock it first",
			EffectiveDeadline: effectiveDeadline,
		}
	}

	// Layer 2: Time-based deadline
	if now.After(effectiveDeadline) {
		return GradeEditResult{
			Allowed:           false,
			Reason:            "grading period has ended",
			EffectiveDeadline: effectiveDeadline,
		}
	}

	return GradeEditResult{
		Allowed:           true,
		Reason:            "within grading period and score is unlocked",
		EffectiveDeadline: effectiveDeadline,
	}
}
