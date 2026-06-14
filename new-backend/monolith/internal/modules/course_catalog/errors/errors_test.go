package errors

import (
	stdlib "errors"
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/stretchr/testify/assert"
)

func TestCatalogErrors_HTTPStatuses(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"COURSE_NOT_FOUND":              {ErrCourseNotFound, http.StatusNotFound},
		"COURSE_CODE_EXISTS":            {ErrCourseCodeExists, http.StatusConflict},
		"INVALID_PREREQUISITE":          {ErrInvalidPrerequisite, http.StatusBadRequest},
		"INVALID_PREREQUISITE_LEVEL":    {ErrInvalidPrerequisiteLevel, http.StatusBadRequest},
		"INVALID_STATUS":                {ErrInvalidStatus, http.StatusBadRequest},
		"COURSE_NOT_ACTIVE":             {ErrCourseNotActive, http.StatusBadRequest},
		"INVALID_SEMESTER_FORMAT":       {ErrInvalidSemesterFormat, http.StatusBadRequest},
		"SEMESTER_COURSE_NOT_FOUND":     {ErrSemesterCourseNotFound, http.StatusNotFound},
		"COURSE_ALREADY_OPENED":         {ErrCourseAlreadyOpened, http.StatusConflict},
		"CLASS_LEVEL_MISMATCH":          {ErrClassLevelMismatch, http.StatusBadRequest},
		"SEMESTER_ALREADY_EXISTS":       {ErrSemesterAlreadyExists, http.StatusConflict},
		"SOURCE_SEMESTER_NOT_FOUND":     {ErrSourceSemesterNotFound, http.StatusNotFound},
		"COURSE_HAS_ENROLLMENTS":        {ErrCourseHasEnrollments, http.StatusConflict},
		"INSTRUCTOR_NOT_FOUND":          {ErrInstructorNotFound, http.StatusNotFound},
		"INSTRUCTOR_NOT_ACTIVE":         {ErrInstructorNotActive, http.StatusBadRequest},
		"INSTRUCTOR_NOT_IN_DEPARTMENT":  {ErrInstructorNotInDepartment, http.StatusBadRequest},
		"INSTRUCTOR_SCHEDULE_CONFLICT":  {ErrInstructorScheduleConflict, http.StatusConflict},
		"INVALID_SLOT_NUMBER":           {ErrInvalidSlotNumber, http.StatusBadRequest},
		"INVALID_DAY_OF_WEEK":           {ErrInvalidDayOfWeek, http.StatusBadRequest},
		"INVALID_SESSION_TYPE":          {ErrInvalidSessionType, http.StatusBadRequest},
		"THEORY_HOURS_ZERO":             {ErrTheoryHoursZero, http.StatusBadRequest},
		"LAB_HOURS_ZERO":                {ErrLabHoursZero, http.StatusBadRequest},
		"THEORY_SLOT_COUNT_MISMATCH":    {ErrTheorySlotCountMismatch, http.StatusBadRequest},
		"LAB_SLOT_COUNT_MISMATCH":       {ErrLabSlotCountMismatch, http.StatusBadRequest},
		"COURSE_CREDITS_ZERO":           {ErrCourseCreditsZero, http.StatusBadRequest},
		"INVALID_ASSESSMENT_SCHEMA":     {ErrInvalidAssessmentSchema, http.StatusBadRequest},
		"ASSESSMENT_WEIGHT_NOT_HUNDRED": {ErrAssessmentWeightNotHundred, http.StatusBadRequest},
		"DUPLICATE_ASSESSMENT_SLUG":     {ErrDuplicateAssessmentSlug, http.StatusBadRequest},
		"SEMESTER_NOT_ACTIVE":           {ErrSemesterNotActive, http.StatusForbidden},
		"SEMESTER_COURSE_FROZEN":        {ErrSemesterCourseFrozen, http.StatusForbidden},
		"SEMESTER_NOT_PLANNED":          {ErrSemesterNotPlanned, http.StatusForbidden},
		"COURSE_CREATION_PERIOD_ENDED":  {ErrCourseCreationPeriodEnded, http.StatusForbidden},
		"COURSE_CREATION_PERIOD_NOT_OPEN": {ErrCourseCreationPeriodNotOpen, http.StatusForbidden},
		"TRANSACTION_FAILED":            {ErrTransactionFailed, http.StatusInternalServerError},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code)
		assert.Equal(t, c.want, c.err.HTTPStatus, "status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestCatalogErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrCourseNotFound, ErrCourseCodeExists, ErrInvalidPrerequisite, ErrInvalidPrerequisiteLevel,
		ErrInvalidStatus, ErrCourseNotActive,
		ErrInvalidSemesterFormat, ErrSemesterCourseNotFound, ErrCourseAlreadyOpened,
		ErrClassLevelMismatch, ErrSemesterAlreadyExists, ErrSourceSemesterNotFound,
		ErrCourseHasEnrollments,
		ErrInstructorNotFound, ErrInstructorNotActive, ErrInstructorNotInDepartment,
		ErrInstructorScheduleConflict,
		ErrInvalidSlotNumber, ErrInvalidDayOfWeek, ErrInvalidSessionType,
		ErrTheoryHoursZero, ErrLabHoursZero,
		ErrTheorySlotCountMismatch, ErrLabSlotCountMismatch, ErrCourseCreditsZero,
		ErrInvalidAssessmentSchema, ErrAssessmentWeightNotHundred, ErrDuplicateAssessmentSlug,
		ErrSemesterNotActive, ErrSemesterCourseFrozen, ErrSemesterNotPlanned,
		ErrCourseCreationPeriodEnded, ErrCourseCreationPeriodNotOpen,
		ErrTransactionFailed,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate code %q", e.Code)
		seen[e.Code] = true
	}
}

func TestScheduleConflictError_CarriesContext(t *testing.T) {
	e := NewScheduleConflictError("CSE201", "CS", "Monday", 3)
	assert.Equal(t, "CSE201", e.CourseCode)
	assert.Equal(t, "CS", e.Department)
	assert.Equal(t, "Monday", e.DayOfWeek)
	assert.Equal(t, int16(3), e.SlotNumber)

	assert.Contains(t, e.Error(), "INSTRUCTOR_SCHEDULE_CONFLICT")
	assert.NotNil(t, e.Unwrap())
}

func TestScheduleConflictError_UnwrapsToAppError(t *testing.T) {
	e := NewScheduleConflictError("X", "Y", "Mon", 1)

	// errors.Is must reach the underlying AppError
	assert.True(t, stdlib.Is(e, ErrInstructorScheduleConflict),
		"errors.Is must traverse to wrapped AppError")

	// errors.As must extract the AppError
	var app *sharedErrors.AppError
	assert.True(t, stdlib.As(e, &app))
	assert.Equal(t, "INSTRUCTOR_SCHEDULE_CONFLICT", app.Code)
}
