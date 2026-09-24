package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/clocksync"
	"github.com/baaaki/mydreamcampus/shared/platform/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// maxSimulationYears bounds the time machine in both directions. Beyond it
// seed data (semesters, schedules) no longer means anything.
const maxSimulationYears = 2

// TimeStatus answers GET /admin/time/status with this service's clock.
// httpserver mounts it on every service; the caller applies JWT + admin.
func TimeStatus(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, timeStatus(service))
	}
}

func timeStatus(service string) dto.TimeStatusResponse {
	state := clock.State()
	resp := dto.TimeStatusResponse{
		Service:       service,
		Active:        state.Active,
		CurrentTime:   state.Now,
		RealTime:      state.Now.Add(-state.Offset),
		OffsetSeconds: int64(state.Offset / time.Second),
	}
	if !state.Until.IsZero() {
		until := state.Until
		resp.Until = &until
	}
	return resp
}

// TimeControlHandler moves the clock of every service at once. Only catalog
// mounts it: one owner for the setting and one audit trail for it.
type TimeControlHandler struct {
	service     string
	backend     clocksync.Backend
	auditLogger audit.Logger
}

// NewTimeControlHandler takes a nil backend when Redis is unavailable; the
// endpoints then answer 503 rather than move only this service's clock.
func NewTimeControlHandler(service string, backend clocksync.Backend, auditLogger audit.Logger) *TimeControlHandler {
	return &TimeControlHandler{service: service, backend: backend, auditLogger: auditLogger}
}

// RegisterRoutes mounts simulate and reset under the given router group.
// The caller is responsible for applying RequireAdmin() middleware.
func (h *TimeControlHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tm := rg.Group("/time")
	{
		tm.POST("/simulate", h.Simulate)
		tm.POST("/reset", h.Reset)
	}
}

// Simulate shifts every service's clock so that it now reads req.Time.
// POST /admin/time/simulate
func (h *TimeControlHandler) Simulate(c *gin.Context) {
	var req dto.SimulateTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid simulate time request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Geçersiz istek: 'time' alanı RFC3339 biçiminde zorunludur",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	now := time.Now()
	if req.Time.Before(now.AddDate(-maxSimulationYears, 0, 0)) || req.Time.After(now.AddDate(maxSimulationYears, 0, 0)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Simüle zaman gerçek zamandan en fazla 2 yıl ileri veya geri olabilir",
			"code":  "VALIDATION_ERROR",
		})
		return
	}
	if h.backend == nil {
		h.unavailable(c, nil)
		return
	}

	actorID := c.GetString("user_id")
	state := clocksync.State{
		OffsetSeconds: int64(req.Time.Sub(now) / time.Second),
		SetAt:         now,
		SetBy:         actorID,
	}
	if err := clocksync.Publish(c.Request.Context(), h.backend, state); err != nil {
		h.unavailable(c, err)
		return
	}

	h.audit(c, "time.simulated", map[string]any{
		"simulated_time": req.Time.Format(time.RFC3339),
		"offset_seconds": state.OffsetSeconds,
	})
	logger.Info("time machine set",
		zap.Time("simulated_time", req.Time),
		zap.Int64("offset_seconds", state.OffsetSeconds),
		zap.String("actor_id", actorID),
	)
	c.JSON(http.StatusOK, timeStatus(h.service))
}

// Reset returns every service to the real clock.
// POST /admin/time/reset
func (h *TimeControlHandler) Reset(c *gin.Context) {
	if h.backend == nil {
		h.unavailable(c, nil)
		return
	}
	if err := clocksync.Clear(c.Request.Context(), h.backend); err != nil {
		h.unavailable(c, err)
		return
	}

	h.audit(c, "time.reset", nil)
	logger.Info("time machine reset", zap.String("actor_id", c.GetString("user_id")))
	c.JSON(http.StatusOK, timeStatus(h.service))
}

func (h *TimeControlHandler) unavailable(c *gin.Context, err error) {
	if err == nil {
		err = errors.New("redis unavailable at startup")
	}
	logger.Error("time machine state not shared", zap.Error(err))
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "Zaman makinesi şu anda kullanılamıyor, lütfen daha sonra tekrar deneyin",
		"code":  "SERVICE_UNAVAILABLE",
	})
}

func (h *TimeControlHandler) audit(c *gin.Context, action string, details map[string]any) {
	if h.auditLogger == nil {
		return
	}
	if err := h.auditLogger.Log(c.Request.Context(), audit.AuditEvent{
		ActorID:      c.GetString("user_id"),
		ActorRole:    c.GetString("role"),
		Action:       action,
		ResourceType: "time_machine",
		Details:      details,
	}); err != nil {
		logger.Warn("audit log write failed", zap.Error(err))
	}
}
