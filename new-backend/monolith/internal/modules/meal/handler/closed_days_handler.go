package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/audit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const requestTimeout = 10 * time.Second

// ClosedDaysStore is the set of operations ClosedDaysHandler needs from the
// repository. Declared as an interface so a caching wrapper can be injected.
type ClosedDaysStore interface {
	CreateClosedDay(ctx context.Context, params db.CreateClosedDayParams) (db.ClosedDay, error)
	DeleteClosedDay(ctx context.Context, id pgtype.UUID) error
	ListClosedDays(ctx context.Context, params db.ListClosedDaysParams) ([]db.ClosedDay, error)
	GetClosedDaysByDateRange(ctx context.Context, params db.GetClosedDaysByDateRangeParams) ([]db.ClosedDay, error)
	DeleteClosedDaysBySemester(ctx context.Context, semester string) error
}

type ClosedDaysHandler struct {
	repo        ClosedDaysStore
	logger      *zap.Logger
	auditLogger audit.Logger
}

func NewClosedDaysHandler(repo ClosedDaysStore, logger *zap.Logger, auditLogger audit.Logger) *ClosedDaysHandler {
	return &ClosedDaysHandler{repo: repo, logger: logger, auditLogger: auditLogger}
}

// RegisterRoutes mounts closed days endpoints under the given router group.
func (h *ClosedDaysHandler) RegisterRoutes(rg *gin.RouterGroup) {
	closedDays := rg.Group("/closed-days")
	{
		closedDays.POST("", h.CreateClosedDay)
		closedDays.GET("", h.ListClosedDays)
		closedDays.DELETE("/:id", h.DeleteClosedDay)
	}
}

// RegisterInternalRoutes mounts internal closed days endpoints for service-to-service calls.
func (h *ClosedDaysHandler) RegisterInternalRoutes(rg *gin.RouterGroup) {
	closedDays := rg.Group("/closed-days")
	{
		closedDays.POST("/batch", h.BatchCreateClosedDays)
		closedDays.DELETE("/by-semester/:semester", h.DeleteClosedDaysBySemester)
		closedDays.PUT("/by-semester/:semester", h.UpdateClosedDaysBySemester)
	}
}

type createClosedDayRequest struct {
	Date   string `json:"date" binding:"required"`   // YYYY-MM-DD
	Reason string `json:"reason" binding:"required"` // e.g. "Republic Day", "Eid al-Fitr"
}

type closedDayResponse struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	Reason    string `json:"reason"`
	Semester  string `json:"semester,omitempty"`
	CreatedAt string `json:"created_at"`
}

// CreateClosedDay adds a closed day.
// POST /admin/closed-days
func (h *ClosedDaysHandler) CreateClosedDay(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req createClosedDayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid date format (expected YYYY-MM-DD)",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	closedDay, err := h.repo.CreateClosedDay(ctx, db.CreateClosedDayParams{
		Date:   pgtype.Date{Time: date, Valid: true},
		Reason: req.Reason,
		Semester: pgtype.Text{},
	})
	if err != nil {
		h.logger.Error("failed to create closed day", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{
			"error": "a closed day already exists for this date",
			"code":  "CONFLICT",
		})
		return
	}

	// Audit log
	if h.auditLogger != nil {
		actorID, _ := c.Get("user_id")
		h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      actorID.(string),
			ActorRole:    "admin",
			Action:       "closed_day.created",
			ResourceType: "closed_day",
			ResourceID:   closedDay.ID.String(),
			Details: map[string]any{
				"date":   req.Date,
				"reason": req.Reason,
			},
		})
	}

	h.logger.Info("closed day created",
		zap.String("date", req.Date),
		zap.String("reason", req.Reason),
	)

	c.JSON(http.StatusCreated, toClosedDayResponse(closedDay))
}

// ListClosedDays lists closed days with optional date range filter.
// GET /admin/closed-days?from=2025-01-01&to=2025-12-31
func (h *ClosedDaysHandler) ListClosedDays(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var fromDate, toDate pgtype.Date

	if from := c.Query("from"); from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid 'from' date format (expected YYYY-MM-DD)",
				"code":  "VALIDATION_ERROR",
			})
			return
		}
		fromDate = pgtype.Date{Time: t, Valid: true}
	}

	if to := c.Query("to"); to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid 'to' date format (expected YYYY-MM-DD)",
				"code":  "VALIDATION_ERROR",
			})
			return
		}
		toDate = pgtype.Date{Time: t, Valid: true}
	}

	days, err := h.repo.ListClosedDays(ctx, db.ListClosedDaysParams{
		Column1: fromDate,
		Column2: toDate,
	})
	if err != nil {
		h.logger.Error("failed to list closed days", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list closed days",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	response := make([]closedDayResponse, 0, len(days))
	for _, d := range days {
		response = append(response, toClosedDayResponse(d))
	}

	c.JSON(http.StatusOK, response)
}

// DeleteClosedDay removes a closed day.
// DELETE /admin/closed-days/:id
func (h *ClosedDaysHandler) DeleteClosedDay(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	idStr := c.Param("id")

	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid closed day ID",
			"code":  "INVALID_ID",
		})
		return
	}

	if err := h.repo.DeleteClosedDay(ctx, id); err != nil {
		h.logger.Error("failed to delete closed day", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete closed day",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	// Audit log
	if h.auditLogger != nil {
		actorID, _ := c.Get("user_id")
		h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      actorID.(string),
			ActorRole:    "admin",
			Action:       "closed_day.deleted",
			ResourceType: "closed_day",
			ResourceID:   idStr,
		})
	}

	h.logger.Info("closed day deleted", zap.String("id", idStr))

	c.JSON(http.StatusOK, gin.H{
		"message": "closed day deleted successfully",
	})
}

