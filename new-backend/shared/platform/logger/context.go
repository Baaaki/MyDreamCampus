package logger

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID adds a freshly generated request ID to the context.
// Use WithRequestIDValue to attach an inbound ID propagated from an
// upstream service.
func WithRequestID(ctx context.Context) context.Context {
	requestID := uuid.New().String()
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithRequestIDValue attaches an explicit request ID to the context.
// If the supplied value is empty, a fresh UUID is generated so callers
// can pass an incoming X-Request-ID header through unconditionally.
func WithRequestIDValue(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithContext returns a logger with request ID field from context
func WithContext(ctx context.Context) *zap.Logger {
	if requestID := GetRequestID(ctx); requestID != "" {
		return Log.With(zap.String("request_id", requestID))
	}
	return Log
}

// WithFields creates a child logger with the given fields
// Example: logger.WithFields(zap.String("service", "StaffService"), zap.String("method", "CreateStaff"))
func WithFields(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}

// WithContextAndFields creates a child logger with request ID from context and additional fields
// Example: logger.WithContextAndFields(ctx, zap.String("user_id", "123"), zap.String("action", "create"))
func WithContextAndFields(ctx context.Context, fields ...zap.Field) *zap.Logger {
	baseLogger := WithContext(ctx)
	return baseLogger.With(fields...)
}
