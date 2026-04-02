package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var Client *redis.Client

// ClientWrapper wraps redis client with helper methods
type ClientWrapper struct {
	client *redis.Client
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

// Init initializes the Redis client (legacy global client)
func Init(addr, password string, db int) error {
	logger.Info("connecting to redis", zap.String("addr", addr))

	Client = redis.NewClient(&redis.Options{
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

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("redis connection established")
	return nil
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

// Close closes the Redis connection (legacy global client)
func Close() error {
	if Client != nil {
		logger.Info("closing redis connection")
		return Client.Close()
	}
	return nil
}

// Get retrieves a value by key
func Get(ctx context.Context, key string) (string, error) {
	val, err := Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	return val, err
}

// Set sets a key-value pair with optional expiration
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Client.Set(ctx, key, value, expiration).Err()
}

// Delete deletes a key
func Delete(ctx context.Context, keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return Client.Exists(ctx, keys...).Result()
}

// Increment increments a counter
func Increment(ctx context.Context, key string) (int64, error) {
	return Client.Incr(ctx, key).Result()
}

// IncrementBy increments a counter by a specific value
func IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return Client.IncrBy(ctx, key, value).Result()
}

// Expire sets an expiration on a key
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return Client.Expire(ctx, key, expiration).Err()
}

// GetInt retrieves an integer value by key
func GetInt(ctx context.Context, key string) (int, error) {
	val, err := Client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// SetNX sets a key only if it doesn't exist (for distributed locking)
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return Client.SetNX(ctx, key, value, expiration).Result()
}

// GetDel gets a value and deletes it atomically
func GetDel(ctx context.Context, key string) (string, error) {
	val, err := Client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}
