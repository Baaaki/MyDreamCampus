package middleware

import (
	"strings"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JWTAuth validates JWT token and sets user claims in context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("missing authorization header")
			c.JSON(401, gin.H{
				"error":   errors.ErrUnauthorized.Code,
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Warn("invalid authorization header format")
			c.JSON(401, gin.H{
				"error":   errors.ErrUnauthorized.Code,
				"message": "Authorization header must be 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

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

		// TODO: Token version check (Redis integration)
		// This prevents revoked tokens from being used
		// cachedVersion := redis.Get("user:version:" + claims.UserID)
		// if cachedVersion != "" && claims.TokenVersion < cachedVersion {
		//     logger.Warn("token version mismatch - token revoked",
		//         zap.String("user_id", claims.UserID),
		//         zap.Int("token_version", claims.TokenVersion),
		//     )
		//     c.JSON(401, gin.H{
		//         "error": errors.ErrTokenRevoked.Code,
		//         "message": "Token has been revoked",
		//     })
		//     c.Abort()
		//     return
		// }

		// Set claims in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("department", claims.Department)
		c.Set("token_version", claims.TokenVersion)

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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
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
