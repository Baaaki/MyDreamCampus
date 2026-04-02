package logger

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context) context.Context {
	requestID := uuid.New().String()
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
