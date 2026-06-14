package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/stretchr/testify/assert"
)

func TestGradesErrors_HTTPStatuses(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"INVALID_SCORE":          {ErrInvalidScore, http.StatusBadRequest},
		"INVALID_SLUG":           {ErrInvalidSlug, http.StatusBadRequest},
		"ALREADY_FINALIZED":      {ErrAlreadyFinalized, http.StatusConflict},
		"SCORE_EXISTS":           {ErrScoreExists, http.StatusConflict},
		"ATTENDANCE_FAILED":      {ErrAttendanceFailed, http.StatusBadRequest},
		"SCORE_LOCKED":           {ErrScoreLocked, http.StatusForbidden},
		"INCOMPLETE_ASSESSMENT":  {ErrIncompleteAssessment, http.StatusBadRequest},
		"GRADING_PERIOD_ENDED":   {ErrGradingPeriodEnded, http.StatusForbidden},
		"NO_PERIOD_DEFINED":      {ErrNoPeriodDefined, http.StatusBadRequest},
		"NOT_COURSE_INSTRUCTOR":  {ErrNotCourseInstructor, http.StatusForbidden},
		"STUDENT_DEACTIVATED":    {ErrStudentDeactivated, http.StatusForbidden},
		"COURSE_NOT_FOUND":       {ErrCourseNotFound, http.StatusNotFound},
		"REGISTRATION_NOT_FOUND": {ErrRegistrationNotFound, http.StatusNotFound},
		"SCORE_NOT_FOUND":        {ErrScoreNotFound, http.StatusNotFound},
		"STUDENT_NOT_FOUND":      {ErrStudentNotFound, http.StatusNotFound},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code)
		assert.Equal(t, c.want, c.err.HTTPStatus, "status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestGradesErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrInvalidScore, ErrInvalidSlug, ErrAlreadyFinalized, ErrScoreExists,
		ErrAttendanceFailed, ErrScoreLocked, ErrIncompleteAssessment,
		ErrGradingPeriodEnded, ErrNoPeriodDefined,
		ErrNotCourseInstructor, ErrStudentDeactivated,
		ErrCourseNotFound, ErrRegistrationNotFound, ErrScoreNotFound, ErrStudentNotFound,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate code %q", e.Code)
		seen[e.Code] = true
	}
}
