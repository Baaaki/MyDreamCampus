package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WriteLockStore provides access to the ops:write_lock state in Redis.
// Implemented by redis.ClientWrapper.
type WriteLockStore interface {
	IsWriteLocked(ctx context.Context) (bool, error)
}

var (
	globalWriteLockStore WriteLockStore
	lockCacheMu          sync.RWMutex
	lockCacheValue       bool
	lockCacheExpiresAt   time.Time
)

// SetWriteLockStore sets the global WriteLockStore instance.
func SetWriteLockStore(store WriteLockStore) {
	lockCacheMu.Lock()
	defer lockCacheMu.Unlock()
	globalWriteLockStore = store
	lockCacheExpiresAt = time.Time{}
}

// exemptFromWriteLock checks whether a given request path is exempt from write lock.
func exemptFromWriteLock(path string) bool {
	if strings.HasPrefix(path, "/internal/") {
		return true
	}
	switch path {
	case "/health", "/ready", "/api/auth/login", "/api/auth/refresh", "/api/auth/logout":
		return true
	default:
		return false
	}
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// checkWriteLock checks if write lock is active using an in-process 1-second cache.
// If Redis check fails, it fails open (returns false).
func checkWriteLock(ctx context.Context) bool {
	now := time.Now()
	lockCacheMu.RLock()
	if now.Before(lockCacheExpiresAt) {
		val := lockCacheValue
		lockCacheMu.RUnlock()
		return val
	}
	lockCacheMu.RUnlock()

	lockCacheMu.Lock()
	defer lockCacheMu.Unlock()
	if now.Before(lockCacheExpiresAt) {
		return lockCacheValue
	}

	if globalWriteLockStore == nil {
		lockCacheValue = false
		lockCacheExpiresAt = now.Add(1 * time.Second)
		return false
	}

	locked, err := globalWriteLockStore.IsWriteLocked(ctx)
	if err != nil {
		if logger.Log != nil {
			logger.Warn("failed to check write lock in redis, failing open", zap.Error(err))
		}
		lockCacheValue = false
		lockCacheExpiresAt = now.Add(1 * time.Second)
		return false
	}

	lockCacheValue = locked
	lockCacheExpiresAt = now.Add(1 * time.Second)
	return locked
}

func isSuperAdminRequest(c *gin.Context) bool {
	if c.GetBool("is_superadmin") {
		return true
	}
	token := accessTokenFromRequest(c)
	if token == "" {
		return false
	}
	claims, err := utils.ValidateAccessToken(token)
	if err != nil {
		return false
	}
	return claims.SuperAdmin
}

// WriteLock middleware blocks write requests (POST, PUT, PATCH, DELETE) with 503
// while demo ops write lock is active, unless the requester is a super admin.
func WriteLock() gin.HandlerFunc {
	return func(c *gin.Context) {
		locked := checkWriteLock(c.Request.Context())
		if locked {
			c.Header("X-System-Editing", "1")
		}

		path := c.Request.URL.Path
		if exemptFromWriteLock(path) {
			c.Next()
			return
		}

		if !isWriteMethod(c.Request.Method) {
			c.Next()
			return
		}

		if !locked {
			c.Next()
			return
		}

		if isSuperAdminRequest(c) {
			c.Next()
			return
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Sistem şu an yönetici tarafından güncelleniyor. Birkaç dakika sonra tekrar deneyin.",
			"code":  "SYSTEM_EDITING",
		})
		c.Abort()
	}
}
