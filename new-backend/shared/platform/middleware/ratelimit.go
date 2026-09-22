package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
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

// exemptFromGlobalLimit lists paths the global limiter never counts.
// /internal is service-to-service traffic: during course registration
// enrollment calls catalog and student for every student, all from one
// container IP, and a 429 there fails a student's request for nothing.
// Probes must not be throttled either.
func exemptFromGlobalLimit(path string) bool {
	return strings.HasPrefix(path, "/internal/") || path == "/health" || path == "/ready"
}

// requestIdentity names whose bucket a request spends: the user when it
// carries a validly signed access token, the client IP otherwise. Behind a
// campus NAT or a Cloudflare Tunnel thousands of users share one IP, and
// keying their logged-in traffic by it made them exhaust each other's limit.
// A signature check is enough — revocation is JWTAuth's job, and a revoked
// token can only spend its own bucket.
func requestIdentity(c *gin.Context) string {
	if userID := c.GetString("user_id"); userID != "" {
		return "user:" + userID
	}
	if token := accessTokenFromRequest(c); token != "" {
		if claims, err := utils.ValidateAccessToken(token); err == nil {
			return "user:" + claims.UserID
		}
	}
	// A refresh carries no access token by definition; without this every
	// user behind the NAT would renew their session out of one IP bucket.
	if c.Request.URL.Path == refreshPath {
		return refreshIdentity(c)
	}
	return "ip:" + c.ClientIP()
}

// refreshPath is the one route whose caller is identified by a refresh token.
const refreshPath = "/api/auth/refresh"

func accessTokenFromRequest(c *gin.Context) string {
	if token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok {
		return token
	}
	if cookie, err := c.Cookie("access_token"); err == nil {
		return cookie
	}
	return ""
}

// refreshIdentity keys a refresh request by the refresh token's owner — 10
// refreshes a minute shared by a whole campus NAT would log users out. The
// token is read from the cookie (web) or the body (mobile); the body is
// restored for the handler.
func refreshIdentity(c *gin.Context) string {
	token, _ := c.Cookie("refresh_token")
	if token == "" && c.Request.Body != nil {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRefreshBodyBytes))
		c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), c.Request.Body))
		if err == nil {
			var payload struct {
				RefreshToken string `json:"refresh_token"`
			}
			if json.Unmarshal(body, &payload) == nil {
				token = payload.RefreshToken
			}
		}
	}
	if token != "" {
		if claims, err := utils.ValidateToken(token); err == nil && claims.TokenType == string(utils.RefreshToken) {
			return "user:" + claims.UserID
		}
	}
	return "ip:" + c.ClientIP()
}

// userBucketCounted marks a request whose user bucket IPRateLimit already
// charged.
const userBucketCounted = "ratelimit_user_counted"

func userBucketKey(service, userID string) string {
	return fmt.Sprintf("ratelimit:%s:user:%s:global", service, userID)
}

// maxRefreshBodyBytes caps what refreshIdentity reads; a refresh body is a
// single JWT.
const maxRefreshBodyBytes = 8 << 10

// IPRateLimit is the global limiter every service mounts. Despite the name
// it is keyed per user for authenticated requests (see requestIdentity) and
// per client IP only for anonymous ones.
// Place after Recovery/CORS/Logger but before auth middleware.
// If no rate limiter is configured, requests pass through.
func IPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil || exemptFromGlobalLimit(c.Request.URL.Path) {
			c.Next()
			return
		}
		rl := globalRateLimiter

		limit, window := rl.config.IPLimit, rl.config.IPWindow
		key := fmt.Sprintf("ratelimit:%s:ip:%s:global", rl.config.ServiceName, c.ClientIP())
		if userID, isUser := strings.CutPrefix(requestIdentity(c), "user:"); isUser {
			limit, window = rl.config.UserLimit, rl.config.UserWindow
			key = userBucketKey(rl.config.ServiceName, userID)
			// UserRateLimit further down the chain spends this same bucket;
			// counting the request twice would halve the user's limit.
			c.Set(userBucketCounted, true)
		}

		allowed, remaining, retryAfter, err := rl.store.CheckRateLimit(
			c.Request.Context(), key, limit, window,
		)

		if err != nil {
			// Fail open - log and allow through
			logger.Error("rate limit check failed", zap.Error(err), zap.String("key", key))
			c.Next()
			return
		}

		setRateLimitHeaders(c, limit, remaining, retryAfter)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   errors.ErrTooManyReqs.Code,
				"message": "Çok fazla istek, lütfen biraz sonra tekrar deneyin",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimit applies rate limiting based on authenticated user ID.
// Place after JWTAuth so the user ID is present in the gin context.
// If no rate limiter is configured or user is not authenticated, requests pass through.
func UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			c.Next()
			return
		}
		rl := globalRateLimiter

		userID := c.GetString("user_id")
		if userID == "" || c.GetBool(userBucketCounted) {
			c.Next()
			return
		}

		key := userBucketKey(rl.config.ServiceName, userID)
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
				"message": "Çok fazla istek, lütfen biraz sonra tekrar deneyin",
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

		// Login and password-reset requests have no user yet and stay keyed
		// by IP — that is what bounds credential stuffing from one address.
		identifier := requestIdentity(c)

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
					"message": "Hizmet şu anda kullanılamıyor, lütfen birazdan tekrar deneyin",
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
				"message": "Çok fazla deneme yapıldı, lütfen biraz sonra tekrar deneyin",
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
