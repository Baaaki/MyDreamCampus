package utils

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCorrelationIDFromContext_RequestIDBecomesCorrelationID(t *testing.T) {
	id := uuid.New()
	ctx := logger.WithRequestIDValue(context.Background(), id.String())

	got := CorrelationIDFromContext(ctx)

	assert.True(t, got.Valid)
	assert.Equal(t, id, uuid.UUID(got.Bytes))
}

// Worker- and scheduler-driven events have no request behind them; the
// column stays NULL rather than picking up a fabricated id.
func TestCorrelationIDFromContext_NoRequestIDIsNull(t *testing.T) {
	assert.False(t, CorrelationIDFromContext(context.Background()).Valid)
}

func TestCorrelationIDFromContext_NonUUIDRequestIDIsNull(t *testing.T) {
	ctx := logger.WithRequestIDValue(context.Background(), "not-a-uuid")
	assert.False(t, CorrelationIDFromContext(ctx).Valid)
}

func TestCorrelationIDString_NullBecomesEmpty(t *testing.T) {
	assert.Empty(t, CorrelationIDString(CorrelationIDFromContext(context.Background())),
		"an all-zero UUID in the envelope would look like a real correlation id")
}

func TestCorrelationIDString_ValidRoundTrips(t *testing.T) {
	id := uuid.New()
	ctx := logger.WithRequestIDValue(context.Background(), id.String())

	assert.Equal(t, id.String(), CorrelationIDString(CorrelationIDFromContext(ctx)))
}
