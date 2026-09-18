package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestGetRetryCount_AcceptsAnyIntegerWidth(t *testing.T) {
	for _, v := range []any{int8(2), int16(2), int32(2), int64(2), 2} {
		assert.Equal(t, 2, getRetryCount(amqp.Table{retryCountHeader: v}), "%T", v)
	}
	assert.Equal(t, 0, getRetryCount(nil))
	assert.Equal(t, 0, getRetryCount(amqp.Table{retryCountHeader: "2"}))
}

func TestRetryQueueArgs_RouteBackToTheOwnQueueOnly(t *testing.T) {
	args := RetryQueueArgs("grades.sync_events")

	assert.Equal(t, "", args["x-dead-letter-exchange"],
		"the default exchange delivers by queue name, never to other bindings")
	assert.Equal(t, "grades.sync_events", args["x-dead-letter-routing-key"])
	assert.Equal(t, int64(5000), args["x-message-ttl"])
	assert.NoError(t, args.Validate())
}

func TestRetryQueueName_IsDistinctFromDLQ(t *testing.T) {
	assert.Equal(t, "q.retry", RetryQueueName("q"))
	assert.NotEqual(t, DLQName("q"), RetryQueueName("q"))
}
