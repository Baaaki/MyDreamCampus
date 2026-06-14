package middleware

import (
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger logs all HTTP requests with duration and status code.
// Honors an incoming X-Request-ID so a single ID flows across the API
// gateway, services, and downstream event consumers; otherwise mints
// a fresh UUID for the trace.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Reuse upstream X-Request-ID when present so a single trace
		// ID survives across Traefik → service → downstream calls.
		incoming := c.GetHeader("X-Request-ID")
		ctx := logger.WithRequestIDValue(c.Request.Context(), incoming)
		c.Request = c.Request.WithContext(ctx)
		requestID := logger.GetRequestID(ctx)

		// Set request ID in response header for debugging
		c.Writer.Header().Set("X-Request-ID", requestID)

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get status code and any errors
		statusCode := c.Writer.Status()

		// Log request details
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Add user context if authenticated
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.String("user_id", userID.(string)))
		}
		if role, exists := c.Get("role"); exists {
			fields = append(fields, zap.String("role", role.(string)))
		}

		// Log with appropriate level based on status code
		switch {
		case statusCode >= 500:
			logger.Error("http request - server error", fields...)
		case statusCode >= 400:
			logger.Warn("http request - client error", fields...)
		default:
			logger.Info("http request", fields...)
		}
	}
}
