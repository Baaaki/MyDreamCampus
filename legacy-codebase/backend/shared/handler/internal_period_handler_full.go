package handler

import (
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/dto"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// InternalPeriodHandlerFull provides internal period endpoints for services
// that use the full PeriodRepository (with course_id support, e.g. grades).
// Internal endpoint: called by catalog-service during semester setup to create
// this service's period. Protected by X-Internal-Secret header.
// Not exposed to external clients.
type InternalPeriodHandlerFull struct {
	repo *repository.PeriodRepository
}

func NewInternalPeriodHandlerFull(repo *repository.PeriodRepository) *InternalPeriodHandlerFull {
	return &InternalPeriodHandlerFull{repo: repo}
}

// RegisterRoutes mounts internal period endpoints.
func (h *InternalPeriodHandlerFull) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/periods", h.CreatePeriod)
	rg.DELETE("/periods/by-semester/:semester", h.DeletePeriodBySemester)
	rg.PUT("/periods/by-semester/:semester", h.UpdatePeriodBySemester)
}

// CreatePeriod handles POST /internal/periods — creates a global period (no course_id).
func (h *InternalPeriodHandlerFull) CreatePeriod(c *gin.Context) {
	var req dto.SimpleCreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid internal period request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	if req.PeriodEnd.Before(req.PeriodStart) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "period_end must be after period_start",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	period, err := h.repo.CreatePeriod(c.Request.Context(), repository.AcademicPeriod{
		Semester:    req.Semester,
		CourseID:    nil, // global period
		PeriodStart: req.PeriodStart,
		PeriodEnd:   req.PeriodEnd,
		IsActive:    true,
	})
	if err != nil {
		logger.Error("failed to create period via internal endpoint", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{
			"error": "a period for this semester already exists",
			"code":  "CONFLICT",
		})
		return
	}

	logger.Info("period created via internal endpoint",
		zap.String("semester", period.Semester),
		zap.String("period_id", period.ID.String()),
	)

	c.JSON(http.StatusCreated, dto.PeriodResponse{
		ID:          period.ID,
		Semester:    period.Semester,
		PeriodStart: period.PeriodStart,
		PeriodEnd:   period.PeriodEnd,
		CourseID:    period.CourseID,
		IsActive:    period.IsActive,
		CreatedAt:   period.CreatedAt,
		UpdatedAt:   period.UpdatedAt,
	})
}

// DeletePeriodBySemester handles DELETE /internal/periods/by-semester/:semester
func (h *InternalPeriodHandlerFull) DeletePeriodBySemester(c *gin.Context) {
	semester := c.Param("semester")
	if semester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester is required", "code": "VALIDATION_ERROR"})
		return
	}

	if err := h.repo.DeletePeriodBySemester(c.Request.Context(), semester); err != nil {
		logger.Error("failed to delete period by semester", zap.Error(err), zap.String("semester", semester))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete period", "code": "INTERNAL_ERROR"})
		return
	}

	c.Status(http.StatusNoContent)
}

type updatePeriodBySemesterRequestFull struct {
	PeriodStart time.Time `json:"period_start" binding:"required"`
	PeriodEnd   time.Time `json:"period_end" binding:"required"`
}

// UpdatePeriodBySemester handles PUT /internal/periods/by-semester/:semester
func (h *InternalPeriodHandlerFull) UpdatePeriodBySemester(c *gin.Context) {
	semester := c.Param("semester")
	if semester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester is required", "code": "VALIDATION_ERROR"})
		return
	}

	var req updatePeriodBySemesterRequestFull
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "code": "VALIDATION_ERROR"})
		return
	}

	if req.PeriodEnd.Before(req.PeriodStart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period_end must be after period_start", "code": "VALIDATION_ERROR"})
		return
	}

	period, err := h.repo.UpdatePeriodBySemester(c.Request.Context(), semester, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		logger.Error("failed to update period by semester", zap.Error(err), zap.String("semester", semester))
		c.JSON(http.StatusNotFound, gin.H{"error": "period not found for semester", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, dto.PeriodResponse{
		ID:          period.ID,
		Semester:    period.Semester,
		PeriodStart: period.PeriodStart,
		PeriodEnd:   period.PeriodEnd,
		CourseID:    period.CourseID,
		IsActive:    period.IsActive,
		CreatedAt:   period.CreatedAt,
		UpdatedAt:   period.UpdatedAt,
	})
}
