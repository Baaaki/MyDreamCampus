package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOpsBackend struct {
	status string
	pushed []string
	err    error
}

func (m *mockOpsBackend) GetOpsStatus(_ context.Context) (string, error) {
	return m.status, m.err
}

func (m *mockOpsBackend) PushOpsCommand(_ context.Context, cmdJSON string) error {
	if m.err != nil {
		return m.err
	}
	m.pushed = append(m.pushed, cmdJSON)
	return nil
}

type mockAuditLogger struct {
	events []audit.AuditEvent
}

func (m *mockAuditLogger) Log(_ context.Context, event audit.AuditEvent) error {
	m.events = append(m.events, event)
	return nil
}

func createTestToken(superAdmin bool) string {
	now := time.Now()
	claims := &utils.Claims{
		UserID:       "user-123",
		Role:         "admin",
		TokenVersion: 1,
		TokenType:    string(utils.AccessToken),
		JTI:          "jti-123",
		SuperAdmin:   superAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(utils.GetJWTSecret())
	return tokenString
}

func setupOpsRouter(t *testing.T, backend OpsBackend, auditLog audit.Logger) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_ = logger.Init("development")
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")

	r := gin.New()
	opsHandler := NewOpsHandler(backend, auditLog)

	api := r.Group("/api/catalog")
	adminOps := api.Group("/admin")
	adminOps.Use(middleware.JWTAuth())
	adminOps.Use(middleware.RequireSuperAdmin())
	{
		opsHandler.RegisterRoutes(adminOps)
	}

	return r
}

func TestOpsHandler_GetStatus_RequiresSuperAdmin(t *testing.T) {
	backend := &mockOpsBackend{status: `{"mode":"normal","current":"20260925-100000"}`}
	auditLog := &mockAuditLogger{}
	r := setupOpsRouter(t, backend, auditLog)

	regularToken := createTestToken(false)
	superToken := createTestToken(true)

	// 1. Regular admin should be 403 Forbidden
	req := httptest.NewRequest(http.MethodGet, "/api/catalog/admin/ops/status", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 2. Super admin should be 200 OK
	req2 := httptest.NewRequest(http.MethodGet, "/api/catalog/admin/ops/status", nil)
	req2.Header.Set("Authorization", "Bearer "+superToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "20260925-100000")
}

func TestOpsHandler_Commands_EnqueuedAndAudited(t *testing.T) {
	backend := &mockOpsBackend{}
	auditLog := &mockAuditLogger{}
	r := setupOpsRouter(t, backend, auditLog)
	superToken := createTestToken(true)

	cases := []struct {
		method string
		path   string
		action string
	}{
		{http.MethodPost, "/api/catalog/admin/ops/begin-edit", "begin_edit"},
		{http.MethodPost, "/api/catalog/admin/ops/save", "save"},
		{http.MethodPost, "/api/catalog/admin/ops/cancel-edit", "cancel_edit"},
		{http.MethodPost, "/api/catalog/admin/ops/restore-now", "restore_now"},
		{http.MethodPost, "/api/catalog/admin/ops/restore/20260924-120000", "restore_version"},
	}

	for i, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+superToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code, "case %s %s should be 202", tc.method, tc.path)

		require.Len(t, backend.pushed, i+1)
		var cmd opsCommand
		err := json.Unmarshal([]byte(backend.pushed[i]), &cmd)
		require.NoError(t, err)
		assert.Equal(t, tc.action, cmd.Action)
		assert.Equal(t, "user-123", cmd.RequestedBy)

		if tc.action == "restore_version" {
			assert.Equal(t, "20260924-120000", cmd.Version)
		}

		require.Len(t, auditLog.events, i+1)
		assert.Equal(t, "baseline."+tc.action, auditLog.events[i].Action)
		assert.Equal(t, "user-123", auditLog.events[i].ActorID)
	}
}

func TestOpsHandler_RestoreVersion_MalformedVersion_Returns400(t *testing.T) {
	backend := &mockOpsBackend{}
	r := setupOpsRouter(t, backend, &mockAuditLogger{})
	superToken := createTestToken(true)

	for _, version := range []string{"..", "current", "20260924-12000", "20260924_120000"} {
		req := httptest.NewRequest(http.MethodPost, "/api/catalog/admin/ops/restore/"+version, nil)
		req.Header.Set("Authorization", "Bearer "+superToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "version %q", version)
	}
	assert.Empty(t, backend.pushed)
}
