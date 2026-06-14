package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/service"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestTimeout = 10 * time.Second

type GradeHandler struct {
	gradeService        *service.GradeService
	studentGradeService *service.StudentGradesService
}

func NewGradeHandler(
	gradeService *service.GradeService,
	studentGradeService *service.StudentGradesService,
) *GradeHandler {
	return &GradeHandler{
		gradeService:        gradeService,
		studentGradeService: studentGradeService,
	}
}

// ============================================
// Instructor Endpoints
// ============================================

// GetCourseStatus - GET /api/v1/grades/course/:courseId/status
func (h *GradeHandler) GetCourseStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "GetCourseStatus"),
	)

	// Get instructor ID from context (set by ExtractUserFromHeaders middleware as string)
	instructorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse string to UUID
	instructorID, err := uuid.Parse(instructorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid instructor ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid instructor ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Parse course ID
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		handlerLogger.Error("invalid course ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid course ID",
			Code:  "INVALID_ID",
		})
		return
	}

	role, _ := c.Get("role")
	isAdmin := role == "admin"

	handlerLogger.Info("getting course status",
		zap.String("instructor_id", instructorID.String()),
		zap.String("course_id", courseID.String()),
		zap.Bool("is_admin", isAdmin),
	)

	// Get course status
	status, err := h.gradeService.GetCourseStatus(ctx, instructorID, courseID, isAdmin)
	if err != nil {
		handlerLogger.Error("failed to get course status", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetCourseStudents - GET /api/v1/grades/course/:courseId/students
func (h *GradeHandler) GetCourseStudents(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "GetCourseStudents"),
	)

	// Get instructor ID from context (set by ExtractUserFromHeaders middleware as string)
	instructorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse string to UUID
	instructorID, err := uuid.Parse(instructorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid instructor ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid instructor ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Parse course ID
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		handlerLogger.Error("invalid course ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid course ID",
			Code:  "INVALID_ID",
		})
		return
	}

	role, _ := c.Get("role")
	isAdmin := role == "admin"

	handlerLogger.Info("getting course students",
		zap.String("instructor_id", instructorID.String()),
		zap.String("course_id", courseID.String()),
		zap.Bool("is_admin", isAdmin),
	)

	// Get course students
	students, err := h.gradeService.GetCourseStudents(ctx, instructorID, courseID, isAdmin)
	if err != nil {
		handlerLogger.Error("failed to get course students", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, students)
}

// SubmitScore - POST /api/v1/grades/course/:courseId/scores
func (h *GradeHandler) SubmitScore(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "SubmitScore"),
	)

	// Get instructor ID from context (set by ExtractUserFromHeaders middleware as string)
	instructorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse string to UUID
	instructorID, err := uuid.Parse(instructorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid instructor ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid instructor ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Parse course ID
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		handlerLogger.Error("invalid course ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid course ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Parse request body
	var req dto.SubmitScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	handlerLogger.Info("submitting score",
		zap.String("instructor_id", instructorID.String()),
		zap.String("course_id", courseID.String()),
		zap.String("registration_id", req.RegistrationID.String()),
	)

	// Submit score
	result, err := h.gradeService.SubmitScore(ctx, instructorID, courseID, req)
	if err != nil {
		handlerLogger.Error("failed to submit score", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("score submitted successfully")
	c.JSON(http.StatusCreated, result)
}

// BulkSubmitScores - POST /api/v1/grades/course/:courseId/scores/bulk
func (h *GradeHandler) BulkSubmitScores(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "BulkSubmitScores"),
	)

	// Get instructor ID from context (set by ExtractUserFromHeaders middleware as string)
	instructorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse string to UUID
	instructorID, err := uuid.Parse(instructorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid instructor ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid instructor ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Parse course ID
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		handlerLogger.Error("invalid course ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid course ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Parse request body
	var req dto.BulkSubmitScoresRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	handlerLogger.Info("bulk submitting scores",
		zap.String("instructor_id", instructorID.String()),
		zap.String("course_id", courseID.String()),
		zap.Int("score_count", len(req.Scores)),
	)

	// Bulk submit scores
	result, err := h.gradeService.BulkSubmitScores(ctx, instructorID, courseID, req)
	if err != nil {
		handlerLogger.Error("failed to bulk submit scores", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("bulk scores submitted successfully",
		zap.Int("success_count", result.SuccessCount),
		zap.Bool("auto_finalized", result.AutoFinalized),
	)
	c.JSON(http.StatusCreated, result)
}

// LockAssessment - POST /course/:course_id/assessments/:slug/lock
// Instructor-only: marks every student's score for this assessment as final.
// Once all assessments are locked, the course auto-finalizes.
func (h *GradeHandler) LockAssessment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "LockAssessment"),
	)

	instructorIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}
	instructorID, err := uuid.Parse(instructorIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid instructor ID", Code: "INVALID_ID"})
		return
	}

	courseID, err := uuid.Parse(c.Param("course_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid course ID", Code: "INVALID_ID"})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "slug is required", Code: "VALIDATION_ERROR"})
		return
	}

	handlerLogger.Info("locking assessment",
		zap.String("instructor_id", instructorID.String()),
		zap.String("course_id", courseID.String()),
		zap.String("slug", slug),
	)

	result, err := h.gradeService.LockAssessmentBySlug(ctx, instructorID, courseID, slug)
	if err != nil {
		handlerLogger.Warn("failed to lock assessment", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================
// Student Endpoints
// ============================================

// GetMyGrades - GET /api/v1/grades/student/my
func (h *GradeHandler) GetMyGrades(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "GetMyGrades"),
	)

	// Get student ID from context (set by ExtractUserFromHeaders middleware as string)
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse string to UUID
	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid student ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid student ID",
			Code:  "INVALID_ID",
		})
		return
	}

	handlerLogger.Info("getting my grades",
		zap.String("student_id", studentID.String()),
	)

	// Get my grades
	grades, err := h.studentGradeService.GetMyGrades(ctx, studentID)
	if err != nil {
		handlerLogger.Error("failed to get my grades", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, grades)
}

