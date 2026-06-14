package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/audit"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/dto"
	authErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	requestTimeout = 10 * time.Second
)

// setAuthCookie writes an access/refresh token cookie with the strictest
// flags appropriate for production: HttpOnly, Secure (when running in
// production), and SameSite=Strict so the cookie cannot ride along on
// cross-site navigations or top-level requests.
func (h *AuthHandler) setAuthCookie(c *gin.Context, name, value string, maxAgeSeconds int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		value,
		maxAgeSeconds,
		"/api",
		"",
		h.config.Server.Environment == "production",
		true,
	)
}

// clearAuthCookie deletes a previously-set auth cookie. Flags must
// match those used when setting the cookie so the browser overwrites
// the entry rather than leaving the original.
func (h *AuthHandler) clearAuthCookie(c *gin.Context, name string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		"",
		-1,
		"/api",
		"",
		h.config.Server.Environment == "production",
		true,
	)
}

type AuthHandler struct {
	authService *service.AuthService
	config      *config.Config
}

func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      cfg,
	}
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "Login"),
		zap.String("handler", "AuthHandler"),
	)

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Error("invalid request body",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
		})
		return
	}

	// Get device info and IP
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	reqLogger.Info("login attempt",
		zap.String("email", req.Email),
		zap.String("ip", ipAddress),
	)

	// Perform login
	response, refreshToken, err := h.authService.Login(ctx, req, deviceInfo, ipAddress)

	if err != nil {
		reqLogger.Error("login failed",
			zap.Error(err),
			zap.String("email", req.Email),
		)

		// Check for specific auth errors
		if sharedErrors.Is(err, authErrors.ErrAccountLocked) {
			audit.LogSecurityFromContextWithDetails(c, audit.EventAccountLocked, "failure", "", "too many failed attempts", map[string]string{"email": req.Email})
			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Error:   "ACCOUNT_LOCKED",
				Message: "Hesabınız çok fazla başarısız giriş denemesi nedeniyle geçici olarak kilitlendi. Lütfen 30 dakika sonra tekrar deneyin.",
			})
			return
		}

		if sharedErrors.Is(err, authErrors.ErrInvalidCredentials) || err == sharedErrors.ErrUnauthorized {
			audit.LogSecurityFromContextWithDetails(c, audit.EventLoginFailed, "failure", "", "invalid credentials", map[string]string{"email": req.Email})
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "INVALID_CREDENTIALS",
				Message: "Geçersiz e-posta veya şifre",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Giriş sırasında bir hata oluştu",
		})
		return
	}

	reqLogger.Info("login successful",
		zap.String("email", req.Email),
		zap.String("role", response.User.Role),
	)

	audit.LogSecurityFromContext(c, audit.EventLogin, "success", response.User.ID)

	h.setAuthCookie(c, "access_token", response.AccessToken, h.config.JWT.AccessTokenExpiry*60)
	h.setAuthCookie(c, "refresh_token", refreshToken, h.config.JWT.RefreshTokenExpiry*3600)

	// Also return refresh token in body for non-cookie clients (mobile).
	response.RefreshToken = refreshToken
	c.JSON(http.StatusOK, response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "Logout"),
		zap.String("handler", "AuthHandler"),
	)

	// Get authenticated user ID from JWT context
	authenticatedUserID := ""
	if uid, exists := c.Get("user_id"); exists {
		authenticatedUserID = uid.(string)
	}

	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		reqLogger.Warn("logout without refresh token cookie",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "MISSING_REFRESH_TOKEN",
			Message: "Refresh token not found",
		})
		return
	}

	// Get access token from Authorization header or cookie for blacklisting
	accessToken := ""
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		accessToken = authHeader[7:]
	}
	if accessToken == "" {
		if cookie, cookieErr := c.Cookie("access_token"); cookieErr == nil {
			accessToken = cookie
		}
	}

	reqLogger.Info("logout attempt")

	// Perform logout (also blacklists the access token)
	err = h.authService.Logout(ctx, refreshToken, accessToken, authenticatedUserID)
	if err != nil {
		reqLogger.Error("logout failed",
			zap.Error(err),
		)
		// Don't fail logout even if there's an error
	}

	h.clearAuthCookie(c, "access_token")
	h.clearAuthCookie(c, "refresh_token")

	reqLogger.Info("logout successful")

	// Extract user ID if available from JWT context
	logoutUserID := ""
	if uid, exists := c.Get("user_id"); exists {
		logoutUserID = uid.(string)
	}
	audit.LogSecurityFromContext(c, audit.EventLogout, "success", logoutUserID)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Successfully logged out",
	})
}

