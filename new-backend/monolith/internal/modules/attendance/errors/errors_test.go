package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/stretchr/testify/assert"
)

func TestAttendanceErrors_HTTPStatuses(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"INVALID_QR_CODE":         {ErrInvalidQRCode, http.StatusBadRequest},
		"SESSION_EXPIRED":         {ErrSessionExpired, http.StatusBadRequest},
		"SESSION_NOT_ACTIVE":      {ErrSessionNotActive, http.StatusBadRequest},
		"SESSION_NOT_FOUND":       {ErrSessionNotFound, http.StatusNotFound},
		"SESSION_ALREADY_EXISTS":  {ErrSessionAlreadyExists, http.StatusConflict},
		"INVALID_WEEK_NUMBER":     {ErrInvalidWeekNumber, http.StatusBadRequest},
		"STUDENT_NOT_FOUND":       {ErrStudentNotFound, http.StatusNotFound},
		"STUDENT_DEACTIVATED":     {ErrStudentDeactivated, http.StatusForbidden},
		"NOT_ENROLLED":            {ErrNotEnrolled, http.StatusForbidden},
		"ALREADY_MARKED":          {ErrAlreadyMarked, http.StatusConflict},
		"COURSE_NOT_FOUND":        {ErrCourseNotFound, http.StatusNotFound},
		"LAB_NOT_AVAILABLE":       {ErrLabNotAvailable, http.StatusBadRequest},
		"FORBIDDEN":               {ErrForbidden, http.StatusForbidden},
		"SEMESTER_ENDED":          {ErrSemesterEnded, http.StatusForbidden},
		"PERIOD_NOT_STARTED":      {ErrPeriodNotStarted, http.StatusForbidden},
		"PERIOD_ENDED":            {ErrPeriodEnded, http.StatusForbidden},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code)
		assert.Equal(t, c.want, c.err.HTTPStatus, "status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestAttendanceErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrInvalidQRCode, ErrSessionExpired, ErrSessionNotActive,
		ErrSessionNotFound, ErrSessionAlreadyExists, ErrInvalidWeekNumber,
		ErrStudentNotFound, ErrStudentDeactivated, ErrNotEnrolled, ErrAlreadyMarked,
		ErrCourseNotFound, ErrLabNotAvailable, ErrForbidden,
		ErrSemesterEnded, ErrPeriodNotStarted, ErrPeriodEnded,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate code %q", e.Code)
		seen[e.Code] = true
	}
}
