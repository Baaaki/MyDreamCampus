package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
)

// ============================================================================
// MEAL SERVICE SPECIFIC ERRORS
// These errors are specific to meal reservation and cafeteria management
// They should NOT be moved to shared/errors as they represent business logic
// unique to the meal service
// ============================================================================

var (
	// Cafeteria resource errors
	ErrCafeteriaNotFound = sharedErrors.New("CAFETERIA_NOT_FOUND", "Cafeteria not found", http.StatusNotFound)
	ErrCafeteriaNotActive = sharedErrors.New("CAFETERIA_NOT_ACTIVE", "Cafeteria is not active", http.StatusBadRequest)

	// Cafeteria business logic errors
	ErrCafeteriaNoDinner = sharedErrors.New("CAFETERIA_NO_DINNER", "This cafeteria does not serve dinner", http.StatusBadRequest)
	ErrCafeteriaNoVegan  = sharedErrors.New("CAFETERIA_NO_VEGAN", "This cafeteria does not have vegan menu", http.StatusBadRequest)

	// Reservation resource errors
	ErrReservationNotFound = sharedErrors.New("RESERVATION_NOT_FOUND", "Reservation not found", http.StatusNotFound)
	ErrNoReservation       = sharedErrors.New("NO_RESERVATION", "No reservation found for this slot", http.StatusNotFound)

	// Reservation business logic errors
	ErrActiveReservationExists  = sharedErrors.New("ACTIVE_RESERVATION_EXISTS", "Active or pending reservation already exists for this date and meal time", http.StatusConflict)
	ErrReservationAlreadyUsed   = sharedErrors.New("RESERVATION_ALREADY_USED", "This reservation has already been used", http.StatusBadRequest)
	ErrInvalidStatusForCancel   = sharedErrors.New("INVALID_STATUS_FOR_CANCEL", "Only confirmed reservations can be cancelled", http.StatusBadRequest)
	ErrNotOwner                 = sharedErrors.New("NOT_OWNER", "This reservation does not belong to you", http.StatusForbidden)
	ErrCancelCutoffPassed       = sharedErrors.New("CANCEL_CUTOFF_PASSED", "Cancellation window has closed for this reservation", http.StatusBadRequest)

	// Student cache errors
	ErrStudentDeactivated = sharedErrors.New("STUDENT_DEACTIVATED", "Student account has been deactivated", http.StatusForbidden)

	// Date and time validation errors
	ErrCafeteriaClosedOnDate       = sharedErrors.New("CAFETERIA_CLOSED", "Cafeteria is closed on this date (holiday)", http.StatusBadRequest)
	ErrInvalidDateRange            = sharedErrors.New("INVALID_DATE_RANGE", "Reservation date must be in next week (Monday-Friday)", http.StatusBadRequest)
	ErrOutsideReservationWindow    = sharedErrors.New("OUTSIDE_RESERVATION_WINDOW", "Reservations can only be made Monday 08:00 - Friday 13:00 (UTC+3)", http.StatusBadRequest)
	ErrInvalidMealTime             = sharedErrors.New("INVALID_MEAL_TIME", "Invalid meal time (must be 'lunch' or 'dinner')", http.StatusBadRequest)
	ErrInvalidMenuType             = sharedErrors.New("INVALID_MENU_TYPE", "Invalid menu type (must be 'normal' or 'vegan')", http.StatusBadRequest)

	// QR code errors
	ErrInvalidQR                 = sharedErrors.New("INVALID_QR", "Invalid QR code format or signature", http.StatusBadRequest)
	ErrInvalidQRDate             = sharedErrors.New("INVALID_QR_DATE", "QR code is not valid for today", http.StatusBadRequest)
	ErrOutsideMealTimeWindow     = sharedErrors.New("OUTSIDE_MEAL_TIME_WINDOW", "QR scan is outside the allowed meal time window", http.StatusBadRequest)

	// Batch reservation errors
	ErrReservationConflicts = sharedErrors.New("RESERVATION_CONFLICTS", "Some dates/meals already have active reservations", http.StatusConflict)
	ErrValidationErrors     = sharedErrors.New("VALIDATION_ERRORS", "Some reservations have validation errors", http.StatusBadRequest)

	// Payment service errors
	ErrPaymentServiceError = sharedErrors.New("PAYMENT_SERVICE_ERROR", "Payment service is unavailable or returned an error", http.StatusFailedDependency)
	ErrPaymentFailed       = sharedErrors.New("PAYMENT_FAILED", "Payment initiation failed", http.StatusFailedDependency)
	ErrRefundFailed        = sharedErrors.New("REFUND_FAILED", "Refund operation failed. Please try again later", http.StatusFailedDependency)

	// Repository-specific sentinel errors (for internal use)
	ErrCafeteriaNotFoundRepo  = sharedErrors.ErrNotFoundRepo
	ErrReservationNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