// LogoutAll handles logout from all devices
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "LogoutAll"),
		zap.String("handler", "AuthHandler"),
	)

	// Get user ID from JWT (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		reqLogger.Warn("logout all attempted without user authentication")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		reqLogger.Error("invalid user ID format",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "Invalid user ID",
		})
		return
	}

	reqLogger = reqLogger.With(zap.String("user_id", userID.String()))
	reqLogger.Info("logout all attempt")

	// Get access token from Authorization header or cookie for blacklisting
	accessToken := ""
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		accessToken = authHeader[7:]
	}
	if accessToken == "" {
		if cookie, cookieErr := c.Cookie("access_token"); cookieErr == nil {
			accessToken = cookie
		}
	}

	// Perform logout all (also blacklists all tokens)
	err = h.authService.LogoutAll(ctx, userID, accessToken)
	if err != nil {
		reqLogger.Error("logout all failed",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to logout from all devices",
		})
		return
	}

	reqLogger.Info("logout all successful")

	audit.LogSecurityFromContext(c, audit.EventLogoutAll, "success", userID.String())

	h.clearAuthCookie(c, "access_token")
	h.clearAuthCookie(c, "refresh_token")

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Successfully logged out from all devices",
	})
}

// RefreshToken handles access token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "RefreshToken"),
		zap.String("handler", "AuthHandler"),
	)

	// Refresh token: cookie (web) or body (mobile/non-cookie clients).
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		var body dto.RefreshTokenRequest
		_ = c.ShouldBindJSON(&body) // body is optional; we already checked cookie
		refreshToken = body.RefreshToken
	}
	if refreshToken == "" {
		reqLogger.Warn("refresh token not found in cookie or body")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "MISSING_REFRESH_TOKEN",
			Message: "Refresh token not found",
		})
		return
	}

	reqLogger.Info("refresh token attempt")

	// Perform refresh
	response, newRefreshToken, err := h.authService.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		reqLogger.Error("refresh token failed",
			zap.Error(err),
		)

		if err == sharedErrors.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "INVALID_TOKEN",
				Message: "Invalid or expired refresh token",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to refresh token",
		})
		return
	}

	reqLogger.Info("refresh token successful")

	audit.LogSecurityFromContext(c, audit.EventTokenRefresh, "success", "")

	h.setAuthCookie(c, "access_token", response.AccessToken, h.config.JWT.AccessTokenExpiry*60)
	h.setAuthCookie(c, "refresh_token", newRefreshToken, h.config.JWT.RefreshTokenExpiry*3600)

	// Also return rotated refresh token in body for non-cookie clients.
	response.RefreshToken = newRefreshToken
	c.JSON(http.StatusOK, response)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuthHandler"),
		zap.String("method", "ChangePassword"),
	)

	// Get user ID from JWT
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "Invalid user ID",
		})
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
		})
		return
	}

	// Perform password change
	response, newRefreshToken, err := h.authService.ChangePassword(ctx, userID, req)
	if err != nil {
		log.Error("change password failed",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)

		if err == sharedErrors.ErrUnauthorized {
			audit.LogSecurityFromContextWithDetails(c, audit.EventPasswordChange, "failure", userID.String(), "invalid old password", nil)
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "INVALID_OLD_PASSWORD",
				Message: "Invalid old password",
			})
			return
		}

		audit.LogSecurityFromContextWithDetails(c, audit.EventPasswordChange, "failure", userID.String(), err.Error(), nil)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "PASSWORD_CHANGE_FAILED",
			Message: err.Error(),
		})
		return
	}

	audit.LogSecurityFromContext(c, audit.EventPasswordChange, "success", userID.String())

	h.setAuthCookie(c, "access_token", response.AccessToken, h.config.JWT.AccessTokenExpiry*60)
	h.setAuthCookie(c, "refresh_token", newRefreshToken, h.config.JWT.RefreshTokenExpiry*3600)

	// Also return refresh token in body for non-cookie clients.
	response.RefreshToken = newRefreshToken
	c.JSON(http.StatusOK, response)
}

