package middleware

import (
	"runtime/debug"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery recovers from panics, logs the panic with full stack trace,
// and returns a generic 500 to the client. The panic value and stack
// stay in the structured log only — the response body must never echo
// internal error text or paths back to the caller.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := logger.GetRequestID(c.Request.Context())

				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.String("request_id", requestID),
				)

				c.JSON(500, gin.H{
					"error":      errors.ErrInternalServer.Code,
					"message":    "Internal server error",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
