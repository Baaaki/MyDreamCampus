package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// reconnectDelay is the pause between reconnect attempts.
const reconnectDelay = 5 * time.Second

// Connection wraps a RabbitMQ connection and one channel, and keeps them
// alive: a single supervisor goroutine notices when either closes and
// reconnects. Consumers learn about a new channel through Reconnected and
// subscribe again — without that, a broker restart leaves every service
// connected but silently consuming nothing.
type Connection struct {
	URL string

	mu      sync.RWMutex
	conn    *amqp.Connection
	channel *amqp.Channel
	// reconnected is closed, then replaced, each time a new channel is up.
	reconnected chan struct{}
	// connClosed/chanClosed are the NotifyClose channels of the current pair.
	connClosed chan *amqp.Error
	chanClosed chan *amqp.Error

	connected atomic.Bool
	closing   atomic.Bool
	done      chan struct{}
}

// NewConnection creates a new RabbitMQ connection with retry backoff
func NewConnection(url string) (*Connection, error) {
	c := &Connection{
		URL:         url,
		reconnected: make(chan struct{}),
		done:        make(chan struct{}),
	}

	// Retry connection with backoff (max 5 attempts)
	maxRetries := 5
	for i := range maxRetries {
		if err := c.connect(); err != nil {
			logger.Error("rabbitmq connection failed",
				zap.Error(err),
				zap.Int("attempt", i+1),
				zap.Int("max_retries", maxRetries),
			)

			if i < maxRetries-1 {
				logger.Info("retrying rabbitmq connection in 5 seconds...")
				time.Sleep(reconnectDelay)
				continue
			}
			return nil, err
		}
		break
	}

	// One supervisor for the life of the Connection. Starting one per
	// connect() — as this used to — left every earlier one looping on a
	// channel that had been replaced.
	go c.supervise()
	return c, nil
}

// connect dials a new connection and channel and publishes them.
func (c *Connection) connect() error {
	logger.Info("connecting to rabbitmq")

	conn, err := amqp.Dial(c.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// prefetchCount: number of unacknowledged messages before blocking
	if err := channel.Qos(10, 0, false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Registered before anything else can use the pair, so no close goes
	// unnoticed. Buffered: amqp blocks on an unread notification.
	connClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
	chanClosed := channel.NotifyClose(make(chan *amqp.Error, 1))

	c.mu.Lock()
	c.conn = conn
	c.channel = channel
	c.connClosed = connClosed
	c.chanClosed = chanClosed
	previous := c.reconnected
	c.reconnected = make(chan struct{})
	c.mu.Unlock()

	c.connected.Store(true)
	close(previous) // wake consumers waiting for a channel

	logger.Info("rabbitmq connection established")
	return nil
}

// supervise waits for the current connection or channel to close and
// replaces them, until Close is called.
func (c *Connection) supervise() {
	for {
		c.mu.RLock()
		connClosed, chanClosed, conn := c.connClosed, c.chanClosed, c.conn
		c.mu.RUnlock()

		var reason *amqp.Error
		select {
		case <-c.done:
			return
		case reason = <-connClosed:
		case reason = <-chanClosed:
		}
		if c.closing.Load() {
			return
		}

		c.connected.Store(false)
		logger.Error("rabbitmq connection lost", zap.Any("reason", reason))
		// A channel-level error (e.g. publishing to a missing exchange)
		// leaves the connection open; close it so it does not leak.
		_ = conn.Close()

		for {
			select {
			case <-c.done:
				return
			case <-time.After(reconnectDelay):
			}
			logger.Info("attempting to reconnect to rabbitmq")
			if err := c.connect(); err != nil {
				logger.Error("rabbitmq reconnection failed", zap.Error(err))
				continue
			}
			logger.Info("rabbitmq reconnection successful")
			break
		}
	}
}

// Channel returns the current channel. It changes after a reconnect, so
// callers must not keep it.
func (c *Connection) Channel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channel
}

// Reconnected returns a channel that is closed the next time a new AMQP
// channel comes up. Take it BEFORE using Channel(), so a reconnect that
// happens in between is not missed.
func (c *Connection) Reconnected() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.reconnected
}

// Done is closed once Close is called.
func (c *Connection) Done() <-chan struct{} {
	return c.done
}

// Close stops reconnecting and closes the connection.
func (c *Connection) Close() error {
	if !c.closing.CompareAndSwap(false, true) {
		return nil
	}
	close(c.done)
	c.connected.Store(false)

	c.mu.RLock()
	channel, conn := c.channel, c.conn
	c.mu.RUnlock()

	if channel != nil {
		_ = channel.Close()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// IsConnected returns connection status
func (c *Connection) IsConnected() bool {
	return c.connected.Load()
}

// Ping verifies the RabbitMQ connection is alive.
// The amqp library doesn't expose a network round-trip, so we check
// the underlying conn for liveness flags.
func (c *Connection) Ping(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if !c.connected.Load() || conn == nil {
		return errors.New("rabbitmq not connected")
	}
	if conn.IsClosed() {
		return errors.New("rabbitmq connection is closed")
	}
	return nil
}
