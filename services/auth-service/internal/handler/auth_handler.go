package handler

import (
	"context"
	"net/http"
	"time"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	"github.com/baaaki/mydreamcampus/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	requestTimeout = 10 * time.Second
)

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

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "Login"),
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

	// Perform login
	response, err := h.authService.Login(ctx, req, deviceInfo, ipAddress)
	if err != nil {
		logger.Error("login failed",
			zap.Error(err),
			zap.String("email", req.Email),
		)

		if err == sharedErrors.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "INVALID_CREDENTIALS",
				Message: "Invalid email or password",
			})
			return
		}

		if err.Error() == "account deactivated" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "ACCOUNT_DEACTIVATED",
				Message: "Your account has been deactivated",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "An error occurred during login",
		})
		return
	}

	// Set refresh token as HttpOnly cookie
	// Note: refreshToken is not in response, it's set via Cookie header
	// We need to get it from the login response or generate it separately
	// For now, we'll skip cookie setting in this simplified version

	c.JSON(http.StatusOK, response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		logger.Warn("logout without refresh token cookie",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "MISSING_REFRESH_TOKEN",
			Message: "Refresh token not found",
		})
		return
	}

	// Perform logout
	err = h.authService.Logout(ctx, refreshToken)
	if err != nil {
		logger.Error("logout failed",
			zap.Error(err),
		)
		// Don't fail logout even if there's an error
	}

	// Clear refresh token cookie
	c.SetCookie(
		"refresh_token",
		"",
		-1, // MaxAge -1 deletes the cookie
		"/api/v1/auth",
		"",
		h.config.Server.Environment == "production",
		true, // HttpOnly
	)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Successfully logged out",
	})
}

// LogoutAll handles logout from all devices
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get user ID from JWT (set by auth middleware)
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

	// Perform logout all
	err = h.authService.LogoutAll(ctx, userID)
	if err != nil {
		logger.Error("logout all failed",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to logout from all devices",
		})
		return
	}

	// Clear current refresh token cookie
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/auth",
		"",
		h.config.Server.Environment == "production",
		true,
	)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Successfully logged out from all devices",
	})
}

// RefreshToken handles access token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "MISSING_REFRESH_TOKEN",
			Message: "Refresh token not found",
		})
		return
	}

	// Perform refresh
	response, newRefreshToken, err := h.authService.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		logger.Error("refresh token failed",
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

	// Set new refresh token cookie
	maxAge := h.config.JWT.RefreshTokenExpiry * 3600 // convert hours to seconds
	c.SetCookie(
		"refresh_token",
		newRefreshToken,
		maxAge,
		"/api/v1/auth",
		"",
		h.config.Server.Environment == "production",
		true, // HttpOnly
	)

	c.JSON(http.StatusOK, response)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

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
		logger.Error("change password failed",
			zap.Error(err),
			zap.String("user_id", userID.String()),
		)

		if err == sharedErrors.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "INVALID_OLD_PASSWORD",
				Message: "Invalid old password",
			})
			return
		}

		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "PASSWORD_CHANGE_FAILED",
			Message: err.Error(),
		})
		return
	}

	// Set new refresh token cookie
	maxAge := h.config.JWT.RefreshTokenExpiry * 3600
	c.SetCookie(
		"refresh_token",
		newRefreshToken,
		maxAge,
		"/api/v1/auth",
		"",
		h.config.Server.Environment == "production",
		true,
	)

	c.JSON(http.StatusOK, response)
}

// GetSessions returns all active sessions for the user
func (h *AuthHandler) GetSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

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
		logger.Error("get sessions failed",
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

// DeleteSession deletes a specific session
func (h *AuthHandler) DeleteSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

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
		logger.Error("delete session failed",
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
