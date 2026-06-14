package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Connection wraps RabbitMQ connection with auto-reconnect
type Connection struct {
	URL       string
	conn      *amqp.Connection
	channel   *amqp.Channel
	closeChan chan *amqp.Error
	connected bool
}

// NewConnection creates a new RabbitMQ connection with retry backoff
func NewConnection(url string) (*Connection, error) {
	c := &Connection{
		URL:       url,
		connected: false,
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
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, err
		}
		break
	}

	return c, nil
}

// connect establishes connection to RabbitMQ
func (c *Connection) connect() error {
	logger.Info("connecting to rabbitmq", zap.String("url", c.URL))

	conn, err := amqp.Dial(c.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS (Quality of Service)
	// prefetchCount: number of unacknowledged messages before blocking
	if err := channel.Qos(10, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	c.conn = conn
	c.channel = channel
	c.connected = true

	// Listen for connection errors
	c.closeChan = make(chan *amqp.Error)
	c.channel.NotifyClose(c.closeChan)

	// Start reconnection listener
	go c.handleReconnect()

	logger.Info("rabbitmq connection established")
	return nil
}

// handleReconnect listens for connection errors and attempts to reconnect
func (c *Connection) handleReconnect() {
	for {
		err := <-c.closeChan
		if err != nil {
			logger.Error("rabbitmq connection lost", zap.Error(err))
			c.connected = false

			// Attempt to reconnect
			for {
				logger.Info("attempting to reconnect to rabbitmq...")
				time.Sleep(5 * time.Second)

				if err := c.connect(); err != nil {
					logger.Error("rabbitmq reconnection failed", zap.Error(err))
					continue
				}

				logger.Info("rabbitmq reconnection successful")
				break
			}
		}
	}
}

// Channel returns the current channel
func (c *Connection) Channel() *amqp.Channel {
	return c.channel
}

// Close closes the connection
func (c *Connection) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected returns connection status
func (c *Connection) IsConnected() bool {
	return c.connected
}

// Ping verifies the RabbitMQ connection is alive.
// The amqp library doesn't expose a network round-trip, so we check
// the underlying conn for liveness flags.
func (c *Connection) Ping(ctx context.Context) error {
	if !c.connected || c.conn == nil {
		return errors.New("rabbitmq not connected")
	}
	if c.conn.IsClosed() {
		return errors.New("rabbitmq connection is closed")
	}
	return nil
}
