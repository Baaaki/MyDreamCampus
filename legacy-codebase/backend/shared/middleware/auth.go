package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TokenBlacklistChecker interface for checking token blacklist
// Implemented by redis.ClientWrapper
type TokenBlacklistChecker interface {
	IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error)
	GetMinTokenVersion(ctx context.Context, userID string) (int, error)
}

// blacklistChecker is the global blacklist checker (set by auth service)
var blacklistChecker TokenBlacklistChecker

// SetBlacklistChecker sets the global blacklist checker
// Should be called during auth service initialization
func SetBlacklistChecker(checker TokenBlacklistChecker) {
	blacklistChecker = checker
}

// AuthOption configures JWT auth middleware behavior.
type AuthOption func(*authConfig)

type authConfig struct {
	failClosed bool
}

// WithFailClosed makes the blacklist/version check return 503 when Redis
// is unreachable. Use for sensitive endpoints (password change, grade
// writes, financial) where unverified token acceptance is unacceptable.
// Default behavior is fail-open for general availability.
func WithFailClosed() AuthOption {
	return func(c *authConfig) { c.failClosed = true }
}

// JWTAuth validates JWT token and sets user claims in context.
// Pass WithFailClosed() to require Redis-backed revocation checks
// to succeed before the request proceeds.
func JWTAuth(opts ...AuthOption) gin.HandlerFunc {
	cfg := authConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *gin.Context) {
		// Try Authorization header first
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// Fallback to cookie if no Authorization header
		if tokenString == "" {
			if cookie, err := c.Cookie("access_token"); err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			logger.Warn("no token provided")
			c.JSON(401, gin.H{
				"error":   errors.ErrUnauthorized.Code,
				"message": "No token provided",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			logger.Warn("token validation failed",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)

			var errMsg string
			if err == utils.ErrExpiredToken {
				errMsg = "Token has expired"
			} else {
				errMsg = "Invalid token"
			}

			c.JSON(401, gin.H{
				"error":   errors.ErrUnauthorized.Code,
				"message": errMsg,
			})
			c.Abort()
			return
		}

		// Check blacklist if checker is configured (Redis available)
		if blacklistChecker != nil {
			ctx := c.Request.Context()

			// Check if specific token JTI is blacklisted
			if claims.JTI != "" {
				isBlacklisted, err := blacklistChecker.IsAccessTokenBlacklisted(ctx, claims.JTI)
				if err != nil {
					logger.Error("failed to check token blacklist",
						zap.Error(err),
						zap.String("jti", claims.JTI),
						zap.Bool("fail_closed", cfg.failClosed),
					)
					if cfg.failClosed {
						c.JSON(http.StatusServiceUnavailable, gin.H{
							"error":   "SERVICE_UNAVAILABLE",
							"message": "Token revocation check unavailable, please try again later",
						})
						c.Abort()
						return
					}
					// Continue on error - fail open for availability
				} else if isBlacklisted {
					logger.Warn("blacklisted token used",
						zap.String("user_id", claims.UserID),
						zap.String("jti", claims.JTI),
					)
					c.JSON(401, gin.H{
						"error":   errors.ErrUnauthorized.Code,
						"message": "Token has been revoked",
					})
					c.Abort()
					return
				}
			}

			// Check token version (for logout-all scenarios)
			minVersion, err := blacklistChecker.GetMinTokenVersion(ctx, claims.UserID)
			if err != nil {
				logger.Error("failed to check min token version",
					zap.Error(err),
					zap.String("user_id", claims.UserID),
					zap.Bool("fail_closed", cfg.failClosed),
				)
				if cfg.failClosed {
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"error":   "SERVICE_UNAVAILABLE",
						"message": "Token revocation check unavailable, please try again later",
					})
					c.Abort()
					return
				}
				// Continue on error - fail open for availability
			} else if minVersion > 0 && claims.TokenVersion < minVersion {
				logger.Warn("token version too old - all tokens revoked",
					zap.String("user_id", claims.UserID),
					zap.Int("token_version", claims.TokenVersion),
					zap.Int("min_version", minVersion),
				)
				c.JSON(401, gin.H{
					"error":   errors.ErrUnauthorized.Code,
					"message": "Token has been revoked",
				})
				c.Abort()
				return
			}
		}

		// Set claims in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("department", claims.Department)
		c.Set("token_version", claims.TokenVersion)
		c.Set("jti", claims.JTI)
		c.Set("force_password_change", claims.ForcePasswordChange)

		logger.Debug("jwt authentication successful",
			zap.String("user_id", claims.UserID),
			zap.String("role", claims.Role),
		)

		c.Next()
	}
}

// OptionalJWTAuth validates JWT if present, but doesn't require it
func OptionalJWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try Authorization header first
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// Fallback to cookie
		if tokenString == "" {
			if cookie, err := c.Cookie("access_token"); err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("department", claims.Department)
		c.Set("token_version", claims.TokenVersion)

		c.Next()
	}
}

// StripInternalHeaders removes internal service headers from incoming requests
// to prevent header spoofing. These headers should only be set by Traefik
// after successful forward-auth verification.
// This middleware should be placed BEFORE other auth middleware in the chain.
// Internal services that communicate directly should include X-Internal-Secret.
//
// INTERNAL_SERVICE_SECRET environment variable must be set; the constructor
// panics on init rather than per-request to fail fast at startup.
func StripInternalHeaders() gin.HandlerFunc {
	internalSecret := os.Getenv("INTERNAL_SERVICE_SECRET")
	if internalSecret == "" {
		panic("INTERNAL_SERVICE_SECRET environment variable is not set")
	}

	return func(c *gin.Context) {
		receivedSecret := c.GetHeader("X-Internal-Secret")
		if receivedSecret != internalSecret {
			// Request is not from a trusted internal source - strip auth headers
			c.Request.Header.Del("X-User-ID")
			c.Request.Header.Del("X-User-Role")
			c.Request.Header.Del("X-User-Email")
			c.Request.Header.Del("X-Internal-Secret")
		}

		c.Next()
	}
}

// ExtractUserFromHeaders extracts user information from X-User-* headers
// These headers are set by Traefik after forward-auth to auth-service
// Use this middleware for services that rely on Traefik for authentication
func ExtractUserFromHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from header (required)
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			logger.Warn("X-User-ID header missing - request may not have passed through auth gateway")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   errors.ErrUnauthorized.Code,
				"message": "Authentication required",
			})
			return
		}

		// Extract role from header (required)
		role := c.GetHeader("X-User-Role")
		if role == "" {
			logger.Warn("X-User-Role header missing - request may not have passed through auth gateway",
				zap.String("user_id", userID),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   errors.ErrUnauthorized.Code,
				"message": "User role not found",
			})
			return
		}

		// Extract department from header (optional)
		department := c.GetHeader("X-User-Department")

		// Set user information in context for downstream handlers
		c.Set("user_id", userID)
		c.Set("role", role)
		if department != "" {
			c.Set("department", department)
		}

		logger.Debug("user extracted from headers",
			zap.String("user_id", userID),
			zap.String("role", role),
		)

		c.Next()
	}
}
