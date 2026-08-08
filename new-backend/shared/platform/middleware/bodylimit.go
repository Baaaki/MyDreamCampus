package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DefaultMaxBodyBytes bounds request bodies; the largest legitimate
// payload (monthly menu upload) stays well under 1 MB.
const DefaultMaxBodyBytes int64 = 1 << 20

// BodySizeLimit caps the request body size. Without a cap, Gin reads
// arbitrarily large bodies into JSON binding — and login bodies feed
// Argon2, so oversized payloads amplify CPU/memory cost. Reads beyond
// the limit fail inside ShouldBindJSON and surface as validation errors.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
