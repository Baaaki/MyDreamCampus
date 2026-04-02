package service

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/baaaki/mydreamcampus/student-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type StudentService struct {
	studentRepo  *repository.StudentRepository
	staffService StaffServiceInterface
}

// StaffServiceInterface defines methods to interact with Staff Service
type StaffServiceInterface interface {
	ValidateAdvisor(ctx context.Context, advisorID uuid.UUID) error
	GetInstructorsByDepartment(ctx context.Context, department string) ([]uuid.UUID, error)
}

func NewStudentService(studentRepo *repository.StudentRepository, staffService StaffServiceInterface) *StudentService {
	return &StudentService{
		studentRepo:  studentRepo,
		staffService: staffService,
	}
}

// CreateStudent creates a new student
func (s *StudentService) CreateStudent(ctx context.Context, req dto.CreateStudentRequest) (dto.StudentResponse, error) {
	// Check if student number already exists
	existingStudent, err := s.studentRepo.GetStudentByNumber(ctx, req.StudentNumber)
	if err != nil && err != pgx.ErrNoRows {
		logger.Error("failed to check student number existence",
			zap.Error(err),
			zap.String("student_number", req.StudentNumber),
		)
		return dto.StudentResponse{}, errors.ErrInternal
	}
	if err == nil && existingStudent.StudentNumber != "" {
		logger.Warn("student number already exists",
			zap.String("student_number", req.StudentNumber),
		)
		return dto.StudentResponse{}, errors.ErrStudentNumberExists
	}

	// Check if email already exists
	existingStudent, err = s.studentRepo.GetStudentByEmail(ctx, req.Email)
	if err != nil && err != pgx.ErrNoRows {
		logger.Error("failed to check email existence",
			zap.Error(err),
			zap.String("email", req.Email),
		)
		return dto.StudentResponse{}, errors.ErrInternal
	}
	if err == nil && existingStudent.Email != "" {
		logger.Warn("email already exists",
			zap.String("email", req.Email),
		)
		return dto.StudentResponse{}, errors.ErrEmailExists
	}

	// Validate advisor exists (Staff Service)
	if err := s.staffService.ValidateAdvisor(ctx, req.AdvisorID); err != nil {
		logger.Error("advisor validation failed",
			zap.Error(err),
			zap.String("advisor_id", req.AdvisorID.String()),
		)
		return dto.StudentResponse{}, errors.ErrAdvisorNotFound
	}

	// Create student with outbox event
	params := db.CreateStudentParams{
		StudentNumber:  req.StudentNumber,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		Faculty:        req.Faculty,
		Department:     req.Department,
		EnrollmentYear: int32(req.EnrollmentYear),
		ClassLevel:     req.ClassLevel,
		AdvisorID:      utils.UUIDToPgtype(req.AdvisorID),
	}

	eventPayload := map[string]interface{}{
		"id":              nil, // Will be set after creation
		"student_number":  req.StudentNumber,
		"first_name":      req.FirstName,
		"last_name":       req.LastName,
		"email":           req.Email,
		"faculty":         req.Faculty,
		"department":      req.Department,
		"enrollment_year": req.EnrollmentYear,
		"class_level":     req.ClassLevel,
		"advisor_id":      req.AdvisorID.String(),
		"status":          "active",
	}

	student, err := s.studentRepo.CreateStudentWithEvent(ctx, params, eventPayload)
	if err != nil {
		logger.Error("failed to create student",
			zap.Error(err),
			zap.String("student_number", req.StudentNumber),
		)
		return dto.StudentResponse{}, errors.ErrInternal
	}

	logger.Info("student created successfully",
		zap.String("student_id", utils.PgtypeToUUIDString(student.ID)),
		zap.String("student_number", student.StudentNumber),
	)

	return s.toStudentResponse(student), nil
}

// GetStudentByID retrieves student by ID
func (s *StudentService) GetStudentByID(ctx context.Context, id string) (dto.StudentResponse, error) {
	studentID, err := uuid.Parse(id)
	if err != nil {
		return dto.StudentResponse{}, errors.ErrInvalidID
	}

	student, err := s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.StudentResponse{}, errors.ErrStudentNotFound
		}
		logger.Error("failed to get student",
			zap.Error(err),
			zap.String("student_id", id),
		)
		return dto.StudentResponse{}, errors.ErrInternal
	}

	return s.toStudentResponse(student), nil
}

