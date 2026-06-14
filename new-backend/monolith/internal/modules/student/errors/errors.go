package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
)

// ============================================================================
// STUDENT SERVICE SPECIFIC ERRORS
// These errors are specific to student management domain
// They should NOT be moved to shared/errors as they represent business logic
// unique to the student service
// ============================================================================

var (
	// Student resource errors
	ErrStudentNotFound     = sharedErrors.New("STUDENT_NOT_FOUND", "Student not found", http.StatusNotFound)
	ErrStudentNumberExists = sharedErrors.New("STUDENT_NUMBER_EXISTS", "Student number already exists", http.StatusConflict)
	ErrStudentEmailExists  = sharedErrors.New("STUDENT_EMAIL_EXISTS", "Email already exists", http.StatusConflict)

	// Advisor-related errors
	ErrAdvisorNotFound = sharedErrors.New("ADVISOR_NOT_FOUND", "Advisor not found", http.StatusNotFound)

	// Import/bulk operation errors
	ErrInvalidCSVFormat = sharedErrors.New("INVALID_CSV_FORMAT", "Invalid CSV format", http.StatusBadRequest)

	// External service errors
	ErrStaffServiceUnavailable = sharedErrors.New("STAFF_SERVICE_UNAVAILABLE", "Staff service is unavailable", http.StatusServiceUnavailable)

	// Future enrollment-related errors (for when enrollment features are implemented)
	ErrStudentAlreadyEnrolled = sharedErrors.New("ALREADY_ENROLLED", "Student is already enrolled in this course", http.StatusConflict)
	ErrStudentGPALow          = sharedErrors.New("GPA_TOO_LOW", "Student GPA does not meet course requirements", http.StatusBadRequest)
	ErrEnrollmentCapacity     = sharedErrors.New("ENROLLMENT_FULL", "Course has reached maximum enrollment capacity", http.StatusConflict)

	// Repository-specific sentinel errors (for internal use)
	ErrStudentNotFoundRepo     = sharedErrors.ErrNotFoundRepo
	ErrStudentNumberExistsRepo = sharedErrors.ErrAlreadyExistsRepo
	ErrStudentEmailExistsRepo  = sharedErrors.ErrAlreadyExistsRepo
)
