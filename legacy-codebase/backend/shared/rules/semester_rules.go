package rules

import (
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
)

// SemesterOperationParams contains the inputs for CanOperateInSemester.
type SemesterOperationParams struct {
	HardDeadline  time.Time  // Absolute semester deadline — after this, nobody can modify data
	PeriodStart   *time.Time // Service-specific period start (nil = no period defined)
	PeriodEnd     *time.Time // Service-specific period end (nil = no period defined)
	IsAdminAction bool       // Whether the caller is an admin
}

// SemesterOperationResult contains the decision and reasoning.
type SemesterOperationResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

// Three-layer semester enforcement for GRADES and ATTENDANCE only:
// Layer 1: hard_deadline passed -> REJECT for everyone (including admin)
// Layer 2: caller is admin -> ALLOW (admin can fix data until hard_deadline)
// Layer 3: within service period -> ALLOW/REJECT for teacher/student
//
// Why this order matters:
// - hard_deadline is the absolute lock. After it, even admin cannot modify data.
//   This ensures completed semester data integrity for auditing/accreditation.
// - Admin bypass before period check allows manual corrections (e.g. wrong grade,
//   missed attendance) without waiting for a new period window.
//
// NOTE: Enrollment uses a DIFFERENT model -- strict period lock, no admin bypass.
// See CanEnrollInSemester() for enrollment-specific rules.
func CanOperateInSemester(params SemesterOperationParams) SemesterOperationResult {
	now := clock.Now()

	// Layer 1: hard_deadline is the absolute lock
	if now.After(params.HardDeadline) {
		return SemesterOperationResult{
			Allowed: false,
			Reason:  "semester_ended",
		}
	}

	// Layer 2: admin can operate anytime before hard_deadline
	if params.IsAdminAction {
		return SemesterOperationResult{
			Allowed: true,
			Reason:  "admin_bypass",
		}
	}

	// Layer 3: if no period defined, allow by default
	if params.PeriodStart == nil {
		return SemesterOperationResult{
			Allowed: true,
			Reason:  "no_period_defined",
		}
	}

	// Layer 3: check period window
	if now.Before(*params.PeriodStart) {
		return SemesterOperationResult{
			Allowed: false,
			Reason:  "period_not_started",
		}
	}

	if now.After(*params.PeriodEnd) {
		return SemesterOperationResult{
			Allowed: false,
			Reason:  "period_ended",
		}
	}

	return SemesterOperationResult{
		Allowed: true,
		Reason:  "within_period",
	}
}

// EnrollmentParams contains the inputs for CanEnrollInSemester.
type EnrollmentParams struct {
	PeriodStart *time.Time // Enrollment period start (nil = enrollment closed)
	PeriodEnd   *time.Time // Enrollment period end (nil = enrollment closed)
}

// Enrollment uses STRICT period lock -- different from grades/attendance.
// Period inside: only students can enroll. Period outside: NOBODY can modify (admin included).
// No hard_deadline check needed -- period is the only lock.
// Why no admin override? Enrollment is the student's own responsibility.
// Admin should not add/remove courses on behalf of students.
// See: docs/semester-wizard-plan.md "Ders Kayit (Enrollment) Icin: Siki Period Kilidi"
func CanEnrollInSemester(params EnrollmentParams) SemesterOperationResult {
	now := clock.Now()

	// No period defined -> enrollment is closed
	if params.PeriodStart == nil {
		return SemesterOperationResult{
			Allowed: false,
			Reason:  "enrollment_not_configured",
		}
	}

	if now.Before(*params.PeriodStart) {
		return SemesterOperationResult{
			Allowed: false,
			Reason:  "enrollment_not_started",
		}
	}

	if now.After(*params.PeriodEnd) {
		return SemesterOperationResult{
			Allowed: false,
			Reason:  "enrollment_ended",
		}
	}

	return SemesterOperationResult{
		Allowed: true,
		Reason:  "within_enrollment_period",
	}
}