// UpdateStudent updates student information
func (s *StudentService) UpdateStudent(ctx context.Context, id string, req dto.UpdateStudentRequest) (dto.StudentResponse, error) {
	studentID, err := uuid.Parse(id)
	if err != nil {
		return dto.StudentResponse{}, errors.ErrInvalidID
	}

	// Check if student exists
	_, err = s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if err.Error() == "student not found" {
			return dto.StudentResponse{}, errors.ErrStudentNotFound
		}
		return dto.StudentResponse{}, errors.ErrInternal
	}

	// Validate advisor if provided
	if req.AdvisorID != nil {
		if err := s.staffService.ValidateAdvisor(ctx, *req.AdvisorID); err != nil {
			logger.Error("advisor validation failed",
				zap.Error(err),
				zap.String("advisor_id", req.AdvisorID.String()),
			)
			return dto.StudentResponse{}, errors.ErrAdvisorNotFound
		}
	}

	// Get current student data for COALESCE defaults
	currentStudent, _ := s.studentRepo.GetStudentByID(ctx, studentID)

	classLevel := currentStudent.ClassLevel
	if req.ClassLevel != nil {
		classLevel = *req.ClassLevel
	}

	params := db.UpdateStudentParams{
		ID:         utils.UUIDToPgtype(studentID),
		ClassLevel: classLevel,
		AdvisorID:  utils.PointerUUIDToPgtype(req.AdvisorID),
		Status:     utils.PointerStringToPgText(req.Status),
	}

	changedFields := make(map[string]interface{})
	if req.ClassLevel != nil {
		changedFields["class_level"] = *req.ClassLevel
	}
	if req.AdvisorID != nil {
		changedFields["advisor_id"] = req.AdvisorID.String()
	}
	if req.Status != nil {
		changedFields["status"] = *req.Status
	}

	eventPayload := map[string]interface{}{
		"id":             id,
		"student_number": "", // Will be filled from DB
		"changed_fields": changedFields,
	}

	student, err := s.studentRepo.UpdateStudentWithEvent(ctx, studentID, params, eventPayload)
	if err != nil {
		logger.Error("failed to update student",
			zap.Error(err),
			zap.String("student_id", id),
		)
		return dto.StudentResponse{}, errors.ErrInternal
	}

	logger.Info("student updated successfully",
		zap.String("student_id", id),
	)

	return s.toStudentResponse(student), nil
}

// DeleteStudent soft deletes a student
func (s *StudentService) DeleteStudent(ctx context.Context, id string) error {
	studentID, err := uuid.Parse(id)
	if err != nil {
		return errors.ErrInvalidID
	}

	// Check if student exists
	student, err := s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if err.Error() == "student not found" {
			return errors.ErrStudentNotFound
		}
		return errors.ErrInternal
	}

	eventPayload := map[string]interface{}{
		"id":             id,
		"student_number": student.StudentNumber,
		"is_active":      false,
		"deleted_at":     time.Now().Format(time.RFC3339),
	}

	err = s.studentRepo.SoftDeleteStudentWithEvent(ctx, studentID, eventPayload)
	if err != nil {
		logger.Error("failed to delete student",
			zap.Error(err),
			zap.String("student_id", id),
		)
		return errors.ErrInternal
	}

	logger.Info("student deleted successfully",
		zap.String("student_id", id),
	)

	return nil
}

