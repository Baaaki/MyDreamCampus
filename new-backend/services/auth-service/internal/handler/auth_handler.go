package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	authErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	requestTimeout = 10 * time.Second

	accessCookie  = "access_token"
	refreshCookie = "refresh_token"
	// Every service validates the access token, so its cookie has to reach
	// all of /api. Only auth ever reads the refresh token; scoping it to
	// /api/auth keeps it off the nine other services' requests and logs.
	accessCookiePath  = "/api"
	refreshCookiePath = "/api/auth"

	// ClientTypeHeader marks a client without a cookie jar (the mobile app).
	// Only those get the refresh token in a response body: there it is
	// readable by any script on the page, which is exactly what the HttpOnly
	// cookie exists to prevent for browsers.
	ClientTypeHeader = "X-Client-Type"
	clientTypeMobile = "mobile"
)

// authService is what the handler needs from service.AuthService; an
// interface so the HTTP mapping can be tested without Postgres and Redis.
type authService interface {
	Login(ctx context.Context, req dto.LoginRequest, deviceInfo, ipAddress string) (dto.LoginResponse, string, error)
	Logout(ctx context.Context, refreshToken, accessToken, authenticatedUserID string) error
	LogoutAll(ctx context.Context, userID uuid.UUID, accessToken string) error
	RefreshAccessToken(ctx context.Context, refreshToken string) (dto.RefreshResponse, string, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) (dto.ChangePasswordResponse, string, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	GetUserSessions(ctx context.Context, userID uuid.UUID, currentJTI string) (dto.SessionsResponse, error)
	DeleteSession(ctx context.Context, sessionID, userID uuid.UUID, currentJTI string) error
}

type AuthHandler struct {
	authService authService
	config      *config.Config
}

func NewAuthHandler(authService authService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      cfg,
	}
}

// setAuthCookie writes an auth cookie HttpOnly, Secure in production and
// SameSite=Strict so it never rides along on a cross-site request.
func (h *AuthHandler) setAuthCookie(c *gin.Context, name, value, path string, maxAgeSeconds int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(name, value, maxAgeSeconds, path, "", h.config.Server.Environment == "production", true)
}

// clearAuthCookie deletes an auth cookie. Name, path and flags must match the
// ones it was set with, or the browser keeps the original.
func (h *AuthHandler) clearAuthCookie(c *gin.Context, name, path string) {
	h.setAuthCookie(c, name, "", path, -1)
}

// issueSession hands a new token pair to the client: cookies for browsers,
// the refresh token in the body for mobile. It returns the value to put in
// the response's refresh_token field ("" means omitted).
func (h *AuthHandler) issueSession(c *gin.Context, accessToken, refreshToken string, refreshInBody bool) string {
	if refreshInBody {
		return refreshToken
	}
	h.setAuthCookie(c, accessCookie, accessToken, accessCookiePath, h.config.JWT.AccessTokenExpiry*60)
	h.setAuthCookie(c, refreshCookie, refreshToken, refreshCookiePath, h.config.JWT.RefreshTokenExpiry*3600)
	// Refresh cookies used to be scoped to /api. Left alone, that copy would
	// keep being sent next to the new one until it expires.
	h.clearAuthCookie(c, refreshCookie, accessCookiePath)
	return ""
}

func (h *AuthHandler) endSession(c *gin.Context) {
	h.clearAuthCookie(c, accessCookie, accessCookiePath)
	h.clearAuthCookie(c, refreshCookie, refreshCookiePath)
	h.clearAuthCookie(c, refreshCookie, accessCookiePath)
}

func isMobileClient(c *gin.Context) bool {
	return c.GetHeader(ClientTypeHeader) == clientTypeMobile
}

func bearerOrCookieAccessToken(c *gin.Context) string {
	if token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok {
		return token
	}
	if cookie, err := c.Cookie(accessCookie); err == nil {
		return cookie
	}
	return ""
}

// bodyRefreshToken reads the optional {"refresh_token": ...} body mobile sends.
func bodyRefreshToken(c *gin.Context) string {
	var body dto.RefreshTokenRequest
	_ = c.ShouldBindJSON(&body) // the body is optional; absent means cookie client
	return body.RefreshToken
}

func respondInvalidCredentials(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
		Error:   authErrors.ErrInvalidCredentials.Code,
		Message: authErrors.ErrInvalidCredentials.Message,
	})
}

func respondValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, dto.ErrorResponse{
		Error:   sharedErrors.ErrValidation.Code,
		Message: message,
	})
}

