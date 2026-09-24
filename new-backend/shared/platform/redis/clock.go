package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================
// TIME MACHINE
// Backing store for clocksync. One key holds the cluster-wide clock state
// and a pub/sub channel tells every service to re-read it; clocksync owns
// both names.
// ============================================

// GetClockState returns the stored state, or "" when no simulation is set.
func (c *ClientWrapper) GetClockState(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

// StoreClockState writes the state and announces it in one transaction, so
// a service woken by the message always reads the new value. A zero ttl
// keeps the key until it is deleted.
func (c *ClientWrapper) StoreClockState(ctx context.Context, key, channel, value string, ttl time.Duration) error {
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, key, value, ttl)
	pipe.Publish(ctx, channel, "set")
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteClockState removes the state and announces it in one transaction.
func (c *ClientWrapper) DeleteClockState(ctx context.Context, key, channel string) error {
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.Publish(ctx, channel, "clear")
	_, err := pipe.Exec(ctx)
	return err
}

// WatchClockChanges signals on the returned channel whenever a message
// arrives on channel, until ctx is cancelled. Bursts collapse into one
// signal: the receiver re-reads the key anyway. go-redis resubscribes on
// its own after a dropped connection; messages sent while it was down are
// lost, which is why clocksync also re-reads on a timer.
func (c *ClientWrapper) WatchClockChanges(ctx context.Context, channel string) <-chan struct{} {
	sub := c.client.Subscribe(ctx, channel)
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		defer func() { _ = sub.Close() }()
		msgs := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-msgs:
				if !ok {
					return
				}
				select {
				case out <- struct{}{}:
				default:
				}
			}
		}
	}()
	return out
}
