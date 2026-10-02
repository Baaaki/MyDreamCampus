package consumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConsumer_Dispatch_InvalidDataPayload(t *testing.T) {
	c := &Consumer{
		log: zap.NewNop(),
	}

	t.Run("missing data field", func(t *testing.T) {
		err := c.dispatch(context.Background(), "evt-1", "user.registered", map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid 'data' payload")
	})

	t.Run("data is not a map", func(t *testing.T) {
		err := c.dispatch(context.Background(), "evt-2", "user.registered", map[string]any{
			"data": "invalid-string-payload",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid 'data' payload")
	})

	t.Run("unhandled event type returns nil without error", func(t *testing.T) {
		err := c.dispatch(context.Background(), "evt-3", "unknown.custom_event", map[string]any{
			"data": map[string]any{"foo": "bar"},
		})
		assert.NoError(t, err)
	})
}
