package audit

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitSecurity_DoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() { InitSecurity("test") })
	require.NotPanics(t, func() { InitSecurity("production") })
	SyncSecurity()
}

func TestLogSecurity_NoCrashWithoutInit(t *testing.T) {
	// Reset securityLogger to nil and log — must be a no-op
	securityLogger = nil
	require.NotPanics(t, func() {
		LogSecurity(SecurityEvent{EventType: EventLogin, Result: "success"})
	})
}

func TestLogSecurity_AcceptsAllOptionalFields(t *testing.T) {
	InitSecurity("test")
	defer SyncSecurity()

	require.NotPanics(t, func() {
		LogSecurity(SecurityEvent{
			EventType: EventLoginFailed,
			UserID:    "u-1",
			Email:     "x@example.com",
			IP:        "1.2.3.4",
			UserAgent: "test-agent",
			Resource:  "/auth/login",
			Action:    "POST",
			Result:    "failure",
			Reason:    "invalid_credentials",
			Metadata:  map[string]string{"attempt": "3"},
		})
	})
}

func TestLogSecurityFromContext(t *testing.T) {
	InitSecurity("test")
	defer SyncSecurity()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("User-Agent", "scanner")

	require.NotPanics(t, func() {
		LogSecurityFromContext(c, EventAccessDenied, "failure", "u-2")
	})

	require.NotPanics(t, func() {
		LogSecurityFromContextWithDetails(
			c, EventCSRFViolation, "failure", "u-3", "missing token",
			map[string]string{"path": "/api/v1/admin"},
		)
	})
}

func TestSecurityEventTypes_StableConstants(t *testing.T) {
	// Sentinel test: any rename here breaks downstream Loki/Grafana queries.
	cases := map[SecurityEventType]string{
		EventLogin:           "LOGIN",
		EventLoginFailed:     "LOGIN_FAILED",
		EventLogout:          "LOGOUT",
		EventLogoutAll:       "LOGOUT_ALL",
		EventPasswordChange:  "PASSWORD_CHANGE",
		EventTokenRefresh:    "TOKEN_REFRESH",
		EventAccountLocked:   "ACCOUNT_LOCKED",
		EventAccountUnlocked: "ACCOUNT_UNLOCKED",
		EventRoleChange:      "ROLE_CHANGE",
		EventUserCreated:     "USER_CREATED",
		EventUserDeactivated: "USER_DEACTIVATED",
		EventAccessDenied:    "ACCESS_DENIED",
		EventCSRFViolation:   "CSRF_VIOLATION",
		EventBulkImport:      "BULK_IMPORT",
	}
	for got, want := range cases {
		assert.Equal(t, want, string(got))
	}
}