type batchCreateClosedDaysRequest struct {
	ClosedDays []createClosedDayRequest `json:"closed_days" binding:"required,min=1"`
	Semester   string                   `json:"semester"`
}

type batchCreateClosedDaysResponse struct {
	Created []closedDayResponse `json:"created"`
	Skipped []string            `json:"skipped"`
}

// BatchCreateClosedDays adds multiple closed days at once, skipping duplicates.
// POST /internal/closed-days/batch
func (h *ClosedDaysHandler) BatchCreateClosedDays(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req batchCreateClosedDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	var created []closedDayResponse
	var skipped []string

	for _, entry := range req.ClosedDays {
		date, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			skipped = append(skipped, entry.Date+" (invalid format)")
			continue
		}

		closedDay, err := h.repo.CreateClosedDay(ctx, db.CreateClosedDayParams{
			Date:     pgtype.Date{Time: date, Valid: true},
			Reason:   entry.Reason,
			Semester: pgtype.Text{String: req.Semester, Valid: req.Semester != ""},
		})
		if err != nil {
			// Duplicate date — skip silently
			skipped = append(skipped, entry.Date+" (already exists)")
			continue
		}

		created = append(created, toClosedDayResponse(closedDay))
	}

	h.logger.Info("batch closed days processed",
		zap.Int("created", len(created)),
		zap.Int("skipped", len(skipped)),
	)

	c.JSON(http.StatusCreated, batchCreateClosedDaysResponse{
		Created: created,
		Skipped: skipped,
	})
}

// DeleteClosedDaysBySemester removes all closed days for a semester.
// DELETE /internal/closed-days/by-semester/:semester
func (h *ClosedDaysHandler) DeleteClosedDaysBySemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester := c.Param("semester")
	if semester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester is required", "code": "VALIDATION_ERROR"})
		return
	}

	if err := h.repo.DeleteClosedDaysBySemester(ctx, semester); err != nil {
		h.logger.Error("failed to delete closed days by semester", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete closed days", "code": "INTERNAL_ERROR"})
		return
	}

	c.Status(http.StatusNoContent)
}

type updateClosedDaysBySemesterRequest struct {
	ClosedDays []createClosedDayRequest `json:"closed_days" binding:"required"`
}

// UpdateClosedDaysBySemester replaces all closed days for a semester.
// PUT /internal/closed-days/by-semester/:semester
func (h *ClosedDaysHandler) UpdateClosedDaysBySemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester := c.Param("semester")
	if semester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester is required", "code": "VALIDATION_ERROR"})
		return
	}

	var req updateClosedDaysBySemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "code": "VALIDATION_ERROR"})
		return
	}

	// Delete existing closed days for this semester
	if err := h.repo.DeleteClosedDaysBySemester(ctx, semester); err != nil {
		h.logger.Error("failed to delete existing closed days", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update closed days", "code": "INTERNAL_ERROR"})
		return
	}

	// Insert new closed days
	var created []closedDayResponse
	for _, entry := range req.ClosedDays {
		date, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			continue
		}

		closedDay, err := h.repo.CreateClosedDay(ctx, db.CreateClosedDayParams{
			Date:     pgtype.Date{Time: date, Valid: true},
			Reason:   entry.Reason,
			Semester: pgtype.Text{String: semester, Valid: true},
		})
		if err != nil {
			continue
		}
		created = append(created, toClosedDayResponse(closedDay))
	}

	c.JSON(http.StatusOK, gin.H{"created": created})
}

func toClosedDayResponse(d db.ClosedDay) closedDayResponse {
	return closedDayResponse{
		ID:        d.ID.String(),
		Date:      d.Date.Time.Format("2006-01-02"),
		Reason:    d.Reason,
		Semester:  d.Semester.String,
		CreatedAt: d.CreatedAt.Time.Format(time.RFC3339),
	}
}
