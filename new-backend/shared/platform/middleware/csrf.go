package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CSRFProtection implements double-submit cookie pattern for CSRF prevention.
// It sets a non-httpOnly CSRF token cookie that the frontend reads and sends
// back as a header. Since an attacker cannot read cross-origin cookies,
// they cannot forge the header value.
func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods (GET, HEAD, OPTIONS)
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip CSRF if request uses Authorization header (API/mobile clients)
		if c.GetHeader("Authorization") != "" {
			c.Next()
			return
		}

		// For cookie-based auth, validate CSRF token
		csrfCookie, err := c.Cookie("csrf_token")
		if err != nil || csrfCookie == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "CSRF_ERROR",
				"message": "CSRF token missing",
			})
			return
		}

		csrfHeader := c.GetHeader("X-CSRF-Token")
		if csrfHeader == "" || csrfHeader != csrfCookie {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "CSRF_ERROR",
				"message": "CSRF token mismatch",
			})
			return
		}

		c.Next()
	}
}

// SetCSRFToken generates and sets a CSRF token cookie if one doesn't exist.
// This should be applied to all routes so the token is available for state-changing requests.
func SetCSRFToken(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Cookie("csrf_token"); err != nil {
			token := generateCSRFToken()
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("csrf_token", token, 86400, "/", "", isProduction, false) // NOT httpOnly so JS can read it
		}
		c.Next()
	}
}

func generateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
