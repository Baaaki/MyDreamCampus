package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

var (
	// Resource errors. A payment that belongs to someone else is reported as
	// not found, so ids cannot be probed.
	ErrPaymentNotFound = sharedErrors.New("PAYMENT_NOT_FOUND", "Ödeme bulunamadı", http.StatusNotFound)

	// State errors
	ErrPaymentNotPending = sharedErrors.New("PAYMENT_NOT_PENDING", "Bu ödeme artık işlem beklemiyor", http.StatusConflict)
	ErrPaymentExpired    = sharedErrors.New("PAYMENT_EXPIRED", "Ödeme süresi doldu, lütfen rezervasyonu yeniden oluşturun", http.StatusConflict)
	ErrRefundNotAllowed  = sharedErrors.New("REFUND_NOT_ALLOWED", "Yalnız tamamlanmış ödemeler iade edilebilir", http.StatusConflict)
	ErrRefundTooLarge    = sharedErrors.New("REFUND_TOO_LARGE", "İade tutarı ödenen tutarı aşıyor", http.StatusConflict)

	// Request errors
	ErrInvalidInitiate = sharedErrors.New("INVALID_PAYMENT_REQUEST", "Ödeme isteği geçersiz", http.StatusBadRequest)

	// Repository sentinel
	ErrPaymentNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
