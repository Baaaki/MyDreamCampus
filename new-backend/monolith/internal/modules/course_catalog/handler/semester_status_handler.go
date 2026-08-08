package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var semesterNameRegex = regexp.MustCompile(`^(\d{4})-(\d{4})-(Fall|Spring)$`)

// isValidSemesterName validates semester name format and year consistency.
// Second year must be exactly one more than the first (e.g., 2025-2026-Fall).
func isValidSemesterName(name string) bool {
	matches := semesterNameRegex.FindStringSubmatch(name)
	if matches == nil {
		return false
	}

	startYear, err1 := strconv.Atoi(matches[1])
	endYear, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return false
	}

	if endYear != startYear+1 {
		return false
	}
	if startYear < 2000 || startYear > 2100 {
		return false
	}

	return true
}

type SemesterStatusHandler struct {
	repo        *repository.SemesterStatusRepository
	periodRepo  *sharedRepo.SimplePeriodRepository
	auditLogger audit.Logger
	mealClient  service.MealClient
	pool        *pgxpool.Pool
}

func NewSemesterStatusHandler(repo *repository.SemesterStatusRepository, periodRepo *sharedRepo.SimplePeriodRepository, auditLogger audit.Logger, mealClient service.MealClient, pool *pgxpool.Pool) *SemesterStatusHandler {
	return &SemesterStatusHandler{repo: repo, periodRepo: periodRepo, auditLogger: auditLogger, mealClient: mealClient, pool: pool}
}

// toClosedDays converts the request rows into the client's wire type.
func toClosedDays(entries []closedDayEntry) []service.ClosedDay {
	out := make([]service.ClosedDay, 0, len(entries))
	for _, e := range entries {
		out = append(out, service.ClosedDay{Date: e.Date, Reason: e.Reason})
	}
	return out
}

// RegisterRoutes mounts semester status management endpoints under admin group.
func (h *SemesterStatusHandler) RegisterRoutes(rg *gin.RouterGroup) {
	semesters := rg.Group("/semesters")
	{
		semesters.POST("", h.CreateSemester)
		semesters.GET("", h.ListSemesters)
		semesters.GET("/active", h.GetActiveSemester)
		semesters.PUT("/:id/activate", h.ActivateSemester)
		semesters.PUT("/:id/complete", h.CompleteSemester)
		semesters.DELETE("/:id", h.DeletePlannedSemester)
		semesters.PUT("/:id", h.UpdatePlannedSemester)
	}
}

// RegisterInternalRoutes mounts internal endpoints for service-to-service communication.
func (h *SemesterStatusHandler) RegisterInternalRoutes(rg *gin.RouterGroup) {
	rg.GET("/semesters/:name/status", h.IsSemesterActive)
	rg.GET("/semesters/:name/info", h.GetSemesterInfo)
	rg.POST("/periods/republish", h.RepublishPeriods)
}

type periodTimeRange struct {
	Start time.Time `json:"start" binding:"required"`
	End   time.Time `json:"end" binding:"required"`
}

type semesterPeriods struct {
	Enrollment *periodTimeRange `json:"enrollment,omitempty"`
	Grading    *periodTimeRange `json:"grading,omitempty"`
	Attendance *periodTimeRange `json:"attendance,omitempty"`
	Catalog    *periodTimeRange `json:"catalog,omitempty"`
}

