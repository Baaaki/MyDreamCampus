package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================
// HTTP IDEMPOTENCY
// Backing store for middleware.Idempotency. Keys are
// idem:<service>:<user_id>:<key>; the middleware owns their shape.
// ============================================

// GetIdempotencyRecord returns the stored record, or "" when the key is
// unclaimed. A missing key is not an error — it is the common case.
func (c *ClientWrapper) GetIdempotencyRecord(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// ClaimIdempotencyKey stores the record only if the key is still free, so
// two concurrent retries cannot both enter the handler. Reports whether the
// claim succeeded.
func (c *ClientWrapper) ClaimIdempotencyKey(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, ttl).Result()
}

// SaveIdempotencyRecord overwrites the in-flight marker with the finished
// response.
func (c *ClientWrapper) SaveIdempotencyRecord(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// ReleaseIdempotencyKey drops the claim so a failed attempt can be retried
// with the same key instead of being locked out for the whole TTL.
func (c *ClientWrapper) ReleaseIdempotencyKey(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
