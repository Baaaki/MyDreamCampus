package errors

import (
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// ============================================================================
// AUTH SERVICE SPECIFIC ERRORS
// These errors are specific to authentication and authorization domain
// They should NOT be moved to shared/errors as they represent business logic
// unique to the auth service
// ============================================================================

// Messages are shown to the user as-is, so they are Turkish; codes stay
// English and are what clients branch on.
var (
	// Authentication errors
	ErrInvalidCredentials = sharedErrors.New("INVALID_CREDENTIALS", "Geçersiz e-posta veya şifre", http.StatusUnauthorized)
	ErrWeakPassword       = sharedErrors.New("WEAK_PASSWORD", "Şifre en az 8 karakter olmalı; büyük harf, küçük harf ve rakam içermeli", http.StatusBadRequest)
	// 400, not 401: the caller IS authenticated. A 401 here makes both
	// clients refresh the session and replay the request for nothing.
	ErrInvalidOldPassword = sharedErrors.New("INVALID_OLD_PASSWORD", "Mevcut şifre hatalı", http.StatusBadRequest)

	// Token errors
	ErrInvalidToken         = sharedErrors.New("INVALID_TOKEN", "Oturum geçersiz, lütfen tekrar giriş yapın", http.StatusUnauthorized)
	ErrExpiredToken         = sharedErrors.New("EXPIRED_TOKEN", "Oturumun süresi doldu, lütfen tekrar giriş yapın", http.StatusUnauthorized)
	ErrTokenRevoked         = sharedErrors.New("TOKEN_REVOKED", "Oturum sonlandırıldı, lütfen tekrar giriş yapın", http.StatusUnauthorized)
	ErrTokenVersionMismatch = sharedErrors.New("TOKEN_VERSION_MISMATCH", "Oturum sonlandırıldı, lütfen tekrar giriş yapın", http.StatusUnauthorized)
	ErrInvalidResetToken    = sharedErrors.New("INVALID_RESET_TOKEN", "Şifre sıfırlama bağlantısı geçersiz veya süresi dolmuş", http.StatusBadRequest)

	// Account status errors
	// Never sent to the client as-is: the handler answers a lockout exactly
	// like INVALID_CREDENTIALS so it cannot be used to confirm an address.
	ErrAccountLocked       = sharedErrors.New("ACCOUNT_LOCKED", "Çok fazla başarısız giriş denemesi", http.StatusUnauthorized)
	ErrAccountDeactivated  = sharedErrors.New("ACCOUNT_DEACTIVATED", "Hesap devre dışı", http.StatusUnauthorized)
	ErrForcePasswordChange = sharedErrors.New("FORCE_PASSWORD_CHANGE", "Devam etmek için şifrenizi değiştirmeniz gerekiyor", http.StatusForbidden)

	// Session errors
	ErrCannotTerminateSession = sharedErrors.New("CANNOT_TERMINATE_CURRENT_SESSION", "Aktif oturumunuzu sonlandırmak için çıkış yapın", http.StatusBadRequest)

	// Rate limiting
	ErrRateLimitExceeded = sharedErrors.New("RATE_LIMIT_EXCEEDED", "Çok fazla istek, lütfen biraz sonra tekrar deneyin", http.StatusTooManyRequests)

	// User errors
	ErrUserNotFound = sharedErrors.New("USER_NOT_FOUND", "Kullanıcı bulunamadı", http.StatusNotFound)
	ErrUserExists   = sharedErrors.New("USER_EXISTS", "Kullanıcı zaten mevcut", http.StatusConflict)
	ErrEmailExists  = sharedErrors.New("EMAIL_EXISTS", "Bu e-posta adresi zaten kullanılıyor", http.StatusConflict)

	// Session errors
	ErrSessionNotFound = sharedErrors.New("SESSION_NOT_FOUND", "Oturum bulunamadı", http.StatusNotFound)

	// Repository-specific sentinel errors (for internal use)
	ErrUserNotFoundRepo    = sharedErrors.ErrNotFoundRepo
	ErrUserExistsRepo      = sharedErrors.ErrAlreadyExistsRepo
	ErrSessionNotFoundRepo = sharedErrors.ErrNotFoundRepo
)