type closedDayEntry struct {
	Date   string `json:"date" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type createSemesterRequest struct {
	Name         string           `json:"name" binding:"required"`
	HardDeadline time.Time        `json:"hard_deadline" binding:"required"`
	Periods      *semesterPeriods `json:"periods,omitempty"`
	ClosedDays   []closedDayEntry `json:"closed_days,omitempty"`
}

type semesterResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	HardDeadline string  `json:"hard_deadline"`
	ActivatedAt  *string `json:"activated_at,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// CreateSemester handles POST /admin/semesters
func (h *SemesterStatusHandler) CreateSemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "CreateSemester"),
	)

	var req createSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "code": "VALIDATION_ERROR"})
		return
	}

	if !isValidSemesterName(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "semester name must match format: YYYY-YYYY-Fall or YYYY-YYYY-Spring (consecutive years, e.g. 2025-2026-Fall)",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	if req.HardDeadline.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "hard_deadline must be in the future",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	semester, err := h.repo.CreateSemester(ctx, req.Name, req.HardDeadline)
	if err != nil {
		handlerLogger.Error("failed to create semester", zap.Error(err))
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "semester already exists", "code": "CONFLICT"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create semester", "code": "INTERNAL_ERROR"})
		return
	}

	resp := toSemesterResponse(semester)

	// Audit log
	if h.auditLogger != nil {
		if err := h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      getActorID(c),
			ActorRole:    "admin",
			Action:       "semester.created",
			ResourceType: "semester",
			ResourceID:   resp.ID,
			Details: map[string]any{
				"semester_name": req.Name,
				"hard_deadline": req.HardDeadline.Format(time.RFC3339),
			},
		}); err != nil {
			handlerLogger.Warn("audit log write failed", zap.Error(err))
		}
	}

	handlerLogger.Info("semester created", zap.String("name", req.Name))

	// Semester setup records every service's period here and announces the
	// consumer ones as events.
	// This is intentionally NOT a single atomic endpoint — the steps are separated so that
	// in the future, a "department_head" role can handle course creation (step 2)
	// while admin handles semester creation (step 1) and activation (step 3).
	var periodErrors []string

	if req.Periods != nil {
		periodErrors = h.distributePeriods(ctx, req.Name, req.HardDeadline, req.Periods)
	}

	// Distribute closed days to meal service
	if len(req.ClosedDays) > 0 {
		if err := h.distributeClosedDays(ctx, req.ClosedDays); err != nil {
			handlerLogger.Warn("failed to distribute closed days to meal service", zap.Error(err))
			periodErrors = append(periodErrors, fmt.Sprintf("meal (closed_days): %v", err))
		}
	}

	if len(periodErrors) > 0 {
		handlerLogger.Warn("some distributions failed", zap.Any("errors", periodErrors))
		c.JSON(http.StatusCreated, gin.H{
			"semester":      resp,
			"period_errors": periodErrors,
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// periodTarget pairs one period type with the range the admin submitted for it.
type periodTarget struct {
	periodType string
	pr         *periodTimeRange
}

// periodTargets orders the four period types as the request carries them.
func periodTargets(periods *semesterPeriods) []periodTarget {
	return []periodTarget{
		{sharedRepo.PeriodTypeCatalog, periods.Catalog},
		{sharedRepo.PeriodTypeEnrollment, periods.Enrollment},
		{sharedRepo.PeriodTypeGrading, periods.Grading},
		{sharedRepo.PeriodTypeAttendance, periods.Attendance},
	}
}

// acceptPeriodTargets drops the unset types and rejects the ones running past
// the semester's hard deadline, returning one error string per rejection.
func acceptPeriodTargets(targets []periodTarget, hardDeadline time.Time) ([]periodTarget, []string) {
	accepted := make([]periodTarget, 0, len(targets))
	var errs []string
	for _, t := range targets {
		if t.pr == nil {
			continue
		}
		if t.pr.End.After(hardDeadline) {
			errs = append(errs, fmt.Sprintf("%s: period end exceeds hard_deadline", t.periodType))
			continue
		}
		accepted = append(accepted, t)
	}
	return accepted, errs
}

// distributePeriods writes all four period rows into catalog's table — the
// source of truth — and queues one outbox event per consuming service so it
// can refresh its local projection. Rows and events share a transaction: a
// consumer must never be told about a period that was rolled back.
func (h *SemesterStatusHandler) distributePeriods(ctx context.Context, semester string, hardDeadline time.Time, periods *semesterPeriods) []string {
	targets, errs := acceptPeriodTargets(periodTargets(periods), hardDeadline)
	if len(targets) == 0 {
		return errs
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return append(errs, fmt.Sprintf("periods: %v", err))
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	for _, t := range targets {
		period, err := h.periodRepo.CreatePeriodOfTypeTx(ctx, tx, sharedRepo.SimplePeriod{
			Semester:    semester,
			PeriodStart: t.pr.Start,
			PeriodEnd:   t.pr.End,
			IsActive:    true,
		}, t.periodType)
		if err != nil {
			return append(errs, fmt.Sprintf("%s: %v", t.periodType, err))
		}
		// Catalog reads its own row directly, so it needs no event.
		if t.periodType == sharedRepo.PeriodTypeCatalog {
			continue
		}
		if err := queuePeriodEvent(ctx, qtx, period, t.periodType, events.PeriodActionCreated); err != nil {
			return append(errs, fmt.Sprintf("%s: %v", t.periodType, err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return append(errs, fmt.Sprintf("periods: %v", err))
	}
	return errs
}

// updateDistributedPeriods moves the period rows of the types the request
// carries and queues an *.updated event for each consuming service. Same
// transaction rule as distributePeriods.
func (h *SemesterStatusHandler) updateDistributedPeriods(ctx context.Context, semester string, hardDeadline time.Time, periods *semesterPeriods) []string {
	targets, errs := acceptPeriodTargets(periodTargets(periods), hardDeadline)
	if len(targets) == 0 {
		return errs
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return append(errs, fmt.Sprintf("periods: %v", err))
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	for _, t := range targets {
		period, err := h.periodRepo.UpdatePeriodBySemesterAndTypeTx(ctx, tx, semester, t.periodType, t.pr.Start, t.pr.End)
		if err != nil {
			return append(errs, fmt.Sprintf("%s: %v", t.periodType, err))
		}
		if t.periodType == sharedRepo.PeriodTypeCatalog {
			continue
		}
		if err := queuePeriodEvent(ctx, qtx, period, t.periodType, events.PeriodActionUpdated); err != nil {
			return append(errs, fmt.Sprintf("%s: %v", t.periodType, err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return append(errs, fmt.Sprintf("periods: %v", err))
	}
	return errs
}

// RepublishPeriods re-queues every consumer period as an *.updated event.
// Needed once per existing installation: periods created before this endpoint
// existed were never announced, so the consumers' projections start empty and
// their period checks silently degrade to fail-open. Idempotent — consumers
// upsert, so it can be called as often as needed.
// POST /internal/periods/republish
func (h *SemesterStatusHandler) RepublishPeriods(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "RepublishPeriods"),
	)

	periods, err := h.periodRepo.ListProjectedPeriods(ctx)
	if err != nil {
		handlerLogger.Error("failed to list projected periods", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list periods", "code": "INTERNAL_ERROR"})
		return
	}
	if len(periods) == 0 {
		c.JSON(http.StatusOK, gin.H{"republished": 0})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		handlerLogger.Error("failed to begin transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	for _, p := range periods {
		if err := queuePeriodEvent(ctx, qtx, &p.SimplePeriod, p.PeriodType, events.PeriodActionUpdated); err != nil {
			handlerLogger.Error("failed to queue period event", zap.Error(err),
				zap.String("semester", p.Semester), zap.String("period_type", p.PeriodType))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		handlerLogger.Error("failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	handlerLogger.Info("academic periods republished", zap.Int("count", len(periods)))
	c.JSON(http.StatusOK, gin.H{"republished": len(periods)})
}

// queuePeriodEvent writes one period projection event to catalog's outbox.
// The routing key doubles as the event type so each consumer binds only its
// own period type and the broker does the filtering.
func queuePeriodEvent(ctx context.Context, qtx *db.Queries, p *sharedRepo.SimplePeriod, periodType, action string) error {
	payload, err := json.Marshal(map[string]any{
		"id":           p.ID.String(),
		"semester":     p.Semester,
		"period_type":  periodType,
		"period_start": p.PeriodStart.Format(time.RFC3339),
		"period_end":   p.PeriodEnd.Format(time.RFC3339),
		"is_active":    p.IsActive,
	})
	if err != nil {
		return fmt.Errorf("marshal period event payload: %w", err)
	}
	return insertPeriodEvent(ctx, qtx, periodType, action, payload)
}

// queuePeriodDeletedEvent announces that a semester's period is gone. Only the
// semester is carried — consumers delete by semester, and the row is already
// gone on this side.
func queuePeriodDeletedEvent(ctx context.Context, qtx *db.Queries, semester, periodType string) error {
	payload, err := json.Marshal(map[string]any{
		"semester":    semester,
		"period_type": periodType,
	})
	if err != nil {
		return fmt.Errorf("marshal period event payload: %w", err)
	}
	return insertPeriodEvent(ctx, qtx, periodType, events.PeriodActionDeleted, payload)
}

func insertPeriodEvent(ctx context.Context, qtx *db.Queries, periodType, action string, payload []byte) error {
	eventType := events.PeriodEventType(periodType, action)
	if _, err := qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     eventType,
		RoutingKey:    eventType,
		Payload:       payload,
	}); err != nil {
		return fmt.Errorf("queue %s event: %w", eventType, err)
	}
	return nil
}

// distributeClosedDays sends closed days to the meal service.
func (h *SemesterStatusHandler) distributeClosedDays(ctx context.Context, closedDays []closedDayEntry) error {
	if h.mealClient == nil {
		return fmt.Errorf("meal client not configured")
	}
	return h.mealClient.CreateClosedDays(ctx, toClosedDays(closedDays))
}

// ListSemesters handles GET /admin/semesters
func (h *SemesterStatusHandler) ListSemesters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "ListSemesters"),
	)

	semesters, err := h.repo.ListSemesters(ctx)
	if err != nil {
		log.Error("failed to list semesters", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list semesters", "code": "INTERNAL_ERROR"})
		return
	}

	var result []semesterResponse
	for _, s := range semesters {
		result = append(result, toSemesterResponse(s))
	}
	if result == nil {
		result = []semesterResponse{}
	}

	c.JSON(http.StatusOK, result)
}

// GetActiveSemester handles GET /admin/semesters/active
func (h *SemesterStatusHandler) GetActiveSemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester, err := h.repo.GetActiveSemester(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active semester found", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, toSemesterResponse(semester))
}

// ActivateSemester handles PUT /admin/semesters/:id/activate
//
// INVARIANT: Only one semester can be active at any given time.
// Before activating, we check if another semester is already active.
// If yes, the request is rejected with a clear error message.
// This rule exists because the entire system (enrollment, grades, attendance)
// operates on a single active semester — having two active semesters would
// cause ambiguity in which semester students enroll in, teachers grade for, etc.
// The database also enforces this via idx_semesters_single_active partial unique index
// as a safety net against race conditions.
func (h *SemesterStatusHandler) ActivateSemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "ActivateSemester"),
	)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid semester ID", "code": "VALIDATION_ERROR"})
		return
	}

	// INVARIANT: Only one semester can be active at any given time.
	// Check before attempting activation to provide a clear, user-friendly error message.
	hasActive, err := h.repo.HasActiveSemester(ctx)
	if err != nil {
		handlerLogger.Error("failed to check active semester", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check active semester", "code": "INTERNAL_ERROR"})
		return
	}
	if hasActive {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Zaten aktif bir dönem bulunuyor. Yeni bir dönem aktifleştirmek için önce mevcut aktif dönemi tamamlayın.",
			"code":  "ACTIVE_SEMESTER_EXISTS",
		})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		handlerLogger.Error("failed to begin transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)

	// Activate semester (returns the semester row with name)
	semester, err := qtx.ActivateSemester(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		handlerLogger.Error("failed to activate semester", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to activate semester — it may not be in 'planned' status",
			"code":  "INVALID_STATE_TRANSITION",
		})
		return
	}

	// Fetch all courses for this semester
	courses, err := qtx.ListSemesterCoursesForActivation(ctx, semester.Name)
	if err != nil {
		handlerLogger.Error("failed to list semester courses for activation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list courses", "code": "INTERNAL_ERROR"})
		return
	}

	// Create outbox event for each course
	for _, course := range courses {
		payload := buildCourseSemesterCreatedPayload(course)
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			handlerLogger.Error("failed to marshal event payload", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
			return
		}

		_, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			CorrelationID: utils.CorrelationIDFromContext(ctx),
			EventType:     events.EventCourseSemesterCreated,
			RoutingKey:    events.EventCourseSemesterCreated,
			Payload:       payloadJSON,
		})
		if err != nil {
			handlerLogger.Error("failed to create outbox event", zap.Error(err),
				zap.String("course_code", course.CourseCode))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		handlerLogger.Error("failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	resp := toSemesterResponse(semester)

	if h.auditLogger != nil {
		if err := h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      getActorID(c),
			ActorRole:    "admin",
			Action:       "semester.activated",
			ResourceType: "semester",
			ResourceID:   resp.ID,
			Details: map[string]any{
				"semester_name": semester.Name,
				"hard_deadline": semester.HardDeadline.Time.Format(time.RFC3339),
				"courses_count": len(courses),
			},
		}); err != nil {
			handlerLogger.Warn("audit log write failed", zap.Error(err))
		}
	}

	handlerLogger.Info("semester activated",
		zap.String("name", semester.Name),
		zap.Int("courses_count", len(courses)),
	)
	c.JSON(http.StatusOK, resp)
}

// buildCourseSemesterCreatedPayload builds the event payload for course.semester.created
func buildCourseSemesterCreatedPayload(course db.ListSemesterCoursesForActivationRow) map[string]any {
	// Parse schedule sessions from JSON
	var scheduleSessions []map[string]any
	_ = json.Unmarshal(course.ScheduleSessions, &scheduleSessions)

	// Parse assessment schema from JSONB
	var assessmentSchema []map[string]any
	_ = json.Unmarshal(course.AssessmentSchema, &assessmentSchema)

	// Parse prerequisites from JSONB
	var prerequisites []map[string]any
	_ = json.Unmarshal(course.Prerequisites, &prerequisites)

	return map[string]any{
		"event_id":            uuid.New().String(),
		"event_type":          events.EventCourseSemesterCreated,
		"timestamp":           time.Now().Format(time.RFC3339),
		"semester_course_id":  utils.PgtypeToUUIDString(course.ID),
		"semester":            course.Semester,
		"course_code":         course.CourseCode,
		"course_name":         course.CourseName,
		"faculty":             course.Faculty,
		"department":          course.Department,
		"credits":             course.Credits,
		"class_level":         course.ClassLevel,
		"course_type":         string(course.CourseType),
		"instructor_id":       utils.PgtypeToUUIDString(course.InstructorID),
		"instructor_fullname": course.InstructorFullname,
		"classroom_location":  course.ClassroomLocation,
		"max_capacity":        course.MaxCapacity,
		"assessment_schema":   assessmentSchema,
		"prerequisites":       prerequisites,
		"schedule_sessions":   scheduleSessions,
	}
}

// CompleteSemester handles PUT /admin/semesters/:id/complete
func (h *SemesterStatusHandler) CompleteSemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "CompleteSemester"),
	)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid semester ID", "code": "VALIDATION_ERROR"})
		return
	}

	semester, err := h.repo.CompleteSemester(ctx, id)
	if err != nil {
		handlerLogger.Error("failed to complete semester", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to complete semester — it may not be in 'active' status",
			"code":  "INVALID_STATE_TRANSITION",
		})
		return
	}

	resp := toSemesterResponse(semester)

	if h.auditLogger != nil {
		if err := h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      getActorID(c),
			ActorRole:    "admin",
			Action:       "semester.completed",
			ResourceType: "semester",
			ResourceID:   resp.ID,
			Details: map[string]any{
				"semester_name": semester.Name,
			},
		}); err != nil {
			handlerLogger.Warn("audit log write failed", zap.Error(err))
		}
	}

	handlerLogger.Info("semester completed", zap.String("name", semester.Name))
	c.JSON(http.StatusOK, resp)
}

