package logger

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRequestID_GeneratesUUID(t *testing.T) {
	ctx := WithRequestID(context.Background())
	id := GetRequestID(ctx)
	require.NotEmpty(t, id)

	parsed, err := uuid.Parse(id)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, parsed)
}

func TestWithRequestIDValue_KeepsExplicitValue(t *testing.T) {
	want := "trace-1234"
	ctx := WithRequestIDValue(context.Background(), want)
	assert.Equal(t, want, GetRequestID(ctx))
}

func TestWithRequestIDValue_FreshUUIDOnEmpty(t *testing.T) {
	ctx := WithRequestIDValue(context.Background(), "")
	id := GetRequestID(ctx)
	require.NotEmpty(t, id)
	_, err := uuid.Parse(id)
	require.NoError(t, err)
}

func TestGetRequestID_EmptyForVanillaContext(t *testing.T) {
	assert.Empty(t, GetRequestID(context.Background()))
}

func TestWithContext_NoLogPanic(t *testing.T) {
	require.NoError(t, Init("test"))
	defer Sync()

	ctx := WithRequestIDValue(context.Background(), "abc-1")
	l := WithContext(ctx)
	require.NotNil(t, l)
	// Logging must not panic with the request_id field attached.
	l.Info("contextual log")

	// Without request id, must still return a logger.
	assert.NotNil(t, WithContext(context.Background()))
}

func TestWithFields_CreatesChildLogger(t *testing.T) {
	require.NoError(t, Init("test"))
	defer Sync()

	require.NotNil(t, WithFields())
}
