package service

import (
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/google/uuid"
)

// Wire-contract payload builders for student outbox events. Every key below is
// consumed by at least one downstream service (auth, attendance, enrollment,
// grades). Renaming or dropping a key silently breaks consumers — JSON marshal
// is too forgiving to catch it. event_payloads_test.go pins these contracts.

// buildStudentCreatedPayload assembles the outbox payload for student.created.
// `id` is nil at this point — CreateStudentWithEvent overwrites it inside the
// same transaction with the generated row id.
func buildStudentCreatedPayload(req dto.CreateStudentRequest, advisorID *uuid.UUID) map[string]any {
	payload := map[string]any{
		"id":              nil,
		"student_number":  req.StudentNumber,
		"first_name":      req.FirstName,
		"last_name":       req.LastName,
		"email":           req.Email,
		"faculty":         req.Faculty,
		"department":      req.Department,
		"enrollment_year": req.EnrollmentYear,
		"class_level":     req.ClassLevel,
		"status":          "active",
	}
	if advisorID != nil {
		payload["advisor_id"] = advisorID.String()
	}
	return payload
}

// StudentUpdatedInputs groups arguments for buildStudentUpdatedPayload — the
// number of independent fields warrants a struct.
type StudentUpdatedInputs struct {
	ID              string
	Current         db.Student
	FirstName       string
	LastName        string
	Email           string
	ClassLevel      int16
	ChangedFields   map[string]any
	StatusOverride  *string
	AdvisorOverride *uuid.UUID
}

// buildStudentUpdatedPayload assembles the outbox payload for student.updated.
// `changed_fields` is a sparse projection used by auth (selective sync); the
// flat top-level keys are the fallback consumed by attendance/grades.
func buildStudentUpdatedPayload(in StudentUpdatedInputs) map[string]any {
	payload := map[string]any{
		"id":              in.ID,
		"student_number":  in.Current.StudentNumber,
		"first_name":      in.FirstName,
		"last_name":       in.LastName,
		"email":           in.Email,
		"faculty":         in.Current.Faculty,
		"department":      in.Current.Department,
		"enrollment_year": int(in.Current.EnrollmentYear),
		"class_level":     in.ClassLevel,
		"advisor_id":      utils.PgtypeToUUIDString(in.Current.AdvisorID),
		"status":          utils.PgTextToString(in.Current.Status),
		"changed_fields":  in.ChangedFields,
	}
	if in.StatusOverride != nil {
		payload["status"] = *in.StatusOverride
	}
	if in.AdvisorOverride != nil {
		payload["advisor_id"] = in.AdvisorOverride.String()
	}
	return payload
}

// buildStudentDeactivatedPayload assembles the outbox payload for
// student.deactivated. `deleted_at` is RFC3339 so consumers can use
// time.Parse(time.RFC3339) directly.
func buildStudentDeactivatedPayload(id, studentNumber string) map[string]any {
	return map[string]any{
		"id":             id,
		"student_number": studentNumber,
		"is_active":      false,
		"deleted_at":     clock.Now().Format(time.RFC3339),
	}
}