// GetTranscript - GET /api/v1/grades/transcript/:studentId
func (h *GradeHandler) GetTranscript(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "GetTranscript"),
	)

	// Get requester info from context (set by ExtractUserFromHeaders middleware as string)
	requesterIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse string to UUID
	requesterID, err := uuid.Parse(requesterIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid requester ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid requester ID",
			Code:  "INVALID_ID",
		})
		return
	}

	requesterRole, exists := c.Get("role")
	if !exists {
		handlerLogger.Error("role not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: sharedErrors.ErrUnauthorized.Message,
			Code:  sharedErrors.ErrUnauthorized.Code,
		})
		return
	}

	// Parse student ID
	studentIDStr := c.Param("student_id")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		handlerLogger.Error("invalid student ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid student ID",
			Code:  "INVALID_ID",
		})
		return
	}

	handlerLogger.Info("getting transcript",
		zap.String("requester_id", requesterID.String()),
		zap.String("requester_role", requesterRole.(string)),
		zap.String("student_id", studentID.String()),
	)

	// Get transcript
	transcript, err := h.studentGradeService.GetTranscript(
		ctx,
		requesterID,
		requesterRole.(string),
		studentID,
	)
	if err != nil {
		handlerLogger.Error("failed to get transcript", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, transcript)
}

// ============================================
// Admin Endpoints
// ============================================

// ProcessAppeal - POST /api/v1/grades/admin/appeal
// Admin only: Recalculate a student's grade after score correction using frozen statistics
func (h *GradeHandler) ProcessAppeal(c *gin.Context) {
	parentCtx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(parentCtx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "ProcessAppeal"),
	)

	// Verify admin role
	role, exists := c.Get("role")
	if !exists || role.(string) != "admin" {
		handlerLogger.Warn("unauthorized appeal attempt", zap.Any("role", role))
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: "only admins can process appeals",
			Code:  "FORBIDDEN",
		})
		return
	}

	// Parse request body
	var req dto.AppealScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	handlerLogger.Info("processing appeal",
		zap.String("student_id", req.StudentID.String()),
		zap.String("course_id", req.CourseID.String()),
		zap.String("slug", req.Slug),
		zap.Float64("new_score", req.NewScore),
	)

	// Add user_id to context for audit logging
	userID, _ := c.Get("user_id")
	ctx := context.WithValue(parentCtx, "user_id", userID)

	// Process appeal
	result, err := h.gradeService.ProcessAppeal(ctx, req)
	if err != nil {
		handlerLogger.Error("failed to process appeal", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("appeal processed successfully",
		zap.String("old_grade", result.OldGradePoint),
		zap.String("new_grade", result.NewGradePoint),
	)
	c.JSON(http.StatusOK, result)
}

// UnlockScore - POST /api/grades/admin/scores/unlock
// Admin only: Unlock a specific score so the teacher can re-enter it
func (h *GradeHandler) UnlockScore(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "UnlockScore"),
	)

	var req dto.ScoreLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	handlerLogger.Info("unlocking score",
		zap.String("registration_id", req.RegistrationID.String()),
		zap.String("slug", req.Slug),
	)

	if err := h.gradeService.UnlockScore(ctx, req.RegistrationID, req.Slug); err != nil {
		handlerLogger.Error("failed to unlock score", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("score unlocked successfully")
	c.JSON(http.StatusOK, gin.H{
		"message":         "score unlocked successfully",
		"registration_id": req.RegistrationID,
		"slug":            req.Slug,
	})
}

// LockScore - POST /api/grades/admin/scores/lock
// Admin only: Lock a specific score to prevent modification
func (h *GradeHandler) LockScore(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "GradeHandler"),
		zap.String("method", "LockScore"),
	)

	var req dto.ScoreLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	handlerLogger.Info("locking score",
		zap.String("registration_id", req.RegistrationID.String()),
		zap.String("slug", req.Slug),
	)

	if err := h.gradeService.LockScore(ctx, req.RegistrationID, req.Slug); err != nil {
		handlerLogger.Error("failed to lock score", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("score locked successfully")
	c.JSON(http.StatusOK, gin.H{
		"message":         "score locked successfully",
		"registration_id": req.RegistrationID,
		"slug":            req.Slug,
	})
}

// ============================================
// Helper Functions
// ============================================

// handleError maps service errors to HTTP status codes
func (h *GradeHandler) handleError(c *gin.Context, err error) {
	reqLogger := logger.WithContextAndFields(c.Request.Context(),
		zap.String("handler", "GradeHandler"),
		zap.String("method", "handleError"),
	)

	// Try to extract AppError
	if appErr, ok := sharedErrors.As(err); ok {
		reqLogger.Warn("application error",
			zap.Error(err),
			zap.String("error_code", appErr.Code),
		)
		c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
			Error: appErr.Message,
			Code:  appErr.Code,
		})
		return
	}

	// Unexpected error (not an AppError)
	reqLogger.Error("unexpected error",
		zap.Error(err),
	)
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Error: sharedErrors.ErrInternal.Message,
		Code:  sharedErrors.ErrInternal.Code,
	})
}
