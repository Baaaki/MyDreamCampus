package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimitStore defines the interface for rate limit operations.
// Implemented by redis.ClientWrapper.
type RateLimitStore interface {
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, retryAfter int, err error)
}

// EndpointLimit defines rate limit for a specific endpoint group.
// FailClosed: when true, Redis errors return 503 instead of allowing
// the request. Use for sensitive endpoints (login, password change,
// grade writes, financial) where bypassing rate limit is unacceptable.
type EndpointLimit struct {
	Limit      int
	Window     time.Duration
	FailClosed bool
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled        bool
	ServiceName    string
	IPLimit        int
	IPWindow       time.Duration
	UserLimit      int
	UserWindow     time.Duration
	EndpointLimits map[string]EndpointLimit
}

// RateLimiter holds the rate limit store and configuration.
type RateLimiter struct {
	store  RateLimitStore
	config RateLimitConfig
}

// NewRateLimiter creates a new RateLimiter instance.
func NewRateLimiter(store RateLimitStore, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{store: store, config: config}
}

// globalRateLimiter is the global rate limiter instance (set by service initialization).
var globalRateLimiter *RateLimiter

// SetRateLimiter sets the global rate limiter.
// Should be called during service initialization after Redis client is ready.
func SetRateLimiter(rl *RateLimiter) {
	globalRateLimiter = rl
}

// IPRateLimit applies rate limiting based on client IP address.
// Place after Recovery/CORS/Logger but before auth middleware.
// If no rate limiter is configured, requests pass through.
func IPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			c.Next()
			return
		}
		rl := globalRateLimiter

		key := fmt.Sprintf("ratelimit:%s:ip:%s:global", rl.config.ServiceName, c.ClientIP())
		allowed, remaining, retryAfter, err := rl.store.CheckRateLimit(
			c.Request.Context(), key,
			rl.config.IPLimit, rl.config.IPWindow,
		)

		if err != nil {
			// Fail open - log and allow through
			logger.Error("rate limit check failed", zap.Error(err), zap.String("key", key))
			c.Next()
			return
		}

		setRateLimitHeaders(c, rl.config.IPLimit, remaining, retryAfter)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   errors.ErrTooManyReqs.Code,
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimit applies rate limiting based on authenticated user ID.
// Place after auth middleware (JWTAuth or ExtractUserFromHeaders).
// If no rate limiter is configured or user is not authenticated, requests pass through.
func UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			c.Next()
			return
		}
		rl := globalRateLimiter

		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:user:%s:global", rl.config.ServiceName, userID)
		allowed, remaining, retryAfter, err := rl.store.CheckRateLimit(
			c.Request.Context(), key,
			rl.config.UserLimit, rl.config.UserWindow,
		)

		if err != nil {
			logger.Error("rate limit check failed", zap.Error(err), zap.String("key", key))
			c.Next()
			return
		}

		setRateLimitHeaders(c, rl.config.UserLimit, remaining, retryAfter)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   errors.ErrTooManyReqs.Code,
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// EndpointRateLimit applies per-endpoint-group rate limiting.
// Use for specific sensitive endpoints like login, register, password change.
// Uses IP for unauthenticated requests, user_id if available.
func EndpointRateLimit(group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			c.Next()
			return
		}
		rl := globalRateLimiter

		endpointLimit, ok := rl.config.EndpointLimits[group]
		if !ok {
			c.Next()
			return
		}

		// Use IP for unauthenticated endpoints, user_id if available
		identifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			identifier = fmt.Sprintf("%v", userID)
		}

		key := fmt.Sprintf("ratelimit:%s:endpoint:%s:%s", rl.config.ServiceName, group, identifier)
		allowed, remaining, retryAfter, err := rl.store.CheckRateLimit(
			c.Request.Context(), key,
			endpointLimit.Limit, endpointLimit.Window,
		)

		if err != nil {
			logger.Error("rate limit check failed",
				zap.Error(err),
				zap.String("group", group),
				zap.String("key", key),
				zap.Bool("fail_closed", endpointLimit.FailClosed),
			)
			if endpointLimit.FailClosed {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "SERVICE_UNAVAILABLE",
					"message": "Service temporarily unavailable, please try again later",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		setRateLimitHeaders(c, endpointLimit.Limit, remaining, retryAfter)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   errors.ErrTooManyReqs.Code,
				"message": fmt.Sprintf("Too many %s attempts, please try again later", group),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// setRateLimitHeaders sets standard rate limit response headers.
func setRateLimitHeaders(c *gin.Context, limit, remaining, retryAfter int) {
	c.Writer.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Writer.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	if retryAfter > 0 {
		c.Writer.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}
}
