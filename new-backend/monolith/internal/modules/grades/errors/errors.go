package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
)

// ============================================================================
// GRADES SERVICE SPECIFIC ERRORS
// These errors are specific to grade management and assessment scoring
// They should NOT be moved to shared/errors as they represent business logic
// unique to the grades service
// ============================================================================

var (
	// Grade submission errors (AppError for HTTP responses)
	ErrInvalidScore     = sharedErrors.New("INVALID_SCORE", "score must be between 0 and 100", http.StatusBadRequest)
	ErrInvalidSlug      = sharedErrors.New("INVALID_SLUG", "assessment slug not found in schema", http.StatusBadRequest)
	ErrAlreadyFinalized = sharedErrors.New("ALREADY_FINALIZED", "course already finalized, cannot modify scores", http.StatusConflict)
	ErrScoreExists      = sharedErrors.New("SCORE_EXISTS", "score already exists for this assessment", http.StatusConflict)
	ErrAttendanceFailed = sharedErrors.New("ATTENDANCE_FAILED", "student failed due to attendance, cannot enter score manually", http.StatusBadRequest)
	ErrScoreLocked      = sharedErrors.New("SCORE_LOCKED", "score is locked and cannot be modified", http.StatusForbidden)
	ErrIncompleteAssessment = sharedErrors.New("INCOMPLETE_ASSESSMENT", "cannot lock assessment: not every student has a score entered", http.StatusBadRequest)
	ErrGradingPeriodEnded = sharedErrors.New("GRADING_PERIOD_ENDED", "grading period has ended", http.StatusForbidden)
	ErrNoPeriodDefined    = sharedErrors.New("NO_PERIOD_DEFINED", "no active grading period defined for this semester", http.StatusBadRequest)

	// Authorization errors (AppError for HTTP responses)
	ErrNotCourseInstructor = sharedErrors.New("NOT_COURSE_INSTRUCTOR", "you are not the instructor of this course", http.StatusForbidden)
	ErrStudentDeactivated  = sharedErrors.New("STUDENT_DEACTIVATED", "student is deactivated", http.StatusForbidden)

	// Not found errors (AppError for HTTP responses)
	ErrCourseNotFound       = sharedErrors.New("COURSE_NOT_FOUND", "course not found", http.StatusNotFound)
	ErrRegistrationNotFound = sharedErrors.New("REGISTRATION_NOT_FOUND", "registration not found", http.StatusNotFound)
	ErrScoreNotFound        = sharedErrors.New("SCORE_NOT_FOUND", "score not found", http.StatusNotFound)
	ErrStudentNotFound      = sharedErrors.New("STUDENT_NOT_FOUND", "student not found", http.StatusNotFound)

	// Repository-specific sentinel errors (for internal use)
	ErrCourseNotFoundRepo       = sharedErrors.ErrNotFoundRepo
	ErrRegistrationNotFoundRepo = sharedErrors.ErrNotFoundRepo
	ErrScoreNotFoundRepo        = sharedErrors.ErrNotFoundRepo
	ErrStudentNotFoundRepo      = sharedErrors.ErrNotFoundRepo
)
