package errors

import "net/http"

// ============================================================================
// COMMON HTTP ERRORS
// These errors are used across ALL services for standard HTTP responses
// Service-specific errors should be defined in service/internal/errors/
// ============================================================================

var (
	// 4xx Client Errors
	ErrBadRequest   = New("BAD_REQUEST", "Geçersiz istek", http.StatusBadRequest)
	ErrUnauthorized = New("UNAUTHORIZED", "Oturum açmanız gerekiyor", http.StatusUnauthorized)
	ErrForbidden    = New("FORBIDDEN", "Bu işlem için yetkiniz yok", http.StatusForbidden)
	ErrNotFound     = New("NOT_FOUND", "Kayıt bulunamadı", http.StatusNotFound)
	ErrConflict     = New("CONFLICT", "İşlem mevcut bir kayıtla çakışıyor", http.StatusConflict)
	ErrValidation   = New("VALIDATION_ERROR", "Gönderilen bilgiler geçersiz", http.StatusBadRequest)
	ErrInvalidID    = New("INVALID_ID", "Geçersiz kimlik formatı", http.StatusBadRequest)
	ErrTooManyReqs  = New("TOO_MANY_REQUESTS", "Çok fazla istek, lütfen biraz sonra tekrar deneyin", http.StatusTooManyRequests)

	// 5xx Server Errors
	ErrInternal           = New("INTERNAL_ERROR", "Beklenmeyen bir hata oluştu, lütfen tekrar deneyin", http.StatusInternalServerError)
	ErrServiceUnavailable = New("SERVICE_UNAVAILABLE", "Hizmet şu anda kullanılamıyor, lütfen birazdan tekrar deneyin", http.StatusServiceUnavailable)
	ErrNotImplemented     = New("NOT_IMPLEMENTED", "Bu özellik henüz desteklenmiyor", http.StatusNotImplemented)

	// Deprecated aliases (kept for backward compatibility, will be removed in future)
	ErrInternalServer = ErrInternal // Use ErrInternal instead
)
