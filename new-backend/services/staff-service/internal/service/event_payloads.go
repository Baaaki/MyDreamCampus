package service

import (
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
)

// Wire-contract payload builders for staff outbox events. The downstream
// consumer is auth-service (and indirectly catalog/student via auth's user
// projection). A silent rename here breaks login for new staff members.

// buildStaffCreatedPayload assembles the outbox payload for staff.created.
// `id` is intentionally nil — CreateStaffWithEvent overwrites it with the
// generated row id inside the same transaction.
func buildStaffCreatedPayload(req dto.CreateStaffRequest) map[string]any {
	return map[string]any{
		"id":         nil,
		"email":      req.Email,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"role":       req.Role,
		"department": req.Department,
	}
}

// StaffUpdatedInputs groups arguments for buildStaffUpdatedPayload.
type StaffUpdatedInputs struct {
	ID             string
	Department     *string
	Phone          *string
	OfficeLocation *string
}

// buildStaffUpdatedPayload assembles the outbox payload for staff.updated.
// Auth-service uses these to refresh its user projection — pointer-typed
// optionals stay nil if not provided so the consumer can distinguish
// "unchanged" from "explicitly cleared".
func buildStaffUpdatedPayload(in StaffUpdatedInputs) map[string]any {
	changedFields := map[string]any{}
	if in.Department != nil {
		changedFields["department"] = *in.Department
	}
	if in.Phone != nil {
		changedFields["phone"] = *in.Phone
	}
	if in.OfficeLocation != nil {
		changedFields["office_location"] = *in.OfficeLocation
	}
	return map[string]any{
		"id":              in.ID,
		"staff_id":        in.ID,
		"department":      in.Department,
		"phone":           in.Phone,
		"office_location": in.OfficeLocation,
		"changed_fields":  changedFields,
	}
}

// buildStaffDeactivatedPayload assembles the outbox payload for
// staff.deactivated. Carries both `id` (auth-service) and `staff_id`
// (student-service) for wire compatibility.
func buildStaffDeactivatedPayload(id string) map[string]any {
	return map[string]any{
		"id":       id,
		"staff_id": id,
	}
}
