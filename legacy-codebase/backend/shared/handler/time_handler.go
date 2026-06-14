package handler

import (
	"net/http"

	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/baaaki/mydreamcampus/shared/dto"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TimeHandler provides admin endpoints to control the simulated clock.
type TimeHandler struct{}

func NewTimeHandler() *TimeHandler {
	return &TimeHandler{}
}

// RegisterRoutes mounts time-control endpoints under the given router group.
// The caller is responsible for applying RequireAdmin() middleware.
func (h *TimeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	time := rg.Group("/time")
	{
		time.POST("/simulate", h.Simulate)
		time.POST("/reset", h.Reset)
		time.GET("/status", h.Status)
	}
}

// Simulate switches the clock to simulated mode at the given time.
// POST /admin/time/simulate
func (h *TimeHandler) Simulate(c *gin.Context) {
	var req dto.SimulateTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid simulate time request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: 'time' field is required (RFC3339 format)",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	clock.Set(req.Time)

	logger.Info("clock switched to simulated mode",
		zap.Time("simulated_time", req.Time),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":        "clock switched to simulated mode",
		"simulated_time": req.Time,
	})
}

// Reset switches the clock back to real system time.
// POST /admin/time/reset
func (h *TimeHandler) Reset(c *gin.Context) {
	clock.Reset()

	logger.Info("clock reset to real time")

	c.JSON(http.StatusOK, gin.H{
		"message":      "clock reset to real time",
		"current_time": clock.Now(),
	})
}

// Status returns the current clock mode and time.
// GET /admin/time/status
func (h *TimeHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, dto.TimeStatusResponse{
		Mode:          string(clock.GetMode()),
		CurrentTime:   clock.Now(),
		SimulatedTime: clock.SimulatedTime(),
	})
}
