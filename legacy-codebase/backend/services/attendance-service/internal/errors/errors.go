package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
)

// ============================================================================
// ATTENDANCE SERVICE SPECIFIC ERRORS
// These errors are specific to attendance management and QR code validation
// They should NOT be moved to shared/errors as they represent business logic
// unique to the attendance service
// ============================================================================

var (
	// QR Code errors (AppError for HTTP responses)
	ErrInvalidQRCode  = sharedErrors.New("INVALID_QR_CODE", "QR kod geçersiz", http.StatusBadRequest)
	ErrSessionExpired = sharedErrors.New("SESSION_EXPIRED", "Yoklama oturumu süresi dolmuş", http.StatusBadRequest)
	ErrSessionNotActive = sharedErrors.New("SESSION_NOT_ACTIVE", "Yoklama oturumu aktif değil", http.StatusBadRequest)

	// Session errors (AppError for HTTP responses)
	ErrSessionNotFound      = sharedErrors.New("SESSION_NOT_FOUND", "Yoklama oturumu bulunamadı", http.StatusNotFound)
	ErrSessionAlreadyExists = sharedErrors.New("SESSION_ALREADY_EXISTS", "Bu hafta için zaten yoklama oturumu var", http.StatusConflict)
	ErrInvalidWeekNumber    = sharedErrors.New("INVALID_WEEK_NUMBER", "Hafta numarası 1-14 aralığında olmalıdır", http.StatusBadRequest)

	// Student errors (AppError for HTTP responses)
	ErrStudentNotFound    = sharedErrors.New("STUDENT_NOT_FOUND", "Öğrenci bulunamadı", http.StatusNotFound)
	ErrStudentDeactivated = sharedErrors.New("STUDENT_DEACTIVATED", "Öğrenci deaktif edilmiş", http.StatusForbidden)
	ErrNotEnrolled   = sharedErrors.New("NOT_ENROLLED", "Bu derse kayıtlı değilsiniz", http.StatusForbidden)
	ErrAlreadyMarked = sharedErrors.New("ALREADY_MARKED", "Bu dersin yoklaması zaten alındı", http.StatusConflict)

	// Course errors (AppError for HTTP responses)
	ErrCourseNotFound    = sharedErrors.New("COURSE_NOT_FOUND", "Ders bulunamadı", http.StatusNotFound)
	ErrLabNotAvailable   = sharedErrors.New("LAB_NOT_AVAILABLE", "Bu dersin uygulama (lab) saati bulunmamaktadır", http.StatusBadRequest)

	// Authorization errors (AppError for HTTP responses)
	ErrForbidden = sharedErrors.New("FORBIDDEN", "Bu işlem için yetkiniz yok", http.StatusForbidden)

	// Semester enforcement errors
	ErrSemesterEnded       = sharedErrors.New("SEMESTER_ENDED", "Dönem sona ermiştir — değişiklik yapılamaz", http.StatusForbidden)
	ErrPeriodNotStarted    = sharedErrors.New("PERIOD_NOT_STARTED", "Yoklama dönemi henüz başlamadı", http.StatusForbidden)
	ErrPeriodEnded         = sharedErrors.New("PERIOD_ENDED", "Yoklama dönemi sona ermiştir", http.StatusForbidden)

	// Repository-specific sentinel errors (for internal use)
	ErrSessionNotFoundRepo = sharedErrors.ErrNotFoundRepo
	ErrStudentNotFoundRepo = sharedErrors.ErrNotFoundRepo
	ErrCourseNotFoundRepo  = sharedErrors.ErrNotFoundRepo
)
