package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/stretchr/testify/assert"
)

func TestEnrollmentAppErrors(t *testing.T) {
	assert.Equal(t, "ENROLLMENT_PERIOD_ENDED", ErrEnrollmentPeriodEnded.Code)
	assert.Equal(t, http.StatusForbidden, ErrEnrollmentPeriodEnded.HTTPStatus)

	assert.Equal(t, "ENROLLMENT_PERIOD_NOT_OPEN", ErrEnrollmentPeriodNotOpen.Code)
	assert.Equal(t, http.StatusForbidden, ErrEnrollmentPeriodNotOpen.HTTPStatus)

	// Distinct codes
	assert.NotEqual(t, ErrEnrollmentPeriodEnded.Code, ErrEnrollmentPeriodNotOpen.Code)
}

func TestEnrollmentSentinelErrors_DistinctMessages(t *testing.T) {
	all := []error{
		ErrStudentDeactivated, ErrStudentNotFound,
		ErrAlreadySubmitted, ErrProgramNotFound, ErrInvalidStatus, ErrCannotModifyApproved,
		ErrCourseFull, ErrCourseNotFound, ErrInvalidDepartment, ErrInvalidClassLevel,
		ErrPrerequisitesNotMet, ErrPrerequisiteNotFound,
		ErrScheduleConflict,
		ErrInvalidSemester, ErrNoCourses, ErrInvalidCourseID,
		ErrTooManyCourses, ErrDuplicateCourse,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.NotEmpty(t, e.Error(), "error message must not be empty")
		assert.False(t, seen[e.Error()], "duplicate error message: %v", e)
		seen[e.Error()] = true
	}
}

func TestSentinels_AreNotAppErrors(t *testing.T) {
	// Distinguishing sentinel errors from AppError matters for the
	// service-layer error handling — the service wraps sentinels via
	// sharedErrors.Wrap or maps them to specific AppErrors.
	_, ok := sharedErrors.As(ErrAlreadySubmitted)
	assert.False(t, ok, "ErrAlreadySubmitted should be a plain sentinel, not AppError")
}

func TestMaxCoursesPerEnrollment_SaneLimit(t *testing.T) {
	assert.Greater(t, MaxCoursesPerEnrollment, 0)
	assert.LessOrEqual(t, MaxCoursesPerEnrollment, 20,
		"limit should not be insanely high; raise it intentionally")
}
