package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// ============================================================================
// STUDENT SERVICE SPECIFIC ERRORS
// These errors are specific to student management domain
// They should NOT be moved to shared/errors as they represent business logic
// unique to the student service
// ============================================================================

var (
	// Student resource errors
	ErrStudentNotFound     = sharedErrors.New("STUDENT_NOT_FOUND", "Öğrenci bulunamadı", http.StatusNotFound)
	ErrStudentNumberExists = sharedErrors.New("STUDENT_NUMBER_EXISTS", "Bu öğrenci numarası zaten kullanılıyor", http.StatusConflict)
	ErrStudentEmailExists  = sharedErrors.New("STUDENT_EMAIL_EXISTS", "Bu e-posta adresi zaten kullanılıyor", http.StatusConflict)

	// Advisor-related errors
	ErrAdvisorNotFound = sharedErrors.New("ADVISOR_NOT_FOUND", "Danışman bulunamadı", http.StatusNotFound)

	// Import/bulk operation errors
	ErrInvalidCSVFormat = sharedErrors.New("INVALID_CSV_FORMAT", "Geçersiz CSV biçimi", http.StatusBadRequest)

	// External service errors
	ErrStaffServiceUnavailable = sharedErrors.New("STAFF_SERVICE_UNAVAILABLE", "Personel bilgisi şu anda alınamıyor, lütfen birazdan tekrar deneyin", http.StatusServiceUnavailable)

	// Future enrollment-related errors (for when enrollment features are implemented)
	ErrStudentAlreadyEnrolled = sharedErrors.New("ALREADY_ENROLLED", "Öğrenci bu derse zaten kayıtlı", http.StatusConflict)
	ErrStudentGPALow          = sharedErrors.New("GPA_TOO_LOW", "Öğrencinin not ortalaması ders için yeterli değil", http.StatusBadRequest)
	ErrEnrollmentCapacity     = sharedErrors.New("ENROLLMENT_FULL", "Ders kontenjanı dolu", http.StatusConflict)

	// Protected system account errors
	ErrProtectedAccountDeletionForbidden     = sharedErrors.New("PROTECTED_ACCOUNT_DELETION_FORBIDDEN", "Korunan sistem hesabı silinemez", http.StatusForbidden)
	ErrProtectedAccountDeactivationForbidden = sharedErrors.New("PROTECTED_ACCOUNT_DEACTIVATION_FORBIDDEN", "Korunan sistem hesabı pasifleştirilemez", http.StatusForbidden)
	ErrProtectedAccountEmailChangeForbidden  = sharedErrors.New("PROTECTED_ACCOUNT_EMAIL_CHANGE_FORBIDDEN", "Korunan sistem hesabının e-postası değiştirilemez", http.StatusForbidden)

	// Repository-specific sentinel errors (for internal use)
	ErrStudentNotFoundRepo     = sharedErrors.ErrNotFoundRepo
	ErrStudentNumberExistsRepo = sharedErrors.ErrAlreadyExistsRepo
	ErrStudentEmailExistsRepo  = sharedErrors.ErrAlreadyExistsRepo
)