// respondError answers with the AppError's own status, code and (Turkish)
// message; anything else is an internal error whose details stay in the log.
func respondError(c *gin.Context, err error) {
	if appErr, ok := sharedErrors.As(err); ok && appErr.HTTPStatus < http.StatusInternalServerError {
		c.JSON(appErr.HTTPStatus, dto.ErrorResponse{Error: appErr.Code, Message: appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Error:   sharedErrors.ErrInternal.Code,
		Message: "Beklenmeyen bir hata oluştu, lütfen tekrar deneyin",
	})
}

// authenticatedUserID reads the user JWTAuth put on the context.
func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   sharedErrors.ErrUnauthorized.Code,
			Message: "Oturum açmanız gerekiyor",
		})
		return uuid.Nil, false
	}
	return id, true
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "Login"),
		zap.String("handler", "AuthHandler"),
	)

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Warn("invalid request body", zap.Error(err))
		respondValidationError(c, "Geçerli bir e-posta adresi ve şifre girin")
		return
	}

	ipAddress := c.ClientIP()
	reqLogger.Info("login attempt", zap.String("email", req.Email), zap.String("ip", ipAddress))

	response, refreshToken, err := h.authService.Login(ctx, req, c.GetHeader("User-Agent"), ipAddress)
	if err != nil {
		reqLogger.Warn("login failed", zap.Error(err), zap.String("email", req.Email))

		// A lockout answers exactly like a wrong password. A separate status
		// would confirm the address exists — undoing the dummy-hash timing
		// protection — so the real reason goes to the audit log only.
		if sharedErrors.Is(err, authErrors.ErrAccountLocked) {
			audit.LogSecurityFromContextWithDetails(c, audit.EventAccountLocked, "failure", "", "locked out after repeated failures", map[string]string{"email": req.Email})
			respondInvalidCredentials(c)
			return
		}
		if sharedErrors.Is(err, authErrors.ErrInvalidCredentials) {
			audit.LogSecurityFromContextWithDetails(c, audit.EventLoginFailed, "failure", "", "invalid credentials", map[string]string{"email": req.Email})
			respondInvalidCredentials(c)
			return
		}
		respondError(c, err)
		return
	}

	reqLogger.Info("login successful", zap.String("email", req.Email), zap.String("role", response.User.Role))
	audit.LogSecurityFromContext(c, audit.EventLogin, "success", response.User.ID)

	response.RefreshToken = h.issueSession(c, response.AccessToken, refreshToken, isMobileClient(c))
	c.JSON(http.StatusOK, response)
}

// Logout ends the caller's session. It always answers 200 and clears the
// cookies: a client asking to log out must end up logged out locally even
// when the server-side cleanup had nothing to do.
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "Logout"),
		zap.String("handler", "AuthHandler"),
	)

	userID := c.GetString("user_id")

	// Mobile sends the refresh token in the body, browsers in the cookie.
	refreshToken := bodyRefreshToken(c)
	if refreshToken == "" {
		refreshToken, _ = c.Cookie(refreshCookie)
	}

	if err := h.authService.Logout(ctx, refreshToken, bearerOrCookieAccessToken(c), userID); err != nil {
		reqLogger.Warn("logout cleanup incomplete", zap.Error(err))
	}

	h.endSession(c)
	audit.LogSecurityFromContext(c, audit.EventLogout, "success", userID)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Çıkış yapıldı"})
}

// LogoutAll handles logout from all devices
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "LogoutAll"),
		zap.String("handler", "AuthHandler"),
		zap.String("user_id", userID.String()),
	)

	if err := h.authService.LogoutAll(ctx, userID, bearerOrCookieAccessToken(c)); err != nil {
		reqLogger.Error("logout all failed", zap.Error(err))
		respondError(c, err)
		return
	}

	audit.LogSecurityFromContext(c, audit.EventLogoutAll, "success", userID.String())
	h.endSession(c)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Tüm cihazlardan çıkış yapıldı"})
}

