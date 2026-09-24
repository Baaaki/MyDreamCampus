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

	// Card validation errors. 422 and not a failed payment: the student
	// fixes the form and tries again on the same payment.
	ErrInvalidCardNumber = sharedErrors.New("INVALID_CARD_NUMBER", "Kart numarası geçersiz", http.StatusUnprocessableEntity)
	ErrInvalidExpiry     = sharedErrors.New("INVALID_CARD_EXPIRY", "Son kullanma tarihi geçersiz", http.StatusUnprocessableEntity)
	ErrCardExpired       = sharedErrors.New("CARD_EXPIRED", "Kartın son kullanma tarihi geçmiş", http.StatusUnprocessableEntity)
	ErrInvalidCVC        = sharedErrors.New("INVALID_CVC", "CVC 3 veya 4 haneli olmalı", http.StatusUnprocessableEntity)
	ErrInvalidCardholder = sharedErrors.New("INVALID_CARDHOLDER", "Kart üzerindeki ad gerekli", http.StatusUnprocessableEntity)

	// Repository sentinel
	ErrPaymentNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
