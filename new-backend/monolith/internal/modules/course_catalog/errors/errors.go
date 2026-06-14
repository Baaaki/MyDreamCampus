package errors

import (
	"fmt"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
)

// ScheduleConflictError wraps the sentinel error and carries conflict details
type ScheduleConflictError struct {
	AppErr     *sharedErrors.AppError
	CourseCode string
	Department string
	DayOfWeek  string
	SlotNumber int16
}

func (e *ScheduleConflictError) Error() string {
	return fmt.Sprintf("[%s] %s", e.AppErr.Code, e.AppErr.Message)
}

func (e *ScheduleConflictError) Unwrap() error {
	return e.AppErr
}

func NewScheduleConflictError(courseCode, department, dayOfWeek string, slotNumber int16) *ScheduleConflictError {
	return &ScheduleConflictError{
		AppErr:     ErrInstructorScheduleConflict,
		CourseCode: courseCode,
		Department: department,
		DayOfWeek:  dayOfWeek,
		SlotNumber: slotNumber,
	}
}

// ============================================================================
// COURSE CATALOG SERVICE SPECIFIC ERRORS
// These errors are specific to course catalog management domain
// ============================================================================

var (
	// Catalog errors (AppError for HTTP responses)
	ErrCourseNotFound            = sharedErrors.New("COURSE_NOT_FOUND", "Course not found", http.StatusNotFound)
	ErrCourseCodeExists          = sharedErrors.New("COURSE_CODE_EXISTS", "Course code already exists", http.StatusConflict)
	ErrInvalidPrerequisite       = sharedErrors.New("INVALID_PREREQUISITE", "Invalid prerequisite course", http.StatusBadRequest)
	ErrInvalidPrerequisiteLevel  = sharedErrors.New("INVALID_PREREQUISITE_LEVEL", "Prerequisite class level must be less than course class level", http.StatusBadRequest)
	ErrInvalidStatus             = sharedErrors.New("INVALID_STATUS", "Invalid course status", http.StatusBadRequest)
	ErrCourseNotActive           = sharedErrors.New("COURSE_NOT_ACTIVE", "Course is not active", http.StatusBadRequest)

	// Semester course errors (AppError for HTTP responses)
	ErrInvalidSemesterFormat     = sharedErrors.New("INVALID_SEMESTER_FORMAT", "Invalid semester format. Expected: YYYY-YYYY-Fall or YYYY-YYYY-Spring", http.StatusBadRequest)
	ErrSemesterCourseNotFound    = sharedErrors.New("SEMESTER_COURSE_NOT_FOUND", "Semester course not found", http.StatusNotFound)
	ErrCourseAlreadyOpened       = sharedErrors.New("COURSE_ALREADY_OPENED", "Course already opened for this semester", http.StatusConflict)
	ErrClassLevelMismatch        = sharedErrors.New("CLASS_LEVEL_MISMATCH", "Class level mismatch between request and catalog", http.StatusBadRequest)
	ErrSemesterAlreadyExists     = sharedErrors.New("SEMESTER_ALREADY_EXISTS", "Target semester already has courses", http.StatusConflict)
	ErrSourceSemesterNotFound    = sharedErrors.New("SOURCE_SEMESTER_NOT_FOUND", "Source semester not found", http.StatusNotFound)
	ErrCourseHasEnrollments      = sharedErrors.New("COURSE_HAS_ENROLLMENTS", "Course has enrollments", http.StatusConflict)

	// Instructor errors (AppError for HTTP responses)
	ErrInstructorNotFound        = sharedErrors.New("INSTRUCTOR_NOT_FOUND", "Instructor not found", http.StatusNotFound)
	ErrInstructorNotActive       = sharedErrors.New("INSTRUCTOR_NOT_ACTIVE", "Instructor is not active", http.StatusBadRequest)
	ErrInstructorNotInDepartment = sharedErrors.New("INSTRUCTOR_NOT_IN_DEPARTMENT", "Instructor not in department", http.StatusBadRequest)
	ErrInstructorScheduleConflict = sharedErrors.New("INSTRUCTOR_SCHEDULE_CONFLICT", "Instructor has schedule conflict", http.StatusConflict)

	// Schedule errors (AppError for HTTP responses)
	ErrInvalidSlotNumber         = sharedErrors.New("INVALID_SLOT_NUMBER", "Invalid slot number (must be 1-9)", http.StatusBadRequest)
	ErrInvalidDayOfWeek          = sharedErrors.New("INVALID_DAY_OF_WEEK", "Invalid day of week", http.StatusBadRequest)
	ErrInvalidSessionType        = sharedErrors.New("INVALID_SESSION_TYPE", "Invalid session type (must be 'theory' or 'lab')", http.StatusBadRequest)
	ErrTheoryHoursZero           = sharedErrors.New("THEORY_HOURS_ZERO", "Cannot create theory schedule: course has 0 theoretical hours in catalog", http.StatusBadRequest)
	ErrLabHoursZero              = sharedErrors.New("LAB_HOURS_ZERO", "Cannot create lab schedule: course has 0 lab hours in catalog", http.StatusBadRequest)
	ErrTheorySlotCountMismatch   = sharedErrors.New("THEORY_SLOT_COUNT_MISMATCH", "Theory slot count must match catalog theoretical_hours", http.StatusBadRequest)
	ErrLabSlotCountMismatch      = sharedErrors.New("LAB_SLOT_COUNT_MISMATCH", "Lab slot count must match catalog lab_hours", http.StatusBadRequest)
	ErrCourseCreditsZero         = sharedErrors.New("COURSE_CREDITS_ZERO", "Course has 0 credits in catalog, cannot open semester course", http.StatusBadRequest)

	// Assessment errors (AppError for HTTP responses)
	ErrInvalidAssessmentSchema   = sharedErrors.New("INVALID_ASSESSMENT_SCHEMA", "Invalid assessment schema", http.StatusBadRequest)
	ErrAssessmentWeightNotHundred = sharedErrors.New("ASSESSMENT_WEIGHT_NOT_HUNDRED", "Assessment weights must sum to 100", http.StatusBadRequest)
	ErrDuplicateAssessmentSlug   = sharedErrors.New("DUPLICATE_ASSESSMENT_SLUG", "Duplicate assessment slug", http.StatusBadRequest)

	// Semester status errors (AppError for HTTP responses)
	ErrSemesterNotActive = sharedErrors.New("SEMESTER_NOT_ACTIVE", "semester is not active — modifications are not allowed", http.StatusForbidden)
	// IMPORTANT: "semester_courses" (courses offered this semester) vs "course_catalog" (all courses ever defined).
	// semester_courses: FROZEN once semester is activated. No add/remove/modify — not even admin.
	// course_catalog: can be modified anytime, independent of semesters.
	// See: docs/semester-wizard-plan.md "Iki Katmanli Degismezlik Modeli"
	ErrSemesterCourseFrozen = sharedErrors.New("SEMESTER_COURSE_FROZEN", "semester course offerings are frozen after activation — no modifications allowed", http.StatusForbidden)
	ErrSemesterNotPlanned   = sharedErrors.New("SEMESTER_NOT_PLANNED", "semester courses can only be modified while semester is in 'planned' status", http.StatusForbidden)

	// Deadline errors (AppError for HTTP responses)
	ErrCourseCreationPeriodEnded   = sharedErrors.New("COURSE_CREATION_PERIOD_ENDED", "course creation period has ended for this semester", http.StatusForbidden)
	ErrCourseCreationPeriodNotOpen = sharedErrors.New("COURSE_CREATION_PERIOD_NOT_OPEN", "course creation period has not started yet", http.StatusForbidden)

	// Transaction errors (AppError for HTTP responses)
	ErrTransactionFailed = sharedErrors.New("TRANSACTION_FAILED", "Failed to start database transaction", http.StatusInternalServerError)

	// Repository-specific sentinel errors (for internal use)
	ErrCourseNotFoundRepo         = sharedErrors.ErrNotFoundRepo
	ErrCourseExistsRepo           = sharedErrors.ErrAlreadyExistsRepo
	ErrSemesterCourseNotFoundRepo = sharedErrors.ErrNotFoundRepo
	ErrCourseAlreadyOpenedRepo    = sharedErrors.ErrAlreadyExistsRepo
)
