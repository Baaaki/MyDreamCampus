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

// ChangeRecorder notes that the data moved away from the permanent state.
// Implemented by redis.ClientWrapper.
type ChangeRecorder interface {
	MarkChanged(ctx context.Context) error
}

var (
	globalWriteLockStore WriteLockStore
	lockCacheMu          sync.RWMutex
	lockCacheValue       bool
	lockCacheExpiresAt   time.Time

	changeRecorderMu     sync.RWMutex
	globalChangeRecorder ChangeRecorder
)

// SetWriteLockStore sets the global WriteLockStore instance.
func SetWriteLockStore(store WriteLockStore) {
	lockCacheMu.Lock()
	defer lockCacheMu.Unlock()
	globalWriteLockStore = store
	lockCacheExpiresAt = time.Time{}
}

// SetChangeRecorder turns change tracking on; nil turns it off. Only demo
// mode sets it: outside it nothing rewinds the data, so there is nothing to
// time.
func SetChangeRecorder(recorder ChangeRecorder) {
	changeRecorderMu.Lock()
	defer changeRecorderMu.Unlock()
	globalChangeRecorder = recorder
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

// requestClaims reads the caller's access token for the global chain, which
// runs before a route's JWTAuth. Nil when there is no valid token.
func requestClaims(c *gin.Context) *utils.Claims {
	token := accessTokenFromRequest(c)
	if token == "" {
		return nil
	}
	claims, err := utils.ValidateAccessToken(token)
	if err != nil {
		return nil
	}
	return claims
}

func isSuperAdminRequest(c *gin.Context) bool {
	if c.GetBool("is_superadmin") {
		return true
	}
	claims := requestClaims(c)
	return claims != nil && claims.SuperAdmin
}

// recordChange starts the demo rewind timer after a write went through. The
// super admin is left out: their lasting changes are made in an edit session
// and saved as the permanent state. Redis trouble only costs the timer, so it
// is logged and the response stands.
func recordChange(c *gin.Context) {
	changeRecorderMu.RLock()
	recorder := globalChangeRecorder
	changeRecorderMu.RUnlock()
	if recorder == nil || c.Writer.Status() >= http.StatusBadRequest || isSuperAdminRequest(c) {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), time.Second)
	defer cancel()
	if err := recorder.MarkChanged(ctx); err != nil && logger.Log != nil {
		logger.Warn("failed to record a change for the demo rewind", zap.Error(err))
	}
}

// WriteLock middleware blocks write requests (POST, PUT, PATCH, DELETE) with 503
// while demo ops write lock is active, unless the requester is a super admin.
// A write that goes through starts the demo rewind timer (recordChange).
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

		if !locked || isSuperAdminRequest(c) {
			c.Next()
			recordChange(c)
			return
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Sistem şu an yönetici tarafından güncelleniyor. Birkaç dakika sonra tekrar deneyin.",
			"code":  "SYSTEM_EDITING",
		})
		c.Abort()
	}
}
