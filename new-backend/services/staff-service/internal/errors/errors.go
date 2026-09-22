package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// ============================================================================
// STAFF SERVICE SPECIFIC ERRORS
// These errors are specific to staff management domain
// They should NOT be moved to shared/errors as they represent business logic
// unique to the staff service
// ============================================================================

var (
	// Staff resource errors
	ErrStaffNotFound = sharedErrors.New("STAFF_NOT_FOUND", "Personel bulunamadı", http.StatusNotFound)
	ErrStaffExists   = sharedErrors.New("STAFF_EXISTS", "Personel zaten mevcut", http.StatusConflict)

	// Staff business logic errors
	ErrEmailExists       = sharedErrors.New("EMAIL_EXISTS", "Bu e-posta adresi zaten kullanılıyor", http.StatusConflict)
	ErrCannotCreateAdmin = sharedErrors.New("CANNOT_CREATE_ADMIN", "Yönetici hesabı bu yolla oluşturulamaz", http.StatusBadRequest)
	ErrInvalidRole       = sharedErrors.New("INVALID_ROLE", "Geçersiz rol", http.StatusBadRequest)

	// Advisor-specific business errors (for future use when advisor features are implemented)
	ErrAdvisorNotQualified       = sharedErrors.New("ADVISOR_NOT_QUALIFIED", "Bu personel danışman olamaz", http.StatusBadRequest)
	ErrAdvisorHasTooManyStudents = sharedErrors.New("ADVISOR_OVERLOADED", "Danışmanın öğrenci kontenjanı dolu", http.StatusConflict)

	// Teacher profile errors
	ErrTeacherProfileNotFound = sharedErrors.New("TEACHER_PROFILE_NOT_FOUND", "Öğretim üyesi profili bulunamadı", http.StatusNotFound)
	ErrNotATeacher            = sharedErrors.New("NOT_A_TEACHER", "Bu personel öğretim üyesi değil", http.StatusBadRequest)

	// Repository-specific sentinel errors (for internal use)
	ErrStaffNotFoundRepo          = sharedErrors.ErrNotFoundRepo
	ErrStaffExistsRepo            = sharedErrors.ErrAlreadyExistsRepo
	ErrTeacherProfileNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
