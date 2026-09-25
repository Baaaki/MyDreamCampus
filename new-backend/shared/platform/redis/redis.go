package redis

import (
	"context"
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

// StoreResetToken stores a password reset token with email
func (c *ClientWrapper) StoreResetToken(ctx context.Context, token, email string, expiry time.Duration) error {
	key := fmt.Sprintf("reset_token:%s", token)
	return c.client.Set(ctx, key, email, expiry).Err()
}

// ConsumeResetToken returns the e-mail a reset token was issued for and
// deletes it in the same step, "" when the token is unknown or expired.
// GETDEL keeps the token single-use even when two resets race.
func (c *ClientWrapper) ConsumeResetToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("reset_token:%s", token)
	val, err := c.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
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

