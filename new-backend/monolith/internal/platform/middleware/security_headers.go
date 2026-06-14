package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets standard hardening headers on every response.
// HSTS is only set when ENVIRONMENT=production, since it would otherwise
// force browsers to refuse plain HTTP during local development.
func SecurityHeaders() gin.HandlerFunc {
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if isProduction {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
