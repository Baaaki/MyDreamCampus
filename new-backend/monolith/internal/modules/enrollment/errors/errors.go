package errors

import (
	"errors"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
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
	ErrAlreadySubmitted     = errors.New("enrollment program already submitted for this semester")
	ErrProgramNotFound      = errors.New("enrollment program not found")
	ErrInvalidStatus        = errors.New("invalid enrollment status")
	ErrCannotModifyApproved = errors.New("cannot modify approved enrollment program")

	// Course errors
	ErrCourseFull        = errors.New("course capacity is full")
	ErrCourseNotFound    = errors.New("course not found")
	ErrInvalidDepartment = errors.New("course does not belong to student's department")
	ErrInvalidClassLevel = errors.New("course class level exceeds student's class level")

	// Prerequisite errors
	ErrPrerequisitesNotMet  = errors.New("prerequisites not met for one or more courses")
	ErrPrerequisiteNotFound = errors.New("prerequisite information not found")

	// Schedule errors
	ErrScheduleConflict = errors.New("schedule conflict detected between courses")

	// Validation errors
	ErrInvalidSemester = errors.New("invalid semester format")
	ErrNoCourses       = errors.New("no courses provided in enrollment request")
	ErrInvalidCourseID = errors.New("invalid course ID")
	ErrTooManyCourses  = errors.New("too many courses selected for enrollment")
	ErrDuplicateCourse = errors.New("duplicate course in enrollment request")
)

// Enrollment business limits
const (
	MaxCoursesPerEnrollment = 10
)

// sentinelMapping pairs each plain sentinel with the AppError the HTTP layer
// should surface. Sentinels stay plain (see TestSentinels_AreNotAppErrors) so
// service-layer comparisons keep using errors.Is; the mapping lives here so
// every handler translates them to the same status/code/Turkish message.
var sentinelMapping = []struct {
	sentinel error
	appErr   *sharedErrors.AppError
}{
	{ErrStudentDeactivated, sharedErrors.New("STUDENT_DEACTIVATED", "Öğrenci hesabı devre dışı", http.StatusForbidden)},
	{ErrStudentNotFound, sharedErrors.New("STUDENT_NOT_FOUND", "Öğrenci bulunamadı", http.StatusNotFound)},
	{ErrAlreadySubmitted, sharedErrors.New("ALREADY_SUBMITTED", "Bu dönem için ders kaydı zaten yapılmış", http.StatusConflict)},
	{ErrProgramNotFound, sharedErrors.New("PROGRAM_NOT_FOUND", "Ders kayıt programı bulunamadı", http.StatusNotFound)},
	{ErrInvalidStatus, sharedErrors.New("INVALID_STATUS", "Geçersiz kayıt durumu", http.StatusBadRequest)},
	{ErrCannotModifyApproved, sharedErrors.New("CANNOT_MODIFY_APPROVED", "Onaylanmış ders kaydı değiştirilemez", http.StatusForbidden)},
	{ErrCourseFull, sharedErrors.New("COURSE_FULL", "Ders kontenjanı dolu", http.StatusConflict)},
	{ErrCourseNotFound, sharedErrors.New("COURSE_NOT_FOUND", "Ders bulunamadı", http.StatusNotFound)},
	{ErrInvalidDepartment, sharedErrors.New("INVALID_DEPARTMENT", "Ders öğrencinin bölümüne ait değil", http.StatusUnprocessableEntity)},
	{ErrInvalidClassLevel, sharedErrors.New("INVALID_CLASS_LEVEL", "Ders öğrencinin sınıf seviyesinin üzerinde", http.StatusUnprocessableEntity)},
	{ErrPrerequisitesNotMet, sharedErrors.New("PREREQUISITES_NOT_MET", "Bir veya daha fazla dersin ön koşulu sağlanmadı", http.StatusUnprocessableEntity)},
	{ErrPrerequisiteNotFound, sharedErrors.New("PREREQUISITE_NOT_FOUND", "Ön koşul bilgisi bulunamadı", http.StatusUnprocessableEntity)},
	{ErrScheduleConflict, sharedErrors.New("SCHEDULE_CONFLICT", "Dersler arasında program çakışması var", http.StatusConflict)},
	{ErrInvalidSemester, sharedErrors.New("INVALID_SEMESTER", "Geçersiz dönem formatı", http.StatusBadRequest)},
	{ErrNoCourses, sharedErrors.New("NO_COURSES", "Ders kaydı için ders seçilmedi", http.StatusBadRequest)},
	{ErrInvalidCourseID, sharedErrors.New("INVALID_COURSE_ID", "Geçersiz ders kimliği", http.StatusBadRequest)},
	{ErrTooManyCourses, sharedErrors.New("TOO_MANY_COURSES", "Çok fazla ders seçildi", http.StatusBadRequest)},
	{ErrDuplicateCourse, sharedErrors.New("DUPLICATE_COURSE", "Ders kaydında tekrar eden ders var", http.StatusBadRequest)},
}

// MapSentinel returns the AppError a known enrollment sentinel should map to at
// the HTTP boundary. Returns (nil, false) for anything else so the caller can
// fall back to a generic 500.
func MapSentinel(err error) (*sharedErrors.AppError, bool) {
	for _, m := range sentinelMapping {
		if errors.Is(err, m.sentinel) {
			return m.appErr, true
		}
	}
	return nil, false
}
