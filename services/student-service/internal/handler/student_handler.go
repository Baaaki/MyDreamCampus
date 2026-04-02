package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/baaaki/mydreamcampus/student-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	requestTimeout = 10 * time.Second
	maxPageLimit   = 100
	defaultLimit   = 20
)

type StudentHandler struct {
	service       *service.StudentService
	importService *service.ImportService
}

func NewStudentHandler(service *service.StudentService, importService *service.ImportService) *StudentHandler {
	return &StudentHandler{
		service:       service,
		importService: importService,
	}
}

// CreateStudent creates a new student
func (h *StudentHandler) CreateStudent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "CreateStudent"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	logger.Info("creating student",
		zap.String("student_number", req.StudentNumber),
		zap.String("email", req.Email),
	)

	response, err := h.service.CreateStudent(ctx, req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("student created successfully",
		zap.String("student_id", response.ID),
	)

	c.JSON(http.StatusCreated, response)
}

// GetStudentByID retrieves student by ID
func (h *StudentHandler) GetStudentByID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	id := c.Param("id")

	logger.Info("getting student by ID",
		zap.String("student_id", id),
	)

	response, err := h.service.GetStudentByID(ctx, id)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateStudent updates student information
func (h *StudentHandler) UpdateStudent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	id := c.Param("id")

	var req dto.UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "UpdateStudent"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	logger.Info("updating student",
		zap.String("student_id", id),
	)

	response, err := h.service.UpdateStudent(ctx, id, req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("student updated successfully",
		zap.String("student_id", id),
	)

	c.JSON(http.StatusOK, response)
}

// DeleteStudent soft deletes a student
func (h *StudentHandler) DeleteStudent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	id := c.Param("id")

	logger.Info("deleting student",
		zap.String("student_id", id),
	)

	err := h.service.DeleteStudent(ctx, id)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("student deleted successfully",
		zap.String("student_id", id),
	)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Student deleted successfully",
	})
}

// ListStudents lists students with pagination
func (h *StudentHandler) ListStudents(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Error("invalid query parameters",
			zap.Error(err),
			zap.String("endpoint", "ListStudents"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	// Set defaults and apply limits
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxPageLimit {
		query.Limit = maxPageLimit
	}
	if query.Limit < 1 {
		query.Limit = defaultLimit
	}

	logger.Info("listing students",
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
	)

	response, err := h.service.ListStudents(ctx, query)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMyAdvisees retrieves students for an advisor (teacher)
func (h *StudentHandler) GetMyAdvisees(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// TODO: Get advisor ID from JWT token (when auth is implemented)
	// For now, accept it as a query parameter
	advisorIDStr := c.Query("advisor_id")
	if advisorIDStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "advisor_id is required",
			Code:  "MISSING_ADVISOR_ID",
		})
		return
	}

	advisorID, err := uuid.Parse(advisorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid advisor_id",
			Code:  "INVALID_ADVISOR_ID",
		})
		return
	}

	logger.Info("getting advisor's students",
		zap.String("advisor_id", advisorID.String()),
	)

	response, err := h.service.ListStudentsByAdvisor(ctx, advisorID)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListOrphanedStudents lists students without advisor
func (h *StudentHandler) ListOrphanedStudents(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Error("invalid query parameters",
			zap.Error(err),
			zap.String("endpoint", "ListOrphanedStudents"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	// Set defaults and apply limits
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxPageLimit {
		query.Limit = maxPageLimit
	}
	if query.Limit < 1 {
		query.Limit = defaultLimit
	}

	logger.Info("listing orphaned students",
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
	)

	response, err := h.service.ListOrphanedStudents(ctx, query)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// BulkAssignAdvisor assigns advisor to multiple students
func (h *StudentHandler) BulkAssignAdvisor(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.BulkAdvisorAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "BulkAssignAdvisor"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	logger.Info("bulk assigning advisor",
		zap.Int("student_count", len(req.StudentIDs)),
		zap.String("advisor_id", req.AdvisorID.String()),
	)

	response, err := h.service.BulkAssignAdvisor(ctx, req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("bulk advisor assignment completed",
		zap.Int("updated_count", response.UpdatedCount),
	)

	c.JSON(http.StatusOK, response)
}

// SearchStudents performs advanced search with filters
func (h *StudentHandler) SearchStudents(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.SearchStudentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "SearchStudents"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	logger.Info("searching students",
		zap.String("query", req.Query),
		zap.Int("filters_count", len(req.Filters.Department)+len(req.Filters.ClassLevel)+len(req.Filters.Status)),
	)

	response, err := h.service.SearchStudents(ctx, req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("search completed",
		zap.String("query", req.Query),
		zap.Int("results", len(response.Data)),
	)

	c.JSON(http.StatusOK, response)
}

// BulkImport handles CSV bulk import
func (h *StudentHandler) BulkImport(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		logger.Error("failed to get uploaded file",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "File is required",
			Code:  errors.ErrValidation.Code,
		})
		return
	}
	defer file.Close()

	// Validate file extension
	fileName := header.Filename
	if !strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Only CSV files are supported",
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	// TODO: Get user ID from JWT token (for now use a placeholder)
	createdBy := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	logger.Info("processing bulk import",
		zap.String("filename", fileName),
		zap.Int64("size", header.Size),
	)

	// Call import service
	jobID, err := h.importService.BulkImportStudents(ctx, fileName, file, createdBy)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("bulk import job created",
		zap.String("job_id", jobID),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"status":  "pending",
		"message": "Import job created successfully",
	})
}

// GetImportJobStatus retrieves import job status
func (h *StudentHandler) GetImportJobStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	jobID := c.Param("job_id")

	logger.Info("getting import job status",
		zap.String("job_id", jobID),
	)

	response, err := h.importService.GetImportJobStatus(ctx, jobID)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListImportJobs lists all import jobs for the current user
func (h *StudentHandler) ListImportJobs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var query dto.ImportJobFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Error("invalid query parameters",
			zap.Error(err),
			zap.String("endpoint", "ListImportJobs"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	// TODO: Get user ID from JWT token (for now use a placeholder)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	logger.Info("listing import jobs",
		zap.String("user_id", userID.String()),
	)

	response, err := h.importService.ListImportJobs(ctx, userID, query)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
