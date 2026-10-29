package errors

import (
	"errors"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// Enrollment-specific AppErrors (with HTTP status for handler mapping)
var (
	ErrEnrollmentPeriodEnded   = sharedErrors.New("ENROLLMENT_PERIOD_ENDED", "Bu dönem için ders kayıt süresi sona erdi", http.StatusForbidden)
	ErrEnrollmentPeriodNotOpen = sharedErrors.New("ENROLLMENT_PERIOD_NOT_OPEN", "Ders kayıt dönemi henüz başlamadı", http.StatusForbidden)
)

// Enrollment-specific errors
var (
	// Student errors
	ErrStudentDeactivated = errors.New("student account is deactivated")
	ErrStudentNotFound    = errors.New("student not found in cache")

	// Enrollment errors
	ErrProgramNotFound      = errors.New("enrollment program not found")
	ErrInvalidStatus        = errors.New("invalid enrollment status")
	ErrCannotModifyApproved = errors.New("cannot modify approved enrollment program")
	// ErrCannotWithdrawApproved: an approved program is changed by sending a
	// new selection while the window is open, never withdrawn outright.
	ErrCannotWithdrawApproved = errors.New("cannot withdraw approved enrollment program")
	// ErrApprovedProgramInUse: the semester already has an attendance record
	// or a score for the student, which changing the program would orphan.
	ErrApprovedProgramInUse = errors.New("approved enrollment program already has attendance or grades")
	ErrProgramNotPending    = errors.New("enrollment program is not pending")
	// ErrSubmissionConflict: another write to the student's program for the
	// semester committed between this submission's read and its transaction.
	ErrSubmissionConflict = errors.New("enrollment program changed during submission")

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
	{ErrProgramNotFound, sharedErrors.New("PROGRAM_NOT_FOUND", "Ders kayıt programı bulunamadı", http.StatusNotFound)},
	{ErrInvalidStatus, sharedErrors.New("INVALID_STATUS", "Geçersiz kayıt durumu", http.StatusBadRequest)},
	{ErrCannotModifyApproved, sharedErrors.New("CANNOT_MODIFY_APPROVED", "Onaylanmış ders kaydı değiştirilemez", http.StatusForbidden)},
	{ErrCannotWithdrawApproved, sharedErrors.New("CANNOT_WITHDRAW_APPROVED", "Onaylanmış ders kaydı geri çekilemez; değiştirmek için yeni ders seçimini gönder", http.StatusForbidden)},
	{ErrApprovedProgramInUse, sharedErrors.New("APPROVED_PROGRAM_IN_USE", "Bu dönemin derslerinde yoklama veya not kaydınız olduğu için onaylanmış ders kaydınız değiştirilemez", http.StatusConflict)},
	{ErrProgramNotPending, sharedErrors.New("PROGRAM_NOT_PENDING", "Ders kayıt programı artık onay beklemiyor", http.StatusConflict)},
	{ErrSubmissionConflict, sharedErrors.New("SUBMISSION_CONFLICT", "Ders kaydınız aynı anda başka bir işlemle değişti, lütfen sayfayı yenileyip tekrar deneyin", http.StatusConflict)},
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
