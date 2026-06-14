package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/stretchr/testify/assert"
)

func TestStudentErrors_HTTPStatuses(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"STUDENT_NOT_FOUND":         {ErrStudentNotFound, http.StatusNotFound},
		"STUDENT_NUMBER_EXISTS":     {ErrStudentNumberExists, http.StatusConflict},
		"STUDENT_EMAIL_EXISTS":      {ErrStudentEmailExists, http.StatusConflict},
		"ADVISOR_NOT_FOUND":         {ErrAdvisorNotFound, http.StatusNotFound},
		"INVALID_CSV_FORMAT":        {ErrInvalidCSVFormat, http.StatusBadRequest},
		"STAFF_SERVICE_UNAVAILABLE": {ErrStaffServiceUnavailable, http.StatusServiceUnavailable},
		"ALREADY_ENROLLED":          {ErrStudentAlreadyEnrolled, http.StatusConflict},
		"GPA_TOO_LOW":               {ErrStudentGPALow, http.StatusBadRequest},
		"ENROLLMENT_FULL":           {ErrEnrollmentCapacity, http.StatusConflict},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code, "code mismatch")
		assert.Equal(t, c.want, c.err.HTTPStatus, "status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestStudentErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrStudentNotFound, ErrStudentNumberExists, ErrStudentEmailExists,
		ErrAdvisorNotFound,
		ErrInvalidCSVFormat,
		ErrStaffServiceUnavailable,
		ErrStudentAlreadyEnrolled, ErrStudentGPALow, ErrEnrollmentCapacity,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate error code %q", e.Code)
		seen[e.Code] = true
	}
}
