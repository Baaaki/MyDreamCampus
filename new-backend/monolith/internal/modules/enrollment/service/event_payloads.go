package service

import (
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/google/uuid"
)

// Wire-contract payload builders for enrollment.* outbox events. Each event
// has its own consumer surface (auth, attendance, grades) and a silent rename
// here is a production incident — JSON marshal won't catch it. event_payloads_test.go
// pins these contracts.

// EnrollmentApprovedInputs groups arguments for buildEnrollmentApprovedPayload.
type EnrollmentApprovedInputs struct {
	ProgramID  uuid.UUID
	StudentID  uuid.UUID
	Semester   string
	CourseIDs  []uuid.UUID
	ApprovedBy uuid.UUID
}

// buildEnrollmentApprovedPayload assembles the WRAPPED outbox payload for
// enrollment.program.approved. Wrapping is the contract for this event —
// consumers (grades, attendance) read .data.* fields directly.
func buildEnrollmentApprovedPayload(in EnrollmentApprovedInputs) map[string]any {
	now := clock.Now()
	return map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "enrollment.program.approved",
		"timestamp":  now.Format(time.RFC3339),
		"data": map[string]any{
			"program_id":  in.ProgramID.String(),
			"student_id":  in.StudentID.String(),
			"semester":    in.Semester,
			"course_ids":  in.CourseIDs,
			"approved_by": in.ApprovedBy.String(),
			"approved_at": now,
		},
	}
}

// EnrollmentRejectedInputs groups arguments for buildEnrollmentRejectedPayload.
type EnrollmentRejectedInputs struct {
	ProgramID       uuid.UUID
	StudentID       uuid.UUID
	Semester        string
	CourseIDs       []uuid.UUID
	RejectedBy      uuid.UUID
	RejectionReason string
}

// buildEnrollmentRejectedPayload assembles the FLAT outbox payload for
// enrollment.program.rejected. The repository's outbox writer wraps it later
// with event_id/event_type/timestamp.
func buildEnrollmentRejectedPayload(in EnrollmentRejectedInputs) map[string]any {
	return map[string]any{
		"program_id":       in.ProgramID.String(),
		"student_id":       in.StudentID.String(),
		"semester":         in.Semester,
		"course_ids":       in.CourseIDs,
		"rejected_by":      in.RejectedBy.String(),
		"rejection_reason": in.RejectionReason,
		"rejected_at":      clock.Now(),
	}
}

// EnrollmentCancelledInputs groups arguments for buildEnrollmentCancelledPayload.
type EnrollmentCancelledInputs struct {
	ProgramID   uuid.UUID
	StudentID   uuid.UUID
	Semester    string
	CourseIDs   []uuid.UUID
	CancelledBy string // "student" | "advisor" | "admin"
	CancelType  string // "manual" | "auto_replace"
}

// buildEnrollmentCancelledPayload assembles the FLAT outbox payload for
// enrollment.program.cancelled. Used by both manual cancel and auto-replace
// flows (see CancelMyEnrollment and CreateEnrollmentProgram).
func buildEnrollmentCancelledPayload(in EnrollmentCancelledInputs) map[string]any {
	return map[string]any{
		"program_id":   in.ProgramID.String(),
		"student_id":   in.StudentID.String(),
		"semester":     in.Semester,
		"course_ids":   in.CourseIDs,
		"cancelled_by": in.CancelledBy,
		"cancel_type":  in.CancelType,
		"cancelled_at": clock.Now(),
	}
}

// EnrollmentSubmittedInputs groups arguments for buildEnrollmentSubmittedPayload.
type EnrollmentSubmittedInputs struct {
	StudentID uuid.UUID
	Semester  string
	CourseIDs []uuid.UUID
}

// buildEnrollmentSubmittedPayload assembles the FLAT outbox payload for
// enrollment.program.submitted. `program_id` is nil here — the repository
// fills it after insert inside the same transaction.
func buildEnrollmentSubmittedPayload(in EnrollmentSubmittedInputs) map[string]any {
	return map[string]any{
		"program_id":    nil,
		"student_id":    in.StudentID.String(),
		"semester":      in.Semester,
		"course_ids":    in.CourseIDs,
		"total_courses": len(in.CourseIDs),
		"submitted_at":  clock.Now(),
	}
}
