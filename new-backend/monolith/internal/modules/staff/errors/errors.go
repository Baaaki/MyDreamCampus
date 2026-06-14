package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
)

// ============================================================================
// STAFF SERVICE SPECIFIC ERRORS
// These errors are specific to staff management domain
// They should NOT be moved to shared/errors as they represent business logic
// unique to the staff service
// ============================================================================

var (
	// Staff resource errors
	ErrStaffNotFound = sharedErrors.New("STAFF_NOT_FOUND", "Staff not found", http.StatusNotFound)
	ErrStaffExists   = sharedErrors.New("STAFF_EXISTS", "Staff already exists", http.StatusConflict)

	// Staff business logic errors
	ErrEmailExists       = sharedErrors.New("EMAIL_EXISTS", "Email already exists", http.StatusConflict)
	ErrCannotCreateAdmin = sharedErrors.New("CANNOT_CREATE_ADMIN", "Admin cannot be created via API", http.StatusBadRequest)
	ErrInvalidRole       = sharedErrors.New("INVALID_ROLE", "Invalid role specified", http.StatusBadRequest)

	// Advisor-specific business errors (for future use when advisor features are implemented)
	ErrAdvisorNotQualified       = sharedErrors.New("ADVISOR_NOT_QUALIFIED", "Staff member is not qualified to be an advisor", http.StatusBadRequest)
	ErrAdvisorHasTooManyStudents = sharedErrors.New("ADVISOR_OVERLOADED", "Advisor has reached maximum student capacity", http.StatusConflict)

	// Teacher profile errors
	ErrTeacherProfileNotFound = sharedErrors.New("TEACHER_PROFILE_NOT_FOUND", "Teacher profile not found", http.StatusNotFound)
	ErrNotATeacher            = sharedErrors.New("NOT_A_TEACHER", "Staff member is not a teacher", http.StatusBadRequest)

	// Repository-specific sentinel errors (for internal use)
	ErrStaffNotFoundRepo          = sharedErrors.ErrNotFoundRepo
	ErrStaffExistsRepo            = sharedErrors.ErrAlreadyExistsRepo
	ErrTeacherProfileNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
