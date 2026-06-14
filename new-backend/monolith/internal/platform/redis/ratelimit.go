package redis

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// rateLimitCounter provides unique members for sorted set entries
var rateLimitCounter int64

// rateLimitScript is a Lua script for sliding window rate limiting using sorted sets.
// It atomically checks and increments the rate limit counter in a single Redis roundtrip.
//
// KEYS[1] = rate limit key
// ARGV[1] = current timestamp (microseconds)
// ARGV[2] = window size (microseconds)
// ARGV[3] = max requests allowed in window
// ARGV[4] = unique member for this request
//
// Returns: {rejected(0/1), remaining, retry_after_seconds}
var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

local count = redis.call('ZCARD', key)

if count < limit then
    redis.call('ZADD', key, now, member)
    redis.call('PEXPIRE', key, math.ceil(window / 1000))
    return {0, limit - count - 1, 0}
else
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retry_after = 0
    if #oldest > 0 then
        retry_after = math.ceil((tonumber(oldest[2]) + window - now) / 1000000)
    end
    return {1, 0, retry_after}
end
`)

// CheckRateLimit checks if a request is within rate limits using a sliding window algorithm.
// Returns: allowed (bool), remaining requests (int), retry-after seconds (int), error.
func (c *ClientWrapper) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, int, error) {
	now := time.Now().UnixMicro()
	windowMicro := window.Microseconds()
	counter := atomic.AddInt64(&rateLimitCounter, 1)
	member := fmt.Sprintf("%d:%d", now, counter)

	result, err := rateLimitScript.Run(ctx, c.client, []string{key},
		now, windowMicro, limit, member,
	).Int64Slice()
	if err != nil {
		return false, 0, 0, fmt.Errorf("rate limit script failed: %w", err)
	}

	rejected := result[0] == 1
	remaining := int(result[1])
	retryAfter := int(result[2])

	return !rejected, remaining, retryAfter, nil
}
