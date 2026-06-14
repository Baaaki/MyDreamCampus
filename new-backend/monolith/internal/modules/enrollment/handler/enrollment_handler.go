package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/service"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestTimeout = 10 * time.Second

type EnrollmentHandler struct {
	enrollmentService *service.EnrollmentService
}

func NewEnrollmentHandler(enrollmentService *service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{
		enrollmentService: enrollmentService,
	}
}

// GetAvailableCourses godoc
// @Summary Get available courses for enrollment
// @Description Get all available courses for a student in a specific semester
// @Tags enrollment
// @Accept json
// @Produce json
// @Param semester query string true "Semester (e.g. 2024-2025-Fall)"
// @Security BearerAuth
// @Success 200 {object} dto.AvailableCoursesResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/available-courses [get]
func (h *EnrollmentHandler) GetAvailableCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "GetAvailableCourses"),
	)

	// Get student ID from JWT context
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Get semester from query
	semester := c.Query("semester")
	if semester == "" {
		handlerLogger.Error("semester query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester query parameter is required"})
		return
	}

	handlerLogger.Info("getting available courses",
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)

	response, err := h.enrollmentService.GetAvailableCourses(ctx, studentID, semester)
	if err != nil {
		handlerLogger.Error("failed to get available courses", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("available courses retrieved successfully",
		zap.Int("total_courses", len(response.AvailableCourses)),
	)
	c.JSON(http.StatusOK, response)
}

// CreateEnrollmentProgram godoc
// @Summary Create enrollment program
// @Description Create a new enrollment program (submit course selections)
// @Tags enrollment
// @Accept json
// @Produce json
// @Param request body dto.CreateEnrollmentRequest true "Enrollment request"
// @Security BearerAuth
// @Success 201 {object} dto.EnrollmentProgramResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 409 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/programs [post]
func (h *EnrollmentHandler) CreateEnrollmentProgram(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "CreateEnrollmentProgram"),
	)

	// Get student ID from JWT context
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req dto.CreateEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": sharedErrors.ErrValidation.Error()})
		return
	}

	// Set student ID from JWT
	req.StudentID = studentID

	handlerLogger.Info("creating enrollment program",
		zap.String("student_id", studentID.String()),
		zap.String("semester", req.Semester),
		zap.Int("course_count", len(req.CourseIDs)),
	)

	response, err := h.enrollmentService.CreateEnrollmentProgram(ctx, req)
	if err != nil {
		handlerLogger.Error("failed to create enrollment program", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("enrollment program created successfully",
		zap.String("program_id", response.ID.String()),
	)
	c.JSON(http.StatusCreated, response)
}

// GetMyEnrollments godoc
// @Summary Get my enrollments
// @Description Get all enrollment programs for the authenticated student
// @Tags enrollment
// @Accept json
// @Produce json
// @Param semester query string false "Semester filter"
// @Param status query string false "Status filter (pending, approved, rejected)"
// @Security BearerAuth
// @Success 200 {object} dto.MyEnrollmentsResponse
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/my-enrollments [get]
func (h *EnrollmentHandler) GetMyEnrollments(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "GetMyEnrollments"),
	)

	// Get student ID from JWT context
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Get optional filters
	var semester *string
	if s := c.Query("semester"); s != "" {
		semester = &s
	}

	var status *string
	if st := c.Query("status"); st != "" {
		status = &st
	}

	handlerLogger.Info("getting my enrollments",
		zap.String("student_id", studentID.String()),
	)

	response, err := h.enrollmentService.GetMyEnrollments(ctx, studentID, semester, status)
	if err != nil {
		handlerLogger.Error("failed to get my enrollments", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("my enrollments retrieved successfully",
		zap.Int("total_programs", len(response.Programs)),
	)
	c.JSON(http.StatusOK, response)
}

// GetLatestRejection godoc
// @Summary Get latest rejection
// @Description Get the latest rejection for a student in a specific semester
// @Tags enrollment
// @Accept json
// @Produce json
// @Param semester query string true "Semester"
// @Security BearerAuth
// @Success 200 {object} dto.LatestRejectionResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/latest-rejection [get]
func (h *EnrollmentHandler) GetLatestRejection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "GetLatestRejection"),
	)

	// Get student ID from JWT context
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	semester := c.Query("semester")
	if semester == "" {
		handlerLogger.Error("semester query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester query parameter is required"})
		return
	}

	handlerLogger.Info("getting latest rejection",
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)

	response, err := h.enrollmentService.GetLatestRejection(ctx, studentID, semester)
	if err != nil {
		handlerLogger.Error("failed to get latest rejection", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("latest rejection retrieved successfully")
	c.JSON(http.StatusOK, response)
}

// GetMyRejections godoc
// @Summary Get my rejections
// @Description Get all rejections for the authenticated student
// @Tags enrollment
// @Accept json
// @Produce json
// @Param semester query string false "Semester filter"
// @Security BearerAuth
// @Success 200 {object} dto.MyRejectionsResponse
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/my-rejections [get]
func (h *EnrollmentHandler) GetMyRejections(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "GetMyRejections"),
	)

	// Get student ID from JWT context
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var semester *string
	if s := c.Query("semester"); s != "" {
		semester = &s
	}

	handlerLogger.Info("getting my rejections",
		zap.String("student_id", studentID.String()),
	)

	response, err := h.enrollmentService.GetMyRejections(ctx, studentID, semester)
	if err != nil {
		handlerLogger.Error("failed to get my rejections", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("my rejections retrieved successfully",
		zap.Int("total_rejections", len(response.Rejections)),
	)
	c.JSON(http.StatusOK, response)
}

// ApproveEnrollmentProgram godoc
// @Summary Approve enrollment program
// @Description Approve a student's enrollment program (advisor only)
// @Tags enrollment-advisor
// @Accept json
// @Produce json
// @Param program_id path string true "Program ID"
// @Security BearerAuth
// @Success 200 {object} dto.EnrollmentProgramResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/advisor/programs/{program_id}/approve [post]
func (h *EnrollmentHandler) ApproveEnrollmentProgram(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "ApproveEnrollmentProgram"),
	)

	// Get advisor ID from JWT context
	advisorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	advisorID, err := uuid.Parse(advisorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	programIDStr := c.Param("program_id")
	programID, err := uuid.Parse(programIDStr)
	if err != nil {
		handlerLogger.Error("invalid program_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program_id"})
		return
	}

	handlerLogger.Info("approving enrollment program",
		zap.String("advisor_id", advisorID.String()),
		zap.String("program_id", programID.String()),
	)

	response, err := h.enrollmentService.ApproveEnrollmentProgram(ctx, programID, advisorID)
	if err != nil {
		handlerLogger.Error("failed to approve enrollment program", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("enrollment program approved successfully",
		zap.String("program_id", programID.String()),
	)
	c.JSON(http.StatusOK, response)
}

// RejectEnrollmentProgram godoc
// @Summary Reject enrollment program
// @Description Reject a student's enrollment program (advisor only)
// @Tags enrollment-advisor
// @Accept json
// @Produce json
// @Param program_id path string true "Program ID"
// @Param request body dto.RejectEnrollmentRequest true "Rejection request"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/advisor/programs/{program_id}/reject [post]
func (h *EnrollmentHandler) RejectEnrollmentProgram(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "RejectEnrollmentProgram"),
	)

	// Get advisor ID and fullname from JWT context
	advisorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	advisorID, err := uuid.Parse(advisorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Get advisor fullname from context (set by auth middleware)
	advisorFullname, _ := c.Get("fullname")
	advisorFullnameStr := ""
	if advisorFullname != nil {
		advisorFullnameStr = advisorFullname.(string)
	}

	programIDStr := c.Param("program_id")
	programID, err := uuid.Parse(programIDStr)
	if err != nil {
		handlerLogger.Error("invalid program_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program_id"})
		return
	}

	var req dto.RejectEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": sharedErrors.ErrValidation.Error()})
		return
	}

	handlerLogger.Info("rejecting enrollment program",
		zap.String("advisor_id", advisorID.String()),
		zap.String("program_id", programID.String()),
		zap.String("reason", req.RejectionReason),
	)

	err = h.enrollmentService.RejectEnrollmentProgram(ctx, programID, advisorID, advisorFullnameStr, req.RejectionReason)
	if err != nil {
		handlerLogger.Error("failed to reject enrollment program", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("enrollment program rejected successfully",
		zap.String("program_id", programID.String()),
	)
	c.JSON(http.StatusOK, gin.H{"message": "enrollment program rejected successfully"})
}

// CancelMyEnrollment godoc
// @Summary Cancel my enrollment
// @Description Cancel the student's enrollment program for a semester (only if not approved)
// @Tags enrollment
// @Accept json
// @Produce json
// @Param semester query string true "Semester (e.g. 2024-2025-Fall)"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H "Enrollment already approved"
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/programs [delete]
func (h *EnrollmentHandler) CancelMyEnrollment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "CancelMyEnrollment"),
	)

	// Get student ID from JWT context
	studentIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	studentID, err := uuid.Parse(studentIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Get semester from query
	semester := c.Query("semester")
	if semester == "" {
		handlerLogger.Error("semester query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "semester query parameter is required"})
		return
	}

	handlerLogger.Info("cancelling enrollment",
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)

	err = h.enrollmentService.CancelMyEnrollment(ctx, studentID, semester)
	if err != nil {
		handlerLogger.Error("failed to cancel enrollment", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("enrollment cancelled successfully",
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)
	c.JSON(http.StatusOK, gin.H{"message": "enrollment cancelled successfully"})
}

// GetPendingProgramsByAdvisor godoc
// @Summary Get pending programs
// @Description Get all pending enrollment programs for an advisor
// @Tags enrollment-advisor
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.AdvisorPendingProgramsResponse
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /enrollment/advisor/pending-programs [get]
func (h *EnrollmentHandler) GetPendingProgramsByAdvisor(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	handlerLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "EnrollmentHandler"),
		zap.String("method", "GetPendingProgramsByAdvisor"),
	)

	// Get advisor ID from JWT context
	advisorIDStr, exists := c.Get("user_id")
	if !exists {
		handlerLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": sharedErrors.ErrUnauthorized.Error()})
		return
	}

	advisorID, err := uuid.Parse(advisorIDStr.(string))
	if err != nil {
		handlerLogger.Error("invalid user_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	handlerLogger.Info("getting pending programs for advisor",
		zap.String("advisor_id", advisorID.String()),
	)

	response, err := h.enrollmentService.GetPendingProgramsByAdvisor(ctx, advisorID)
	if err != nil {
		handlerLogger.Error("failed to get pending programs", zap.Error(err))
		h.handleError(c, err)
		return
	}

	handlerLogger.Info("pending programs retrieved successfully",
		zap.Int("total_programs", len(response.Programs)),
	)
	c.JSON(http.StatusOK, response)
}

// handleError maps service errors to HTTP status codes
func (h *EnrollmentHandler) handleError(c *gin.Context, err error) {
	reqLogger := logger.WithContextAndFields(c.Request.Context(),
		zap.String("handler", "EnrollmentHandler"),
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