// ListStudents lists students with pagination, filtering, and sorting
func (s *StudentService) ListStudents(ctx context.Context, query dto.PaginationQuery) (dto.StudentListResponse, error) {
	limit := int32(query.Limit)
	offset := int32((query.Page - 1) * query.Limit)

	// Set default sort if not provided
	sortBy := "created_at"
	sortOrder := "desc"
	if query.SortBy != nil {
		sortBy = *query.SortBy
	}
	if query.SortOrder != nil {
		sortOrder = *query.SortOrder
	}

	// Build params for database query
	params := db.ListStudentsParams{
		Department: utils.PointerStringToPgText(query.Department),
		ClassLevel: utils.PointerInt16ToPgInt2(query.ClassLevel),
		Status:     utils.PointerStringToPgText(query.Status),
		AdvisorID:  utils.PointerUUIDToPgtype(query.AdvisorID),
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Limit:      limit,
		Offset:     offset,
	}

	students, err := s.studentRepo.ListStudentsFiltered(ctx, params)
	if err != nil {
		logger.Error("failed to list students",
			zap.Error(err),
		)
		return dto.StudentListResponse{}, errors.ErrInternal
	}

	// Get total count (we'll use existing CountStudents for now, ideally should count with filters)
	total, err := s.studentRepo.CountStudents(ctx)
	if err != nil {
		logger.Error("failed to count students",
			zap.Error(err),
		)
		return dto.StudentListResponse{}, errors.ErrInternal
	}

	var studentResponses []dto.StudentResponse
	for _, student := range students {
		studentResponses = append(studentResponses, s.toStudentResponse(student))
	}

	return dto.StudentListResponse{
		Data: studentResponses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      int(total),
			TotalPages: (int(total) + query.Limit - 1) / query.Limit,
		},
	}, nil
}

// ListStudentsByAdvisor lists students by advisor (for teachers)
func (s *StudentService) ListStudentsByAdvisor(ctx context.Context, advisorID uuid.UUID) (dto.MyAdviseesResponse, error) {
	students, err := s.studentRepo.ListStudentsByAdvisor(ctx, advisorID)
	if err != nil {
		logger.Error("failed to list students by advisor",
			zap.Error(err),
			zap.String("advisor_id", advisorID.String()),
		)
		return dto.MyAdviseesResponse{}, errors.ErrInternal
	}

	var studentResponses []dto.StudentResponse
	for _, student := range students {
		studentResponses = append(studentResponses, s.toStudentResponse(student))
	}

	return dto.MyAdviseesResponse{
		Advisor: dto.AdvisorInfo{
			ID: advisorID.String(),
		},
		Students:   studentResponses,
		TotalCount: len(studentResponses),
	}, nil
}

// ListOrphanedStudents lists students without advisor
func (s *StudentService) ListOrphanedStudents(ctx context.Context, query dto.PaginationQuery) (dto.OrphanedStudentsResponse, error) {
	limit := int32(query.Limit)
	offset := int32((query.Page - 1) * query.Limit)

	students, total, err := s.studentRepo.ListOrphanedStudents(ctx, limit, offset)
	if err != nil {
		logger.Error("failed to list orphaned students",
			zap.Error(err),
		)
		return dto.OrphanedStudentsResponse{}, errors.ErrInternal
	}

	var studentResponses []dto.StudentResponse
	for _, student := range students {
		studentResponses = append(studentResponses, s.toStudentResponse(student))
	}

	return dto.OrphanedStudentsResponse{
		Data: studentResponses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      int(total),
			TotalPages: (int(total) + query.Limit - 1) / query.Limit,
		},
	}, nil
}

// BulkAssignAdvisor assigns advisor to multiple students
func (s *StudentService) BulkAssignAdvisor(ctx context.Context, req dto.BulkAdvisorAssignRequest) (dto.BulkAdvisorAssignResponse, error) {
	// Validate advisor exists
	if err := s.staffService.ValidateAdvisor(ctx, req.AdvisorID); err != nil {
		logger.Error("advisor validation failed",
			zap.Error(err),
			zap.String("advisor_id", req.AdvisorID.String()),
		)
		return dto.BulkAdvisorAssignResponse{}, errors.ErrAdvisorNotFound
	}

	// Create event payloads for each student
	eventPayloads := make([]map[string]interface{}, len(req.StudentIDs))
	for i, studentID := range req.StudentIDs {
		eventPayloads[i] = map[string]interface{}{
			"id":             studentID.String(),
			"student_number": "", // Will be filled from DB if needed
			"changed_fields": map[string]interface{}{
				"advisor_id": req.AdvisorID.String(),
			},
		}
	}

	// Bulk assign
	err := s.studentRepo.BulkAssignAdvisor(ctx, req.StudentIDs, req.AdvisorID, eventPayloads)
	if err != nil {
		logger.Error("failed to bulk assign advisor",
			zap.Error(err),
		)
		return dto.BulkAdvisorAssignResponse{}, errors.ErrInternal
	}

	// Build response
	studentBasicInfos := make([]dto.StudentBasicInfo, len(req.StudentIDs))
	for i, id := range req.StudentIDs {
		studentBasicInfos[i] = dto.StudentBasicInfo{
			ID:            id.String(),
			StudentNumber: "", // Could be fetched if needed
		}
	}

	logger.Info("bulk advisor assignment completed",
		zap.Int("student_count", len(req.StudentIDs)),
		zap.String("advisor_id", req.AdvisorID.String()),
	)

	return dto.BulkAdvisorAssignResponse{
		Message:      "Advisor assigned successfully",
		UpdatedCount: len(req.StudentIDs),
		Advisor: dto.AdvisorInfo{
			ID: req.AdvisorID.String(),
		},
		Students: studentBasicInfos,
	}, nil
}

