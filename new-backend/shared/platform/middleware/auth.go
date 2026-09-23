package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
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

// ForcePasswordChangeCode is the error code clients branch on to send the
// user to the password change screen.
const ForcePasswordChangeCode = "FORCE_PASSWORD_CHANGE"

// passwordChangeAllowlist holds the only routes a token flagged with
// force_password_change may reach. Matched on the route template, not the
// raw URL, so path tricks cannot widen it.
var passwordChangeAllowlist = map[string]string{
	"/api/auth/change-password": http.MethodPost,
	"/api/auth/logout":          http.MethodPost,
}

func passwordChangeAllowed(c *gin.Context) bool {
	method, ok := passwordChangeAllowlist[c.FullPath()]
	return ok && method == c.Request.Method
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
				"message": "Oturum açmanız gerekiyor",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := utils.ValidateAccessToken(tokenString)
		if err != nil {
			logger.Warn("token validation failed",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)

			errMsg := "Oturum geçersiz, lütfen tekrar giriş yapın"
			if errors.Is(err, utils.ErrExpiredToken) {
				errMsg = "Oturumun süresi doldu, lütfen tekrar giriş yapın"
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
							"message": "Oturum doğrulanamadı, lütfen birazdan tekrar deneyin",
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
						"message": "Oturum sonlandırıldı, lütfen tekrar giriş yapın",
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
						"message": "Oturum doğrulanamadı, lütfen birazdan tekrar deneyin",
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
					"message": "Oturum sonlandırıldı, lütfen tekrar giriş yapın",
				})
				c.Abort()
				return
			}
		}

		// The first password is the user's e-mail address, which is not a
		// secret. Until it is changed the token may only change it or end
		// the session; every other route in every service refuses it.
		if claims.ForcePasswordChange && !passwordChangeAllowed(c) {
			logger.Warn("request blocked until password is changed",
				zap.String("user_id", claims.UserID),
				zap.String("path", c.FullPath()),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Devam etmek için şifrenizi değiştirmeniz gerekiyor",
				"code":  ForcePasswordChangeCode,
			})
			c.Abort()
			return
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
