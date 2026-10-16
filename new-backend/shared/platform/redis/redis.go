package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ClientWrapper wraps redis client with helper methods
type ClientWrapper struct {
	client *redis.Client
}

// Client exposes the underlying go-redis client for advanced operations
func (c *ClientWrapper) Client() *redis.Client {
	return c.client
}

// NewClient creates a new Redis client instance
func NewClient(addr, password string, db int) (*ClientWrapper, error) {
	logger.Info("connecting to redis", zap.String("addr", addr))

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("redis connection established")
	return &ClientWrapper{client: rdb}, nil
}

// Ping verifies the Redis connection is alive.
func (c *ClientWrapper) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection wrapper
func (c *ClientWrapper) Close() error {
	if c.client != nil {
		logger.Info("closing redis connection")
		return c.client.Close()
	}
	return nil
}

// ============================================
// Auth Service - Access Token Blacklist
// ============================================

// Blacklist key format: "blacklist:{jti}"
// Value: "1" (just a marker)
// TTL: remaining time until token expiry

// BlacklistAccessToken adds an access token to the blacklist
// jti: unique token identifier from JWT claims
// remainingTTL: time until the token would naturally expire
func (c *ClientWrapper) BlacklistAccessToken(ctx context.Context, jti string, remainingTTL time.Duration) error {
	// Only blacklist if there's remaining TTL (don't blacklist already expired tokens)
	if remainingTTL <= 0 {
		return nil
	}
	key := fmt.Sprintf("blacklist:%s", jti)
	return c.client.Set(ctx, key, "1", remainingTTL).Err()
}

// IsAccessTokenBlacklisted checks if an access token is blacklisted
func (c *ClientWrapper) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", jti)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// BlacklistAllUserTokens revokes every access token issued to the user
// before tokenVersion: JWTAuth rejects any token whose version is lower.
// Called on logout-all and password change. The TTL only has to outlive the
// longest-lived access token.
func (c *ClientWrapper) BlacklistAllUserTokens(ctx context.Context, userID string, tokenVersion int) error {
	// Store the minimum valid token version for this user
	// Any token with version < this is considered blacklisted
	key := fmt.Sprintf("user:min_token_version:%s", userID)
	return c.client.Set(ctx, key, tokenVersion, 24*time.Hour).Err()
}

// GetMinTokenVersion gets the minimum valid token version for a user
func (c *ClientWrapper) GetMinTokenVersion(ctx context.Context, userID string) (int, error) {
	key := fmt.Sprintf("user:min_token_version:%s", userID)
	val, err := c.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil // No minimum version set, all versions valid
	}
	return val, err
}

// ============================================
// Auth Service - Login Failure Throttle
// ============================================

// LoginFailureCount returns the failed logins recorded under key in its
// current window, 0 when there are none.
func (c *ClientWrapper) LoginFailureCount(ctx context.Context, key string) (int64, error) {
	n, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return n, err
}

// RecordLoginFailure counts one failed login and restarts the window, so a
// lockout lasts `window` from the most recent failure. Returns the new count.
func (c *ClientWrapper) RecordLoginFailure(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// ClearLoginFailures forgets the failures once the login succeeds.
func (c *ClientWrapper) ClearLoginFailures(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// ============================================
// Auth Service - Password Reset Token
// ============================================

func resetTokenKey(token string) string {
	return fmt.Sprintf("reset_token:%s", token)
}

// StoreResetToken stores a password reset token with email
func (c *ClientWrapper) StoreResetToken(ctx context.Context, token, email string, expiry time.Duration) error {
	return c.client.Set(ctx, resetTokenKey(token), email, expiry).Err()
}

// PeekResetToken returns the e-mail a reset token was issued for and how
// long it has left, without spending it; "" when the token is unknown or
// expired. A reset looks first and consumes only once the new password is
// ready to be written, so a failure in between leaves the link usable.
func (c *ClientWrapper) PeekResetToken(ctx context.Context, token string) (string, time.Duration, error) {
	key := resetTokenKey(token)
	pipe := c.client.Pipeline()
	get := pipe.Get(ctx, key)
	pttl := pipe.PTTL(ctx, key)
	// Exec reports GET's redis.Nil for an unknown token; that is an answer.
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return "", 0, err
	}
	email, err := get.Result()
	if errors.Is(err, redis.Nil) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}
	ttl, err := pttl.Result()
	if err != nil {
		return "", 0, err
	}
	return email, ttl, nil
}

// ConsumeResetToken returns the e-mail a reset token was issued for and
// deletes it in the same step, "" when the token is unknown or expired.
// GETDEL keeps the token single-use even when two resets race.
func (c *ClientWrapper) ConsumeResetToken(ctx context.Context, token string) (string, error) {
	val, err := c.client.GetDel(ctx, resetTokenKey(token)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// RestoreResetToken puts back a token that was consumed for a reset that
// then failed. NX: if the key exists again, whatever is there wins.
func (c *ClientWrapper) RestoreResetToken(ctx context.Context, token, email string, ttl time.Duration) error {
	return c.client.SetNX(ctx, resetTokenKey(token), email, ttl).Err()
}

// ============================================
// Demo Ops - Write Lock
// ============================================

// IsWriteLocked checks if ops:write_lock key is present in Redis DB 0.
func (c *ClientWrapper) IsWriteLocked(ctx context.Context) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	exists, err := c.client.Exists(ctx, "ops:write_lock").Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// MarkChanged records when the data first moved away from the permanent
// state; demo-ops rewinds it once that moment is REVERT_AFTER_MINUTES old and
// deletes the key. NX keeps the first change's time, so a steady stream of
// visitor writes cannot keep pushing the rewind back.
func (c *ClientWrapper) MarkChanged(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.SetNX(ctx, "ops:dirty_since", time.Now().Unix(), 0).Err()
}

// GetOpsStatus returns the current JSON status string from ops:status in Redis DB 0.
func (c *ClientWrapper) GetOpsStatus(ctx context.Context) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("redis client not available")
	}
	val, err := c.client.Get(ctx, "ops:status").Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// PushOpsCommand pushes a command JSON string to the ops:commands list in Redis DB 0.
func (c *ClientWrapper) PushOpsCommand(ctx context.Context, cmdJSON string) error {
	if c == nil || c.client == nil {
		return errors.New("redis client not available")
	}
	return c.client.LPush(ctx, "ops:commands", cmdJSON).Err()
}
