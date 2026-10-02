//go:build integration

package rabbitmq

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBrokerURL names the variable holding the AMQP URL of the test broker.
const testBrokerURL = "TEST_RABBITMQ_URL"

// newTestConnection connects to the throwaway broker `make test-integration`
// starts. A missing URL fails rather than skips: a tagged run that silently
// tested nothing would read as green.
func newTestConnection(t *testing.T) *Connection {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	url := os.Getenv(testBrokerURL)
	if url == "" {
		t.Fatalf("%s is not set; run `make test-integration` from the repository root", testBrokerURL)
	}
	conn, err := NewConnection(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func testName(kind string) string { return "it." + kind + "." + uuid.NewString()[:8] }

func TestPublish_ReturnsOnlyOnceTheBrokerHoldsTheMessage(t *testing.T) {
	conn := newTestConnection(t)
	publisher := NewPublisher(conn)
	exchange, queue := testName("exchange"), testName("queue")
	require.NoError(t, publisher.DeclareExchange(exchange))
	require.NoError(t, publisher.DeclareAndBindQueue(queue, exchange, "it.created"))

	require.NoError(t, publisher.Publish(context.Background(), exchange, "it.created", map[string]any{"id": 1}))

	msg, ok, err := conn.Channel().Get(queue, true)
	require.NoError(t, err)
	require.True(t, ok, "a confirmed message is already in its queue")
	assert.JSONEq(t, `{"id":1}`, string(msg.Body))
}

// Without confirms this publish "succeeded": the bytes left the socket, and
// only then did the broker close the channel over the missing exchange.
func TestPublish_BrokerRefusesTheMessage_NotConfirmed(t *testing.T) {
	conn := newTestConnection(t)

	err := NewPublisher(conn).Publish(context.Background(), testName("missing"), "it.created", map[string]any{"id": 1})

	assert.ErrorIs(t, err, ErrNotConfirmed)
}

// A failed delivery is copied to the retry queue before the original is
// acked, and the copy is confirmed first: losing it would lose the message.
func TestConsumeWithDLQ_FailedDeliveryComesBackAfterTheRetryDelay(t *testing.T) {
	conn := newTestConnection(t)
	consumer := NewConsumer(conn)
	queue := testName("work")
	require.NoError(t, consumer.DeclareQueue(queue))

	var attempts atomic.Int32
	var once sync.Once
	redelivered := make(chan struct{})
	require.NoError(t, consumer.ConsumeWithDLQ(queue, func([]byte) error {
		if attempts.Add(1) == 1 {
			return errors.New("first attempt fails")
		}
		once.Do(func() { close(redelivered) })
		return nil
	}, MaxDeliveryAttempts))

	require.NoError(t, publishConfirmed(context.Background(), conn.Channel(), "", queue,
		amqp.Publishing{ContentType: "application/json", Body: []byte(`{}`), DeliveryMode: amqp.Persistent}))

	select {
	case <-redelivered:
	case <-time.After(RetryDelay + 10*time.Second):
		t.Fatal("the failed delivery never came back from the retry queue")
	}
	assert.Equal(t, int32(2), attempts.Load())
}
