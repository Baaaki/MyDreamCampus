package service

import (
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment-service/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
)

// Pure-function validators for enrollment program submission. These run before
// any DB call, so testing them in isolation is the cheapest way to lock in the
// rules. The validation order is contractual — admin tooling expects the same
// sequence of error codes.

// validateCourseSelection checks the basic shape of the requested course list:
// non-empty, under the enrollment cap, and free of duplicates. Returning a
// nil error means the list is structurally valid (still subject to deeper
// DB-backed checks like capacity, prereqs, schedule conflicts).
func validateCourseSelection(courseIDs []uuid.UUID) error {
	if len(courseIDs) == 0 {
		return serviceErrors.ErrNoCourses
	}
	if len(courseIDs) > serviceErrors.MaxCoursesPerEnrollment {
		return serviceErrors.ErrTooManyCourses
	}
	seen := make(map[uuid.UUID]struct{}, len(courseIDs))
	for _, id := range courseIDs {
		if _, dup := seen[id]; dup {
			return serviceErrors.ErrDuplicateCourse
		}
		seen[id] = struct{}{}
	}
	return nil
}

// validateCoursesAgainstStudent checks each course is in the student's own
// department and at-or-below the student's class level. Both rules exist
// because the catalog can serve cross-department courses for shared classes;
// guarding here keeps that possibility from leaking into enrollment by
// accident.
func validateCoursesAgainstStudent(courses []db.SemesterCoursesCache, studentDept string, studentClassLevel int16) error {
	for _, course := range courses {
		if utils.PgTextToString(course.Department) != studentDept {
			return serviceErrors.ErrInvalidDepartment
		}
		if course.ClassLevel.Int16 > studentClassLevel {
			return serviceErrors.ErrInvalidClassLevel
		}
	}
	return nil
}
