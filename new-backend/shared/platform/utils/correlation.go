package utils

import (
	"context"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// CorrelationIDFromContext turns the request ID carried in ctx into an
// outbox correlation_id, so "which events did this request cause" is a SQL
// question. Returns NULL when there is no request behind the write — worker
// and scheduler events — or when the id is not a UUID.
func CorrelationIDFromContext(ctx context.Context) pgtype.UUID {
	requestID := logger.GetRequestID(ctx)
	if requestID == "" {
		return pgtype.UUID{}
	}
	parsed, err := uuid.Parse(requestID)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

// CorrelationIDString renders a stored correlation_id for the event
// envelope. NULL becomes "" rather than the all-zero UUID, so the envelope
// can omit the field instead of shipping a meaningless id.
func CorrelationIDString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
