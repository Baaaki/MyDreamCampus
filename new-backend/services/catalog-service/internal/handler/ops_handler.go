package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OpsBackend provides Redis-backed ops status read and command dispatch.
type OpsBackend interface {
	GetOpsStatus(ctx context.Context) (string, error)
	PushOpsCommand(ctx context.Context, cmdJSON string) error
}

// OpsHandler manages superadmin operations on demo baseline and permanent state.
type OpsHandler struct {
	backend     OpsBackend
	auditLogger audit.Logger
}

func NewOpsHandler(backend OpsBackend, auditLogger audit.Logger) *OpsHandler {
	return &OpsHandler{backend: backend, auditLogger: auditLogger}
}

type opsCommand struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Version     string `json:"version,omitempty"`
	RequestedBy string `json:"requested_by"`
	RequestedAt string `json:"requested_at"`
}

// RegisterRoutes mounts the /ops endpoints. The caller applies JWT, CSRF, and RequireSuperAdmin middleware.
func (h *OpsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ops := rg.Group("/ops")
	{
		ops.GET("/status", h.GetStatus)
		ops.POST("/begin-edit", h.BeginEdit)
		ops.POST("/save", h.Save)
		ops.POST("/cancel-edit", h.CancelEdit)
		ops.POST("/restore-now", h.RestoreNow)
		ops.POST("/restore/:version", h.RestoreVersion)
	}
}

// GetStatus returns the current ops status from Redis.
// GET /api/catalog/admin/ops/status
func (h *OpsHandler) GetStatus(c *gin.Context) {
	if h.backend == nil {
		c.JSON(http.StatusOK, gin.H{
			"mode":          "normal",
			"current":       "",
			"versions":      []string{},
			"edit_deadline": nil,
			"last_action":   "",
			"last_error":    nil,
			"updated_at":    "",
		})
		return
	}

	raw, err := h.backend.GetOpsStatus(c.Request.Context())
	if err != nil {
		logger.Error("failed to get ops status from redis", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Operasyon durumu alınamadı",
			"code":  "SERVICE_UNAVAILABLE",
		})
		return
	}

	if raw == "" {
		c.JSON(http.StatusOK, gin.H{
			"mode":          "normal",
			"current":       "",
			"versions":      []string{},
			"edit_deadline": nil,
			"last_action":   "",
			"last_error":    nil,
			"updated_at":    "",
		})
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(raw))
}

func (h *OpsHandler) queueCommand(c *gin.Context, action, version string) {
	if h.backend == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Redis operasyon arka ucu yapılandırılmamış",
			"code":  "SERVICE_UNAVAILABLE",
		})
		return
	}

	actorID := c.GetString("user_id")
	actorRole := c.GetString("role")

	cmd := opsCommand{
		ID:          uuid.New().String(),
		Action:      action,
		Version:     version,
		RequestedBy: actorID,
		RequestedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Komut oluşturulamadı",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	if err := h.backend.PushOpsCommand(c.Request.Context(), string(data)); err != nil {
		logger.Error("failed to push ops command to redis", zap.Error(err), zap.String("action", action))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Komut iş kuyruğuna iletilemedi",
			"code":  "SERVICE_UNAVAILABLE",
		})
		return
	}

	if h.auditLogger != nil {
		details := map[string]any{"command_id": cmd.ID}
		if version != "" {
			details["version"] = version
		}
		_ = h.auditLogger.Log(c.Request.Context(), audit.AuditEvent{
			Service:      "catalog",
			ActorID:      actorID,
			ActorRole:    actorRole,
			Action:       "baseline." + action,
			ResourceType: "baseline",
			ResourceID:   version,
			Details:      details,
		})
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "İşlem kuyruğa alındı",
		"command_id": cmd.ID,
		"action":     action,
	})
}

// BeginEdit puts the system into editing mode and restores to the latest baseline.
// POST /api/catalog/admin/ops/begin-edit
func (h *OpsHandler) BeginEdit(c *gin.Context) {
	h.queueCommand(c, "begin_edit", "")
}

// Save takes a new baseline snapshot and releases the write lock.
// POST /api/catalog/admin/ops/save
func (h *OpsHandler) Save(c *gin.Context) {
	h.queueCommand(c, "save", "")
}

// CancelEdit discards changes made during editing mode and restores to the baseline.
// POST /api/catalog/admin/ops/cancel-edit
func (h *OpsHandler) CancelEdit(c *gin.Context) {
	h.queueCommand(c, "cancel_edit", "")
}

// RestoreNow restores all databases immediately to the current baseline.
// POST /api/catalog/admin/ops/restore-now
func (h *OpsHandler) RestoreNow(c *gin.Context) {
	h.queueCommand(c, "restore_now", "")
}

// RestoreVersion restores all databases to the specified past baseline version.
// POST /api/catalog/admin/ops/restore/:version
func (h *OpsHandler) RestoreVersion(c *gin.Context) {
	version := c.Param("version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Sürüm parametresi zorunludur",
			"code":  "VALIDATION_ERROR",
		})
		return
	}
	h.queueCommand(c, "restore_version", version)
}
