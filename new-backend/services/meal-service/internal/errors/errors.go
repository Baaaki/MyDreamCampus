package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// ============================================================================
// MEAL SERVICE SPECIFIC ERRORS
// These errors are specific to meal reservation and cafeteria management
// They should NOT be moved to shared/errors as they represent business logic
// unique to the meal service
// ============================================================================

var (
	// Cafeteria resource errors
	ErrCafeteriaNotFound  = sharedErrors.New("CAFETERIA_NOT_FOUND", "Yemekhane bulunamadı", http.StatusNotFound)
	ErrCafeteriaNotActive = sharedErrors.New("CAFETERIA_NOT_ACTIVE", "Yemekhane aktif değil", http.StatusBadRequest)

	// Cafeteria business logic errors
	ErrCafeteriaNoDinner = sharedErrors.New("CAFETERIA_NO_DINNER", "Bu yemekhanede akşam yemeği verilmiyor", http.StatusBadRequest)
	ErrCafeteriaNoVegan  = sharedErrors.New("CAFETERIA_NO_VEGAN", "Bu yemekhanede vegan menü yok", http.StatusBadRequest)

	// Reservation resource errors
	ErrReservationNotFound = sharedErrors.New("RESERVATION_NOT_FOUND", "Rezervasyon bulunamadı", http.StatusNotFound)
	ErrNoReservation       = sharedErrors.New("NO_RESERVATION", "Bu öğün için rezervasyon bulunamadı", http.StatusNotFound)

	// Reservation business logic errors
	ErrActiveReservationExists = sharedErrors.New("ACTIVE_RESERVATION_EXISTS", "Bu gün ve öğün için zaten rezervasyonunuz var", http.StatusConflict)
	ErrReservationAlreadyUsed  = sharedErrors.New("RESERVATION_ALREADY_USED", "Bu rezervasyon zaten kullanıldı", http.StatusBadRequest)
	ErrInvalidStatusForCancel  = sharedErrors.New("INVALID_STATUS_FOR_CANCEL", "Yalnızca onaylanmış rezervasyonlar iptal edilebilir", http.StatusBadRequest)
	ErrNotOwner                = sharedErrors.New("NOT_OWNER", "Bu rezervasyon size ait değil", http.StatusForbidden)
	ErrCancelCutoffPassed      = sharedErrors.New("CANCEL_CUTOFF_PASSED", "Bu rezervasyonun iptal süresi geçti", http.StatusBadRequest)

	// Student cache errors
	ErrStudentDeactivated = sharedErrors.New("STUDENT_DEACTIVATED", "Öğrenci hesabı devre dışı", http.StatusForbidden)

	// Date and time validation errors
	ErrCafeteriaClosedOnDate    = sharedErrors.New("CAFETERIA_CLOSED", "Yemekhane bu tarihte kapalı (tatil)", http.StatusBadRequest)
	ErrInvalidDateRange         = sharedErrors.New("INVALID_DATE_RANGE", "Rezervasyon tarihi gelecek haftanın iş günlerinden biri olmalıdır", http.StatusBadRequest)
	ErrOutsideReservationWindow = sharedErrors.New("OUTSIDE_RESERVATION_WINDOW", "Rezervasyonlar yalnızca Pazartesi 08:00 ile Cuma 13:00 arasında yapılabilir", http.StatusBadRequest)
	ErrInvalidMealTime          = sharedErrors.New("INVALID_MEAL_TIME", "Geçersiz öğün (öğle veya akşam olmalıdır)", http.StatusBadRequest)
	ErrInvalidMenuType          = sharedErrors.New("INVALID_MENU_TYPE", "Geçersiz menü türü (normal veya vegan olmalıdır)", http.StatusBadRequest)

	// QR code errors
	ErrInvalidQR             = sharedErrors.New("INVALID_QR", "QR kod geçersiz", http.StatusBadRequest)
	ErrInvalidQRDate         = sharedErrors.New("INVALID_QR_DATE", "QR kod bugün için geçerli değil", http.StatusBadRequest)
	ErrOutsideMealTimeWindow = sharedErrors.New("OUTSIDE_MEAL_TIME_WINDOW", "QR okutma saati öğün saatlerinin dışında", http.StatusBadRequest)

	// Batch reservation errors
	ErrReservationConflicts = sharedErrors.New("RESERVATION_CONFLICTS", "Bazı gün ve öğünler için zaten rezervasyonunuz var", http.StatusConflict)
	ErrValidationErrors     = sharedErrors.New("VALIDATION_ERRORS", "Bazı rezervasyonlar geçersiz", http.StatusBadRequest)

	// Payment service errors
	ErrPaymentServiceError = sharedErrors.New("PAYMENT_SERVICE_ERROR", "Ödeme hizmetine şu anda ulaşılamıyor, lütfen birazdan tekrar deneyin", http.StatusFailedDependency)
	ErrPaymentFailed       = sharedErrors.New("PAYMENT_FAILED", "Ödeme başlatılamadı", http.StatusFailedDependency)
	ErrRefundFailed        = sharedErrors.New("REFUND_FAILED", "İade yapılamadı, lütfen daha sonra tekrar deneyin", http.StatusFailedDependency)

	// Repository-specific sentinel errors (for internal use)
	ErrCafeteriaNotFoundRepo   = sharedErrors.ErrNotFoundRepo
	ErrReservationNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
