package middleware

import (
	"slices"

	"github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
				"message": "Bu işlem için yetkiniz yok",
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
			"message": "Bu işlem için yetkiniz yok",
		})
		c.Abort()
	}
}

// RequireSelfOrRole lets a request through when the :param path segment is
// the caller's own user ID, or when the caller holds one of the roles. Use it
// on per-user reads (GET /staff/:id) that must not become a directory of
// everyone else's contact details.
func RequireSelfOrRole(param string, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if slices.Contains(allowedRoles, c.GetString("role")) {
			c.Next()
			return
		}

		self, errSelf := uuid.Parse(c.GetString("user_id"))
		target, errTarget := uuid.Parse(c.Param(param))
		if errSelf == nil && errTarget == nil && self == target {
			c.Next()
			return
		}

		logger.Warn("access denied - not the resource owner",
			zap.String("user_role", c.GetString("role")),
			zap.String("path", c.Request.URL.Path),
		)
		c.JSON(403, gin.H{
			"error":   errors.ErrForbidden.Code,
			"message": "Bu kaynağa erişim yetkiniz yok",
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