// GetSemesterInfo handles GET /internal/semesters/:name/info
// Returns semester info including hard_deadline for enforcement by other services.
func (h *SemesterStatusHandler) GetSemesterInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	name := c.Param("name")

	info, err := h.repo.GetSemesterInfo(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "semester not found", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":             info.Name,
		"status":           info.Status,
		"hard_deadline":    info.HardDeadline.Format(time.RFC3339),
		"is_past_deadline": info.IsPastDeadline,
	})
}

// IsSemesterActive handles GET /internal/semesters/:name/status
func (h *SemesterStatusHandler) IsSemesterActive(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	name := c.Param("name")

	active, err := h.repo.IsSemesterActive(ctx, name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false, "error": "semester not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"active": active})
}

func toSemesterResponse(s db.Semester) semesterResponse {
	resp := semesterResponse{
		ID:           utils.PgtypeToUUIDString(s.ID),
		Name:         s.Name,
		Status:       string(s.Status),
		HardDeadline: utils.PgTimestamptzToTime(s.HardDeadline).Format(time.RFC3339),
		CreatedAt:    utils.PgTimestamptzToTime(s.CreatedAt).Format(time.RFC3339),
		UpdatedAt:    utils.PgTimestamptzToTime(s.UpdatedAt).Format(time.RFC3339),
	}

	if s.ActivatedAt.Valid {
		t := s.ActivatedAt.Time.Format(time.RFC3339)
		resp.ActivatedAt = &t
	}
	if s.CompletedAt.Valid {
		t := s.CompletedAt.Time.Format(time.RFC3339)
		resp.CompletedAt = &t
	}

	return resp
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// getActorID safely extracts user_id from gin context as a string.
func getActorID(c *gin.Context) string {
	actorID, exists := c.Get("user_id")
	if !exists {
		return "unknown"
	}
	if s, ok := actorID.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", actorID)
}

// DeletePlannedSemester handles DELETE /admin/semesters/:id
func (h *SemesterStatusHandler) DeletePlannedSemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "DeletePlannedSemester"),
	)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid semester ID", "code": "VALIDATION_ERROR"})
		return
	}

	queries := db.New(h.pool)

	// Get semester
	semester, err := queries.GetSemesterByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "semester not found", "code": "NOT_FOUND"})
		return
	}

	if semester.Status != db.CourseCatalogSemesterStatusPlanned {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Sadece planlanmış dönemler silinebilir",
			"code":  "INVALID_STATE",
		})
		return
	}

	// Transaction: delete courses, periods, semester
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		handlerLogger.Error("failed to begin transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}
	// Rollback after successful commit is a no-op returning ErrTxClosed — safe to discard.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)

	// Delete semester courses (CASCADE deletes schedule_sessions)
	if err := qtx.DeleteSemesterCoursesBySemester(ctx, semester.Name); err != nil {
		handlerLogger.Error("failed to delete semester courses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	// Delete every period type for this semester — catalog's own row and the
	// consumer rows alike.
	if err := qtx.DeletePeriodsBySemester(ctx, semester.Name); err != nil {
		handlerLogger.Error("failed to delete periods", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	// Consumers must forget the semester too, or their local projection keeps
	// enforcing a deadline for a semester that no longer exists.
	for _, periodType := range sharedRepo.ProjectedPeriodTypes {
		if err := queuePeriodDeletedEvent(ctx, qtx, semester.Name, periodType); err != nil {
			handlerLogger.Error("failed to queue period deleted event", zap.Error(err),
				zap.String("period_type", periodType))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
			return
		}
	}

	// Delete semester
	if err := qtx.DeletePlannedSemester(ctx, utils.UUIDToPgtype(id)); err != nil {
		handlerLogger.Error("failed to delete semester", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		handlerLogger.Error("failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	// Clean up meal service closed days (best effort)
	if h.mealClient != nil {
		if err := h.mealClient.DeleteClosedDays(ctx, semester.Name); err != nil {
			handlerLogger.Warn("failed to delete meal closed days", zap.Error(err))
		}
	}

	// Audit log
	if h.auditLogger != nil {
		if err := h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      getActorID(c),
			ActorRole:    "admin",
			Action:       "semester.deleted",
			ResourceType: "semester",
			ResourceID:   utils.PgtypeToUUIDString(semester.ID),
			Details: map[string]any{
				"semester_name": semester.Name,
			},
		}); err != nil {
			handlerLogger.Warn("audit log write failed", zap.Error(err))
		}
	}

	handlerLogger.Info("planned semester deleted", zap.String("name", semester.Name))
	c.Status(http.StatusNoContent)
}

// UpdatePlannedSemester handles PUT /admin/semesters/:id
func (h *SemesterStatusHandler) UpdatePlannedSemester(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterStatusHandler"),
		zap.String("method", "UpdatePlannedSemester"),
	)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid semester ID", "code": "VALIDATION_ERROR"})
		return
	}

	var req updateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "code": "VALIDATION_ERROR"})
		return
	}

	queries := db.New(h.pool)

	// Get semester
	semester, err := queries.GetSemesterByID(ctx, utils.UUIDToPgtype(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "semester not found", "code": "NOT_FOUND"})
		return
	}

	if semester.Status != db.CourseCatalogSemesterStatusPlanned {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Sadece planlanmış dönemler düzenlenebilir",
			"code":  "INVALID_STATE",
		})
		return
	}

	// Validation: hard_deadline must be in the future
	if req.HardDeadline.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "hard_deadline must be in the future",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	// Update hard_deadline
	updated, err := queries.UpdatePlannedSemester(ctx, db.UpdatePlannedSemesterParams{
		ID:           utils.UUIDToPgtype(id),
		HardDeadline: utils.TimeToPgTimestamptz(req.HardDeadline),
	})
	if err != nil {
		handlerLogger.Error("failed to update semester", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "INTERNAL_ERROR"})
		return
	}

	var updateErrors []string

	// Update periods if provided — catalog's rows are the source of truth and
	// each consumer type also gets an event.
	if req.Periods != nil {
		updateErrors = append(updateErrors,
			h.updateDistributedPeriods(ctx, semester.Name, req.HardDeadline, req.Periods)...)
	}

	// Update closed days if provided
	if len(req.ClosedDays) > 0 {
		if err := h.updateRemoteClosedDays(ctx, semester.Name, req.ClosedDays); err != nil {
			updateErrors = append(updateErrors, fmt.Sprintf("meal (closed_days): %v", err))
		}
	}

	// Audit log
	if h.auditLogger != nil {
		if err := h.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      getActorID(c),
			ActorRole:    "admin",
			Action:       "semester.updated",
			ResourceType: "semester",
			ResourceID:   utils.PgtypeToUUIDString(updated.ID),
			Details: map[string]any{
				"semester_name": updated.Name,
				"hard_deadline": req.HardDeadline.Format(time.RFC3339),
			},
		}); err != nil {
			handlerLogger.Warn("audit log write failed", zap.Error(err))
		}
	}

	resp := toSemesterResponse(updated)
	handlerLogger.Info("planned semester updated", zap.String("name", updated.Name))

	if len(updateErrors) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"semester":      resp,
			"update_errors": updateErrors,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

type updateSemesterRequest struct {
	HardDeadline time.Time        `json:"hard_deadline" binding:"required"`
	Periods      *semesterPeriods `json:"periods,omitempty"`
	ClosedDays   []closedDayEntry `json:"closed_days,omitempty"`
}

// updateRemoteClosedDays replaces closed days for a semester in meal service.
func (h *SemesterStatusHandler) updateRemoteClosedDays(ctx context.Context, semester string, closedDays []closedDayEntry) error {
	if h.mealClient == nil {
		return fmt.Errorf("meal client not configured")
	}
	return h.mealClient.ReplaceClosedDays(ctx, semester, toClosedDays(closedDays))
}