// GetSessions returns all active sessions for the user
func (h *AuthHandler) GetSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuthHandler"),
		zap.String("method", "GetSessions"),
	)

	// Get user ID from JWT
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "Invalid user ID",
		})
		return
	}

	// Get current JTI from context (set by middleware)
	currentJTI, _ := c.Get("jti")
	jti := ""
	if currentJTI != nil {
		jti = currentJTI.(string)
	}

	// Get sessions
	response, err := h.authService.GetUserSessions(ctx, userID, jti)
	if err != nil {
		log.Error("get sessions failed",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to retrieve sessions",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Verify validates the JWT token and returns user info for Traefik forward auth
// This endpoint is called by Traefik before forwarding requests to protected services
func (h *AuthHandler) Verify(c *gin.Context) {
	// Get user claims from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Block downstream service access until password is changed
	if fpc, ok := c.Get("force_password_change"); ok {
		if mustChange, _ := fpc.(bool); mustChange {
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Error:   "FORCE_PASSWORD_CHANGE",
				Message: "Şifrenizi değiştirmeden diğer servislere erişemezsiniz",
			})
			return
		}
	}

	role, _ := c.Get("role")
	department, _ := c.Get("department")

	// Set headers for downstream services (Traefik forwards these)
	c.Header("X-User-ID", userID.(string))
	c.Header("X-User-Role", role.(string))
	if dept, ok := department.(string); ok && dept != "" {
		c.Header("X-User-Department", dept)
	}

	c.Status(http.StatusOK)
}

// DeleteSession deletes a specific session
func (h *AuthHandler) DeleteSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuthHandler"),
		zap.String("method", "DeleteSession"),
	)

	// Get user ID from JWT
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "INVALID_USER_ID",
			Message: "Invalid user ID",
		})
		return
	}

	// Get session ID from URL param
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "INVALID_SESSION_ID",
			Message: "Invalid session ID",
		})
		return
	}

	// Get current JTI from context
	currentJTI, _ := c.Get("jti")
	jti := ""
	if currentJTI != nil {
		jti = currentJTI.(string)
	}

	// Delete session
	err = h.authService.DeleteSession(ctx, sessionID, userID, jti)
	if err != nil {
		log.Error("delete session failed",
			zap.Error(err),
			zap.String("session_id", sessionID.String()),
		)

		if err.Error() == "cannot terminate current session, use logout instead" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "CANNOT_TERMINATE_CURRENT_SESSION",
				Message: "Aktif oturumunuzu sonlandırmak için logout endpoint'ini kullanın",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to delete session",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Session terminated successfully",
	})
}

// RequestPasswordReset handles password reset request
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuthHandler"),
		zap.String("method", "RequestPasswordReset"),
	)

	var req dto.RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
		})
		return
	}

	err := h.authService.RequestPasswordReset(ctx, req.Email)
	if err != nil {
		log.Error("request password reset failed",
			zap.Error(err),
			zap.String("email", req.Email),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to request password reset",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "If an account with that email exists, a password reset link has been sent.",
	})
}