// SearchStudents performs advanced search with filters
func (s *StudentService) SearchStudents(ctx context.Context, req dto.SearchStudentsRequest) (dto.SearchStudentsResponse, error) {
	// Set defaults for pagination
	limit := req.Pagination.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// For now, use simple SQL query with basic filters
	// Extract first filter values for simple implementation
	var department *string
	if len(req.Filters.Department) > 0 {
		department = &req.Filters.Department[0]
	}

	var classLevel *int16
	if len(req.Filters.ClassLevel) > 0 {
		classLevel = &req.Filters.ClassLevel[0]
	}

	var status *string
	if len(req.Filters.Status) > 0 {
		status = &req.Filters.Status[0]
	}

	// Prepare search parameters using existing SearchStudents query
	var query *string
	if req.Query != "" {
		query = &req.Query
	}

	params := db.SearchStudentsParams{
		Query:      utils.PointerStringToPgText(query),
		Department: utils.PointerStringToPgText(department),
		ClassLevel: utils.PointerInt16ToPgInt2(classLevel),
		Status:     utils.PointerStringToPgText(status),
		AdvisorID:  utils.PointerUUIDToPgtype(req.Filters.AdvisorID),
		Limit:      int32(limit),
		Offset:     0, // For cursor-based pagination, always start from 0
	}

	students, err := s.studentRepo.SearchStudents(ctx, params)
	if err != nil {
		logger.Error("failed to search students",
			zap.Error(err),
			zap.String("query", req.Query),
		)
		return dto.SearchStudentsResponse{}, errors.ErrInternal
	}

	var studentResponses []dto.StudentResponse
	for _, student := range students {
		studentResponses = append(studentResponses, s.toStudentResponse(student))
	}

	hasMore := len(studentResponses) >= limit
	nextCursor := ""
	if hasMore && len(studentResponses) > 0 {
		// In real implementation, use actual cursor (e.g., last student's ID or timestamp)
		nextCursor = studentResponses[len(studentResponses)-1].ID
	}

	logger.Info("search completed",
		zap.String("query", req.Query),
		zap.Int("results", len(studentResponses)),
	)

	return dto.SearchStudentsResponse{
		Data: studentResponses,
		Pagination: dto.SearchPaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
			TotalCount: len(studentResponses), // Note: This is not accurate total, just current page count
		},
	}, nil
}

// toStudentResponse converts db.Student to dto.StudentResponse
func (s *StudentService) toStudentResponse(student db.Student) dto.StudentResponse {
	var advisorID *string
	if student.AdvisorID.Valid {
		id := utils.PgtypeToUUIDString(student.AdvisorID)
		advisorID = &id
	}

	return dto.StudentResponse{
		ID:             utils.PgtypeToUUIDString(student.ID),
		StudentNumber:  student.StudentNumber,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Faculty:        student.Faculty,
		Department:     student.Department,
		EnrollmentYear: int(student.EnrollmentYear),
		ClassLevel:     student.ClassLevel,
		AdvisorID:      advisorID,
		Status:         utils.PgTextToString(student.Status),
		CreatedAt:      student.CreatedAt.Time,
		UpdatedAt:      student.UpdatedAt.Time,
	}
}
