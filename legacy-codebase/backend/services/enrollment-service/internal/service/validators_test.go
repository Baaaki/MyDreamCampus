package service

import (
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment-service/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestValidateCourseSelection(t *testing.T) {
	a := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	b := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	t.Run("empty list", func(t *testing.T) {
		err := validateCourseSelection(nil)
		assert.ErrorIs(t, err, serviceErrors.ErrNoCourses,
			"empty selection must return ErrNoCourses, not pass through to ErrTooManyCourses")
	})

	t.Run("single course", func(t *testing.T) {
		assert.NoError(t, validateCourseSelection([]uuid.UUID{a}))
	})

	t.Run("at the cap", func(t *testing.T) {
		ids := make([]uuid.UUID, serviceErrors.MaxCoursesPerEnrollment)
		for i := range ids {
			ids[i] = uuid.New()
		}
		assert.NoError(t, validateCourseSelection(ids),
			"a selection exactly at the cap must be accepted — boundary fence")
	})

	t.Run("over the cap", func(t *testing.T) {
		ids := make([]uuid.UUID, serviceErrors.MaxCoursesPerEnrollment+1)
		for i := range ids {
			ids[i] = uuid.New()
		}
		err := validateCourseSelection(ids)
		assert.ErrorIs(t, err, serviceErrors.ErrTooManyCourses)
	})

	t.Run("duplicate id", func(t *testing.T) {
		err := validateCourseSelection([]uuid.UUID{a, b, a})
		assert.ErrorIs(t, err, serviceErrors.ErrDuplicateCourse,
			"the dedup pass is the last line of defence against double-enrollment when the UI sends a stale list")
	})

	t.Run("dedup runs after cap check", func(t *testing.T) {
		// If the cap is N, a list of N+1 items where two are duplicates is
		// still over-cap — the cap check fires first by design (callers see
		// ErrTooManyCourses, not ErrDuplicateCourse). This locks the order.
		ids := make([]uuid.UUID, 0, serviceErrors.MaxCoursesPerEnrollment+1)
		for i := 0; i < serviceErrors.MaxCoursesPerEnrollment-1; i++ {
			ids = append(ids, uuid.New())
		}
		ids = append(ids, a, a) // 2 duplicates push it over the cap
		err := validateCourseSelection(ids)
		assert.ErrorIs(t, err, serviceErrors.ErrTooManyCourses,
			"validation order: cap check must precede dedup")
	})
}

func TestValidateCoursesAgainstStudent(t *testing.T) {
	cs := func(dept string, level int16) db.SemesterCoursesCache {
		return db.SemesterCoursesCache{
			CourseCode: "CS101",
			Department: pgtype.Text{String: dept, Valid: true},
			ClassLevel: pgtype.Int2{Int16: level, Valid: true},
		}
	}

	t.Run("all matching", func(t *testing.T) {
		err := validateCoursesAgainstStudent(
			[]db.SemesterCoursesCache{cs("CS", 1), cs("CS", 2)},
			"CS", 2,
		)
		assert.NoError(t, err)
	})

	t.Run("first course wrong department", func(t *testing.T) {
		err := validateCoursesAgainstStudent(
			[]db.SemesterCoursesCache{cs("EE", 1), cs("CS", 1)},
			"CS", 2,
		)
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidDepartment,
			"a single off-department course must reject the whole submission")
	})

	t.Run("level above student", func(t *testing.T) {
		err := validateCoursesAgainstStudent(
			[]db.SemesterCoursesCache{cs("CS", 3)},
			"CS", 2,
		)
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidClassLevel,
			"a 3rd-year course must be denied to a 2nd-year student — capacity bypass cannot live behind class level")
	})

	t.Run("at student level", func(t *testing.T) {
		err := validateCoursesAgainstStudent(
			[]db.SemesterCoursesCache{cs("CS", 2)},
			"CS", 2,
		)
		assert.NoError(t, err, "level == student level is allowed (>= boundary)")
	})

	t.Run("below student level", func(t *testing.T) {
		err := validateCoursesAgainstStudent(
			[]db.SemesterCoursesCache{cs("CS", 1)},
			"CS", 3,
		)
		assert.NoError(t, err, "lower-level courses are allowed (retake / catch-up)")
	})

	t.Run("empty list", func(t *testing.T) {
		err := validateCoursesAgainstStudent(nil, "CS", 2)
		assert.NoError(t, err,
			"empty list is structurally accepted here — emptiness is rejected earlier in validateCourseSelection")
	})
}
