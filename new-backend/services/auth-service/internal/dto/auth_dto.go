package dto

import "time"

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse represents the login response payload. RefreshToken is set
// only for clients that sent X-Client-Type: mobile; browsers get it as an
// HttpOnly cookie and never in a body script could read.
type LoginResponse struct {
	AccessToken         string       `json:"access_token"`
	RefreshToken        string       `json:"refresh_token,omitempty"`
	ExpiresIn           int          `json:"expires_in"` // seconds
	User                UserResponse `json:"user"`
	ForcePasswordChange bool         `json:"force_password_change"`
	Message             string       `json:"message,omitempty"`
}

// RefreshTokenRequest represents an optional body for /auth/refresh.
// Mobile clients send the refresh token here; web clients leave it
// empty and the handler reads it from the HttpOnly cookie.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents the refresh token response. RefreshToken is set
// only when the old one arrived in the body.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// ChangePasswordRequest represents the change password request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=8"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// RequestPasswordResetRequest represents the password reset request
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest completes a password reset with the e-mailed token.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// ChangePasswordResponse represents the change password response
type ChangePasswordResponse struct {
	Message      string `json:"message"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// UserResponse represents the user data in responses
type UserResponse struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	Department *string `json:"department,omitempty"`
}

// SessionResponse represents a user session
type SessionResponse struct {
	ID         string    `json:"id"`
	DeviceInfo *string   `json:"device_info,omitempty"`
	IPAddress  *string   `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsCurrent  bool      `json:"is_current"`
}

// SessionsResponse represents the list of user sessions
type SessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
