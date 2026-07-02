package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequireInternalSecret guards /internal/* routes that exist for
// service-to-service (loopback) calls. Callers must send the shared
// secret in X-Internal-Secret; requests without it are rejected before
// reaching the handler. The constructor panics on an empty secret so a
// missing config surfaces at startup, not as an open internal route.
func RequireInternalSecret(secret string) gin.HandlerFunc {
	if secret == "" {
		panic("internal secret must not be empty - set INTERNAL_SERVICE_SECRET")
	}

	return func(c *gin.Context) {
		received := c.GetHeader("X-Internal-Secret")
		// Constant-time compare: the secret is long-lived, so don't leak
		// prefix-match timing to a brute-force caller.
		if subtle.ConstantTimeCompare([]byte(received), []byte(secret)) != 1 {
			logger.Warn("internal route called without valid secret",
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "UNAUTHORIZED",
				"message": "Authentication required",
			})
			return
		}

		c.Next()
	}
}
