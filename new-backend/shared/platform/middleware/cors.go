package middleware

import (
	"os"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// devDefaultOrigins is used only when CORS_ALLOWED_ORIGINS is unset and
// ENVIRONMENT != "production".
var devDefaultOrigins = []string{
	"http://localhost:3000",
	"http://localhost:3002",
}

// resolveAllowedOrigins reads CORS_ALLOWED_ORIGINS (comma-separated). In
// production the env var must be set; otherwise the constructor panics so
// the misconfiguration surfaces at startup rather than as a silent reflect-all.
func resolveAllowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw != "" {
		parts := strings.Split(raw, ",")
		origins := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
		return origins
	}

	if os.Getenv("ENVIRONMENT") == "production" {
		panic("CORS_ALLOWED_ORIGINS environment variable must be set in production")
	}
	return devDefaultOrigins
}

// CORS handles Cross-Origin Resource Sharing headers.
// Dynamically checks the Origin header against an allowed list
// and reflects the specific origin instead of using a wildcard,
// so Access-Control-Allow-Credentials can be safely set to true.
func CORS() gin.HandlerFunc {
	allowedOrigins := resolveAllowedOrigins()

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if slices.Contains(allowedOrigins, origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-CSRF-Token")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// CORSWithOrigins allows CORS with specific origins
func CORSWithOrigins(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if slices.Contains(allowedOrigins, origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-CSRF-Token")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// CORSForMobile allows CORS for web and mobile apps.
// In production, only origins in CORS_ALLOWED_ORIGINS are allowed.
// In development, localhost and Expo origins are also permitted.
func CORSForMobile() gin.HandlerFunc {
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	allowedOrigins := resolveAllowedOrigins()

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin == "" {
			// Mobile apps (native HTTP clients) don't send an Origin header.
			// No Access-Control-Allow-Origin is needed; the browser isn't involved.
			// Just set common headers and continue.
			setCORSCommonHeaders(c)
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
			return
		}

		allowed := false
		if isProduction {
			allowed = slices.Contains(allowedOrigins, origin)
		} else {
			allowed = strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "exp://") ||
				slices.Contains(allowedOrigins, origin)
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		setCORSCommonHeaders(c)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// setCORSCommonHeaders writes the shared CORS headers that don't depend on origin validation.
func setCORSCommonHeaders(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-CSRF-Token")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	c.Writer.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
}
