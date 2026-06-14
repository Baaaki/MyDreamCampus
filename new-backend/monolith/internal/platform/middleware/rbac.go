package middleware

import (
	"slices"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequireRole checks if the authenticated user has one of the allowed roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			logger.Error("role not found in context - JWT middleware not applied?")
			c.JSON(403, gin.H{
				"error":   errors.ErrForbidden.Code,
				"message": "Role information not found",
			})
			c.Abort()
			return
		}

		userRole := role.(string)

		// Check if user role is in allowed roles
		if slices.Contains(allowedRoles, userRole) {
			c.Next()
			return
		}

		// Access denied
		logger.Warn("access denied - insufficient permissions",
			zap.String("user_role", userRole),
			zap.Strings("allowed_roles", allowedRoles),
			zap.String("path", c.Request.URL.Path),
		)

		c.JSON(403, gin.H{
			"error":   errors.ErrForbidden.Code,
			"message": "You do not have permission to access this resource",
		})
		c.Abort()
	}
}

// RequireAdmin is a convenience wrapper for admin-only endpoints
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin")
}

// RequireTeacherOrAdmin allows both teachers and admins
func RequireTeacherOrAdmin() gin.HandlerFunc {
	return RequireRole("teacher", "admin")
}

// RequireStudent is a convenience wrapper for student-only endpoints
func RequireStudent() gin.HandlerFunc {
	return RequireRole("student")
}
