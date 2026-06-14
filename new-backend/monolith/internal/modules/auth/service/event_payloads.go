package service

import "time"

// buildUserRegisteredPayload builds the payload for auth.user.registered event.
func buildUserRegisteredPayload(id, email, firstName, lastName, role string) map[string]any {
	return map[string]any{
		"id":         id,
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"role":       role,
	}
}

// buildUserPasswordResetRequestedPayload builds the payload for auth.user.password_reset_requested event.
func buildUserPasswordResetRequestedPayload(userID, email, resetToken string, expiresAt time.Time) map[string]any {
	return map[string]any{
		"user_id":     userID,
		"email":       email,
		"reset_token": resetToken,
		"expires_at":  expiresAt.Format(time.RFC3339),
	}
}