// RefreshToken rotates the token pair. A body token (mobile) wins over the
// cookie: a native HTTP stack may keep a cookie jar of its own, and a mobile
// client that got its new refresh token only as a cookie could never store it.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "RefreshToken"),
		zap.String("handler", "AuthHandler"),
	)

	refreshToken := bodyRefreshToken(c)
	// The new refresh token goes back the way the old one came: a token the
	// client already held in the body may be returned in the body, one that
	// arrived as an HttpOnly cookie never is.
	fromBody := refreshToken != ""
	if !fromBody {
		refreshToken, _ = c.Cookie(refreshCookie)
	}
	if refreshToken == "" {
		reqLogger.Warn("refresh token not found in body or cookie")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "MISSING_REFRESH_TOKEN",
			Message: "Oturum bulunamadı, lütfen tekrar giriş yapın",
		})
		return
	}

	response, newRefreshToken, err := h.authService.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		if isSessionEnded(err) {
			reqLogger.Warn("refresh rejected", zap.Error(err))
			// The cookie is dead; clearing it stops the browser from
			// replaying it on every page load. Not on SESSION_NOT_FOUND:
			// the losing tab of two refreshing at once gets it, and its
			// Set-Cookie would wipe the cookies the winner has just set.
			if !sharedErrors.Is(err, authErrors.ErrSessionNotFound) {
				h.endSession(c)
			}
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   authErrors.ErrInvalidToken.Code,
				Message: authErrors.ErrInvalidToken.Message,
			})
			return
		}
		reqLogger.Error("refresh failed", zap.Error(err))
		respondError(c, err)
		return
	}

	audit.LogSecurityFromContext(c, audit.EventTokenRefresh, "success", "")

	response.RefreshToken = h.issueSession(c, response.AccessToken, newRefreshToken, fromBody)
	c.JSON(http.StatusOK, response)
}

// isSessionEnded reports the refresh failures that mean "log in again" — as
// opposed to the server failing, which must stay a 5xx so clients retry.
func isSessionEnded(err error) bool {
	for _, target := range []error{
		authErrors.ErrInvalidToken,
		authErrors.ErrSessionNotFound,
		authErrors.ErrUserNotFound,
		authErrors.ErrTokenVersionMismatch,
	} {
		if sharedErrors.Is(err, target) {
			return true
		}
	}
	return false
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuthHandler"),
		zap.String("method", "ChangePassword"),
		zap.String("user_id", userID.String()),
	)

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "Mevcut ve yeni şifre alanları zorunludur (en az 8 karakter)")
		return
	}

	response, newRefreshToken, err := h.authService.ChangePassword(ctx, userID, req)
	if err != nil {
		reqLogger.Warn("change password failed", zap.Error(err))
		reason := "internal error"
		if appErr, ok := sharedErrors.As(err); ok {
			reason = appErr.Code
		}
		audit.LogSecurityFromContextWithDetails(c, audit.EventPasswordChange, "failure", userID.String(), reason, nil)
		respondError(c, err)
		return
	}

	audit.LogSecurityFromContext(c, audit.EventPasswordChange, "success", userID.String())

	response.RefreshToken = h.issueSession(c, response.AccessToken, newRefreshToken, isMobileClient(c))
	c.JSON(http.StatusOK, response)
}

// GetSessions returns all active sessions for the user
func (h *AuthHandler) GetSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	response, err := h.authService.GetUserSessions(ctx, userID, c.GetString("jti"))
	if err != nil {
		logger.WithContextAndFields(ctx, zap.String("handler", "AuthHandler"), zap.String("method", "GetSessions")).
			Error("get sessions failed", zap.Error(err), zap.String("user_id", userID.String()))
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSession deletes a specific session
func (h *AuthHandler) DeleteSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondValidationError(c, "Geçersiz oturum kimliği")
		return
	}

	if err := h.authService.DeleteSession(ctx, sessionID, userID, c.GetString("jti")); err != nil {
		logger.WithContextAndFields(ctx, zap.String("handler", "AuthHandler"), zap.String("method", "DeleteSession")).
			Warn("delete session failed", zap.Error(err), zap.String("session_id", sessionID.String()))
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Oturum sonlandırıldı"})
}

// RequestPasswordReset handles password reset request. The answer is the
// same whether or not the address exists.
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "Geçerli bir e-posta adresi girin")
		return
	}

	if err := h.authService.RequestPasswordReset(ctx, req.Email); err != nil {
		logger.WithContextAndFields(ctx, zap.String("handler", "AuthHandler"), zap.String("method", "RequestPasswordReset")).
			Error("request password reset failed", zap.Error(err), zap.String("email", req.Email))
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Bu adrese kayıtlı bir hesap varsa şifre sıfırlama bağlantısı gönderildi.",
	})
}

// ResetPassword sets a new password with the token from the reset e-mail.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "Sıfırlama bağlantısı ve yeni şifre zorunludur")
		return
	}

	if err := h.authService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		logger.WithContextAndFields(ctx, zap.String("handler", "AuthHandler"), zap.String("method", "ResetPassword")).
			Warn("password reset failed", zap.Error(err))
		audit.LogSecurityFromContextWithDetails(c, audit.EventPasswordChange, "failure", "", "password reset rejected", nil)
		respondError(c, err)
		return
	}

	audit.LogSecurityFromContextWithDetails(c, audit.EventPasswordChange, "success", "", "password reset", nil)
	h.endSession(c)
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Şifreniz güncellendi, yeni şifrenizle giriş yapabilirsiniz"})
}
