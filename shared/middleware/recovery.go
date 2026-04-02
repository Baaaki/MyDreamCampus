package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery recovers from panics and logs the error with stack trace
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic with full stack trace
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", stack),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
				)

				// Return 500 error
				c.JSON(500, gin.H{
					"error":   errors.ErrInternalServer.Code,
					"message": fmt.Sprintf("Internal server error: %v", err),
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
