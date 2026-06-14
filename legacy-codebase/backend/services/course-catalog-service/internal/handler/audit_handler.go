package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/audit"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuditHandler struct {
	repo *repository.AuditRepository
}

func NewAuditHandler(repo *repository.AuditRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

// RegisterAdminRoutes mounts admin audit log endpoints.
func (h *AuditHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.GET("/audit-log", h.ListAuditLog)
}

// RegisterInternalRoutes mounts internal audit log endpoints (no auth).
func (h *AuditHandler) RegisterInternalRoutes(rg *gin.RouterGroup) {
	rg.POST("/audit-log", h.CreateAuditLog)
}

// ListAuditLog handles GET /admin/audit-log
func (h *AuditHandler) ListAuditLog(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuditHandler"),
		zap.String("method", "ListAuditLog"),
	)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	service := c.Query("service")
	action := c.Query("action")
	actorIDStr := c.Query("actor_id")

	params := db.ListAuditLogParams{
		Limit:   int32(limit),
		Offset:  int32(offset),
		Service: utils.StringToPgText(service),
		Action:  utils.StringToPgText(action),
	}

	if actorIDStr != "" {
		actorID, err := uuid.Parse(actorIDStr)
		if err == nil {
			params.ActorID = utils.UUIDToPgtype(actorID)
		}
	}

	logs, err := h.repo.ListAuditLog(ctx, params)
	if err != nil {
		log.Error("failed to list audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs", "code": "INTERNAL_ERROR"})
		return
	}

	countParams := db.CountAuditLogParams{
		Service: params.Service,
		Action:  params.Action,
		ActorID: params.ActorID,
	}

	total, err := h.repo.CountAuditLog(ctx, countParams)
	if err != nil {
		log.Error("failed to count audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count audit logs", "code": "INTERNAL_ERROR"})
		return
	}

	var result []auditLogResponse
	for _, l := range logs {
		result = append(result, toAuditLogResponse(l))
	}
	if result == nil {
		result = []auditLogResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  result,
		"total": total,
	})
}

// CreateAuditLog handles POST /internal/audit-log
func (h *AuditHandler) CreateAuditLog(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AuditHandler"),
		zap.String("method", "CreateAuditLog"),
	)

	var event audit.AuditEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var details []byte
	if event.Details != nil {
		var err error
		details, err = json.Marshal(event.Details)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid details"})
			return
		}
	}

	params := db.InsertAuditLogParams{
		Service:      event.Service,
		ActorRole:    event.ActorRole,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		Details:      details,
	}

	if event.ActorID != "" {
		parsed, err := uuid.Parse(event.ActorID)
		if err == nil {
			params.ActorID = utils.UUIDToPgtype(parsed)
		}
	}

	if event.ResourceID != "" {
		parsed, err := uuid.Parse(event.ResourceID)
		if err == nil {
			params.ResourceID = utils.UUIDToPgtype(parsed)
		}
	}

	_, err := h.repo.InsertAuditLog(ctx, params)
	if err != nil {
		log.Error("failed to insert audit log", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert audit log"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "audit log created"})
}

type auditLogResponse struct {
	ID           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	Service      string `json:"service"`
	ActorID      string `json:"actor_id"`
	ActorRole    string `json:"actor_role"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Details      any    `json:"details,omitempty"`
}

func toAuditLogResponse(l db.AuditLog) auditLogResponse {
	resp := auditLogResponse{
		ID:           utils.PgtypeToUUIDString(l.ID),
		Timestamp:    utils.PgTimestamptzToTime(l.Timestamp).Format(time.RFC3339),
		Service:      l.Service,
		ActorID:      utils.PgtypeToUUIDString(l.ActorID),
		ActorRole:    l.ActorRole,
		Action:       l.Action,
		ResourceType: l.ResourceType,
		ResourceID:   utils.PgtypeToUUIDString(l.ResourceID),
	}

	if len(l.Details) > 0 {
		var details any
		if err := json.Unmarshal(l.Details, &details); err == nil {
			resp.Details = details
		}
	}

	return resp
}
