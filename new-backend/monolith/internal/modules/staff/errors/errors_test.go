package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/stretchr/testify/assert"
)

func TestStaffErrors_HTTPStatuses(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"STAFF_NOT_FOUND":          {ErrStaffNotFound, http.StatusNotFound},
		"STAFF_EXISTS":             {ErrStaffExists, http.StatusConflict},
		"EMAIL_EXISTS":             {ErrEmailExists, http.StatusConflict},
		"CANNOT_CREATE_ADMIN":      {ErrCannotCreateAdmin, http.StatusBadRequest},
		"INVALID_ROLE":             {ErrInvalidRole, http.StatusBadRequest},
		"ADVISOR_NOT_QUALIFIED":    {ErrAdvisorNotQualified, http.StatusBadRequest},
		"ADVISOR_OVERLOADED":       {ErrAdvisorHasTooManyStudents, http.StatusConflict},
		"TEACHER_PROFILE_NOT_FOUND": {ErrTeacherProfileNotFound, http.StatusNotFound},
		"NOT_A_TEACHER":            {ErrNotATeacher, http.StatusBadRequest},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code)
		assert.Equal(t, c.want, c.err.HTTPStatus, "status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestStaffErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrStaffNotFound, ErrStaffExists,
		ErrEmailExists, ErrCannotCreateAdmin, ErrInvalidRole,
		ErrAdvisorNotQualified, ErrAdvisorHasTooManyStudents,
		ErrTeacherProfileNotFound, ErrNotATeacher,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate error code %q", e.Code)
		seen[e.Code] = true
	}
}

func TestRepositorySentinels_AliasShared(t *testing.T) {
	assert.Same(t, sharedErrors.ErrNotFoundRepo, ErrStaffNotFoundRepo)
	assert.Same(t, sharedErrors.ErrAlreadyExistsRepo, ErrStaffExistsRepo)
	assert.Same(t, sharedErrors.ErrNotFoundRepo, ErrTeacherProfileNotFoundRepo)
}
