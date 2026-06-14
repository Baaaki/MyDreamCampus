package handler

import (
	"net/http"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/audit"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/semester"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// SimplePeriodHandler provides admin endpoints for managing academic periods
// in services that don't need course-specific overrides (catalog, enrollment).
type SimplePeriodHandler struct {
	repo            *repository.SimplePeriodRepository
	semesterChecker semester.Checker
	auditLogger     audit.Logger
}

func NewSimplePeriodHandler(repo *repository.SimplePeriodRepository, checker semester.Checker, auditLogger audit.Logger) *SimplePeriodHandler {
	return &SimplePeriodHandler{repo: repo, semesterChecker: checker, auditLogger: auditLogger}
}

// RegisterRoutes mounts period CRUD endpoints under the given router group.
func (h *SimplePeriodHandler) RegisterRoutes(rg *gin.RouterGroup) {
	periods := rg.Group("/periods")
	{
		periods.POST("", h.CreatePeriod)
		periods.GET("", h.ListPeriods)
		periods.PUT("/:id", h.UpdatePeriod)
		periods.DELETE("/:id", h.DeletePeriod)
	}
}

// CreatePeriod creates a new academic period.
// POST /admin/periods
func (h *SimplePeriodHandler) CreatePeriod(c *gin.Context) {
	var req dto.SimpleCreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid create period request", zap.Error(err))
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

	// Check semester is active
	if h.semesterChecker != nil {
		active, err := h.semesterChecker.IsSemesterActive(c.Request.Context(), req.Semester)
		if err != nil {
			logger.Warn("semester status check failed", zap.Error(err))
		} else if !active {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "semester is not active — modifications are not allowed",
				"code":  "SEMESTER_NOT_ACTIVE",
			})
			return
		}
	}

	period, err := h.repo.CreatePeriod(c.Request.Context(), repository.SimplePeriod{
		Semester:    req.Semester,
		PeriodStart: req.PeriodStart,
		PeriodEnd:   req.PeriodEnd,
		IsActive:    true,
	})
	if err != nil {
		logger.Error("failed to create academic period", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{
			"error": "a period for this semester already exists",
			"code":  "CONFLICT",
		})
		return
	}

	// Audit log
	if h.auditLogger != nil {
		actorID, _ := c.Get("user_id")
		h.auditLogger.Log(c.Request.Context(), audit.AuditEvent{
			ActorID:      actorID.(string),
			ActorRole:    "admin",
			Action:       "period.created",
			ResourceType: "academic_period",
			ResourceID:   period.ID.String(),
			Details: map[string]any{
				"semester":     req.Semester,
				"period_start": req.PeriodStart.Format("2006-01-02T15:04:05Z07:00"),
				"period_end":   req.PeriodEnd.Format("2006-01-02T15:04:05Z07:00"),
			},
		})
	}

	logger.Info("academic period created",
		zap.String("semester", period.Semester),
		zap.String("period_id", period.ID.String()),
	)

	c.JSON(http.StatusCreated, toSimplePeriodResponse(period))
}

// ListPeriods lists academic periods, optionally filtered by semester.
// GET /admin/periods?semester=2025-2026-Fall
func (h *SimplePeriodHandler) ListPeriods(c *gin.Context) {
	semester := c.Query("semester")

	var periods []repository.SimplePeriod
	var err error

	if semester != "" {
		periods, err = h.repo.GetPeriodsBySemester(c.Request.Context(), semester)
	} else {
		periods, err = h.repo.GetAllPeriods(c.Request.Context())
	}

	if err != nil {
		logger.Error("failed to list academic periods", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list periods",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	response := make([]dto.SimplePeriodResponse, 0, len(periods))
	for _, p := range periods {
		response = append(response, toSimplePeriodResponse(&p))
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePeriod updates a period's end date and/or active status.
// PUT /admin/periods/:id
func (h *SimplePeriodHandler) UpdatePeriod(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid period ID",
			"code":  "INVALID_ID",
		})
		return
	}

	var req dto.UpdatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	// Fetch existing period to check semester
	existing, err := h.repo.GetPeriodByID(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "period not found", "code": "NOT_FOUND"})
			return
		}
		logger.Error("failed to get period", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get period", "code": "INTERNAL_ERROR"})
		return
	}

	// Check semester is active
	if h.semesterChecker != nil {
		active, checkErr := h.semesterChecker.IsSemesterActive(c.Request.Context(), existing.Semester)
		if checkErr != nil {
			logger.Warn("semester status check failed", zap.Error(checkErr))
		} else if !active {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "semester is not active — modifications are not allowed",
				"code":  "SEMESTER_NOT_ACTIVE",
			})
			return
		}
	}

	period, err := h.repo.UpdatePeriod(c.Request.Context(), id, req.PeriodEnd, req.IsActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "period not found",
				"code":  "NOT_FOUND",
			})
			return
		}
		logger.Error("failed to update academic period", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update period",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	// Audit log
	if h.auditLogger != nil {
		actorID, _ := c.Get("user_id")
		h.auditLogger.Log(c.Request.Context(), audit.AuditEvent{
			ActorID:      actorID.(string),
			ActorRole:    "admin",
			Action:       "period.updated",
			ResourceType: "academic_period",
			ResourceID:   id.String(),
			Details: map[string]any{
				"semester": existing.Semester,
			},
		})
	}

	logger.Info("academic period updated",
		zap.String("period_id", id.String()),
	)

	c.JSON(http.StatusOK, toSimplePeriodResponse(period))
}

// DeletePeriod removes a period.
// DELETE /admin/periods/:id
func (h *SimplePeriodHandler) DeletePeriod(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid period ID",
			"code":  "INVALID_ID",
		})
		return
	}

	// Fetch existing period to check semester
	existing, err := h.repo.GetPeriodByID(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "period not found", "code": "NOT_FOUND"})
			return
		}
		logger.Error("failed to get period", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get period", "code": "INTERNAL_ERROR"})
		return
	}

	// Check semester is active
	if h.semesterChecker != nil {
		active, checkErr := h.semesterChecker.IsSemesterActive(c.Request.Context(), existing.Semester)
		if checkErr != nil {
			logger.Warn("semester status check failed", zap.Error(checkErr))
		} else if !active {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "semester is not active — modifications are not allowed",
				"code":  "SEMESTER_NOT_ACTIVE",
			})
			return
		}
	}

	if err := h.repo.DeletePeriod(c.Request.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "period not found",
				"code":  "NOT_FOUND",
			})
			return
		}
		logger.Error("failed to delete academic period", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete period",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	// Audit log
	if h.auditLogger != nil {
		actorID, _ := c.Get("user_id")
		h.auditLogger.Log(c.Request.Context(), audit.AuditEvent{
			ActorID:      actorID.(string),
			ActorRole:    "admin",
			Action:       "period.deleted",
			ResourceType: "academic_period",
			ResourceID:   id.String(),
			Details: map[string]any{
				"semester": existing.Semester,
			},
		})
	}

	logger.Info("academic period deleted",
		zap.String("period_id", id.String()),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "period deleted successfully",
	})
}

func toSimplePeriodResponse(p *repository.SimplePeriod) dto.SimplePeriodResponse {
	return dto.SimplePeriodResponse{
		ID:          p.ID,
		Semester:    p.Semester,
		PeriodStart: p.PeriodStart,
		PeriodEnd:   p.PeriodEnd,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
