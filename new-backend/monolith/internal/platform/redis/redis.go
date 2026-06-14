package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
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

// SetTokenVersion sets the token version for a user in Redis
func (c *ClientWrapper) SetTokenVersion(ctx context.Context, userID string, version int) error {
	key := fmt.Sprintf("user:version:%s", userID)
	return c.client.Set(ctx, key, version, 24*time.Hour).Err()
}

// GetTokenVersion gets the token version for a user from Redis
func (c *ClientWrapper) GetTokenVersion(ctx context.Context, userID string) (int, error) {
	key := fmt.Sprintf("user:version:%s", userID)
	val, err := c.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil // Key doesn't exist
	}
	return val, err
}

// ============================================
// Auth Service - Refresh Token Management
// ============================================

// RefreshToken key format: "refresh_token:{jti}"
// Value: user_id
// TTL: refresh token expiry time

// StoreRefreshToken stores a refresh token JTI with user_id
func (c *ClientWrapper) StoreRefreshToken(ctx context.Context, jti, userID string, expiry time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", jti)
	return c.client.Set(ctx, key, userID, expiry).Err()
}

// GetRefreshToken retrieves user_id by refresh token JTI
func (c *ClientWrapper) GetRefreshToken(ctx context.Context, jti string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", jti)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Token not found (expired or never existed)
	}
	return val, err
}

// DeleteRefreshToken removes a refresh token from Redis
func (c *ClientWrapper) DeleteRefreshToken(ctx context.Context, jti string) error {
	key := fmt.Sprintf("refresh_token:%s", jti)
	return c.client.Del(ctx, key).Err()
}

// DeleteAllUserRefreshTokens removes all refresh tokens for a user
// Uses SCAN to find and delete all tokens (safer than KEYS for production)
func (c *ClientWrapper) DeleteAllUserRefreshTokens(ctx context.Context, userID string) error {
	var cursor uint64
	var keysToDelete []string

	for {
		var keys []string
		var err error
		keys, cursor, err = c.client.Scan(ctx, cursor, "refresh_token:*", 100).Result()
		if err != nil {
			return err
		}

		// Check each key's value to see if it belongs to this user
		for _, key := range keys {
			val, err := c.client.Get(ctx, key).Result()
			if err == nil && val == userID {
				keysToDelete = append(keysToDelete, key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	if len(keysToDelete) > 0 {
		return c.client.Del(ctx, keysToDelete...).Err()
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

// BlacklistAllUserTokens blacklists all tokens for a user by incrementing token version
// This is more efficient than storing individual blacklist entries
// Called during logout-all or password change
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
// Auth Service - Password Reset Token
// ============================================

// StoreResetToken stores a password reset token with email
func (c *ClientWrapper) StoreResetToken(ctx context.Context, token, email string, expiry time.Duration) error {
	key := fmt.Sprintf("reset_token:%s", token)
	return c.client.Set(ctx, key, email, expiry).Err()
}

// GetResetToken retrieves email by reset token
func (c *ClientWrapper) GetResetToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("reset_token:%s", token)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Token not found or expired
	}
	return val, err
}

// DeleteResetToken removes a password reset token from Redis
func (c *ClientWrapper) DeleteResetToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("reset_token:%s", token)
	return c.client.Del(ctx, key).Err()
}
