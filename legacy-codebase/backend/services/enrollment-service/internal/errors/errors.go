package errors

import (
	"errors"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
)

// Enrollment-specific AppErrors (with HTTP status for handler mapping)
var (
	ErrEnrollmentPeriodEnded   = sharedErrors.New("ENROLLMENT_PERIOD_ENDED", "enrollment period has ended for this semester", http.StatusForbidden)
	ErrEnrollmentPeriodNotOpen = sharedErrors.New("ENROLLMENT_PERIOD_NOT_OPEN", "enrollment period has not started yet", http.StatusForbidden)
)

// Enrollment-specific errors
var (
	// Student errors
	ErrStudentDeactivated = errors.New("student account is deactivated")
	ErrStudentNotFound    = errors.New("student not found in cache")

	// Enrollment errors
	ErrAlreadySubmitted      = errors.New("enrollment program already submitted for this semester")
	ErrProgramNotFound       = errors.New("enrollment program not found")
	ErrInvalidStatus         = errors.New("invalid enrollment status")
	ErrCannotModifyApproved  = errors.New("cannot modify approved enrollment program")

	// Course errors
	ErrCourseFull            = errors.New("course capacity is full")
	ErrCourseNotFound        = errors.New("course not found")
	ErrInvalidDepartment     = errors.New("course does not belong to student's department")
	ErrInvalidClassLevel     = errors.New("course class level exceeds student's class level")

	// Prerequisite errors
	ErrPrerequisitesNotMet   = errors.New("prerequisites not met for one or more courses")
	ErrPrerequisiteNotFound  = errors.New("prerequisite information not found")

	// Schedule errors
	ErrScheduleConflict      = errors.New("schedule conflict detected between courses")

	// Validation errors
	ErrInvalidSemester       = errors.New("invalid semester format")
	ErrNoCourses             = errors.New("no courses provided in enrollment request")
	ErrInvalidCourseID       = errors.New("invalid course ID")
	ErrTooManyCourses        = errors.New("too many courses selected for enrollment")
	ErrDuplicateCourse       = errors.New("duplicate course in enrollment request")
)

// Enrollment business limits
const (
	MaxCoursesPerEnrollment = 10
)
