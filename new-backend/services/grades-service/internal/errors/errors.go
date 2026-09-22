package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// ============================================================================
// GRADES SERVICE SPECIFIC ERRORS
// These errors are specific to grade management and assessment scoring
// They should NOT be moved to shared/errors as they represent business logic
// unique to the grades service
// ============================================================================

var (
	// Grade submission errors (AppError for HTTP responses)
	ErrInvalidScore         = sharedErrors.New("INVALID_SCORE", "Not 0 ile 100 arasında olmalıdır", http.StatusBadRequest)
	ErrInvalidSlug          = sharedErrors.New("INVALID_SLUG", "Değerlendirme, dersin değerlendirme şemasında yok", http.StatusBadRequest)
	ErrAlreadyFinalized     = sharedErrors.New("ALREADY_FINALIZED", "Dersin notları kesinleşti, değiştirilemez", http.StatusConflict)
	ErrScoreExists          = sharedErrors.New("SCORE_EXISTS", "Bu değerlendirme için not zaten girilmiş", http.StatusConflict)
	ErrAttendanceFailed     = sharedErrors.New("ATTENDANCE_FAILED", "Öğrenci devamsızlıktan kaldı, not girilemez", http.StatusBadRequest)
	ErrScoreLocked          = sharedErrors.New("SCORE_LOCKED", "Not kilitli, değiştirilemez", http.StatusForbidden)
	ErrIncompleteAssessment = sharedErrors.New("INCOMPLETE_ASSESSMENT", "Değerlendirme kilitlenemez: her öğrencinin notu girilmemiş", http.StatusBadRequest)
	ErrGradingPeriodEnded   = sharedErrors.New("GRADING_PERIOD_ENDED", "Not giriş dönemi sona erdi", http.StatusForbidden)
	ErrNoPeriodDefined      = sharedErrors.New("NO_PERIOD_DEFINED", "Bu dönem için tanımlı bir not giriş dönemi yok", http.StatusBadRequest)
	// Raised when catalog cannot be reached: the hard deadline binds admins
	// too, so an unverifiable deadline blocks the edit.
	ErrSemesterInfoUnavailable = sharedErrors.New("SEMESTER_INFO_UNAVAILABLE", "Dönem bilgisi şu anda alınamıyor, lütfen birazdan tekrar deneyin", http.StatusServiceUnavailable)

	// Authorization errors (AppError for HTTP responses)
	ErrNotCourseInstructor = sharedErrors.New("NOT_COURSE_INSTRUCTOR", "Bu dersin öğretim üyesi değilsiniz", http.StatusForbidden)
	ErrStudentDeactivated  = sharedErrors.New("STUDENT_DEACTIVATED", "Öğrenci hesabı devre dışı", http.StatusForbidden)

	// Not found errors (AppError for HTTP responses)
	ErrCourseNotFound       = sharedErrors.New("COURSE_NOT_FOUND", "Ders bulunamadı", http.StatusNotFound)
	ErrRegistrationNotFound = sharedErrors.New("REGISTRATION_NOT_FOUND", "Ders kaydı bulunamadı", http.StatusNotFound)
	ErrScoreNotFound        = sharedErrors.New("SCORE_NOT_FOUND", "Not bulunamadı", http.StatusNotFound)
	ErrStudentNotFound      = sharedErrors.New("STUDENT_NOT_FOUND", "Öğrenci bulunamadı", http.StatusNotFound)

	// Repository-specific sentinel errors (for internal use)
	ErrCourseNotFoundRepo       = sharedErrors.ErrNotFoundRepo
	ErrRegistrationNotFoundRepo = sharedErrors.ErrNotFoundRepo
	ErrScoreNotFoundRepo        = sharedErrors.ErrNotFoundRepo
	ErrStudentNotFoundRepo      = sharedErrors.ErrNotFoundRepo
)
