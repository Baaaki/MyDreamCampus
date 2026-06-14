package service

import (
	"context"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/student-service/internal/errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StudentRepositoryInterface defines methods for student data access
type StudentRepositoryInterface interface {
	GetStudentByNumber(ctx context.Context, studentNumber string) (db.Student, error)
	GetStudentByEmail(ctx context.Context, email string) (db.Student, error)
	GetStudentByID(ctx context.Context, id uuid.UUID) (db.Student, error)
	CreateStudentWithEvent(ctx context.Context, params db.CreateStudentParams, eventPayload map[string]any) (db.Student, error)
	UpdateStudentWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStudentParams, eventPayload map[string]any) (db.Student, error)
	SoftDeleteStudentWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]any) error
	ListStudentsFiltered(ctx context.Context, params db.ListStudentsParams) ([]db.Student, error)
	CountStudents(ctx context.Context) (int64, error)
	ListStudentsByAdvisor(ctx context.Context, advisorID uuid.UUID) ([]db.Student, error)
	ListOrphanedStudents(ctx context.Context, limit, offset int32) ([]db.Student, int64, error)
	BulkAssignAdvisor(ctx context.Context, studentIDs []uuid.UUID, advisorID uuid.UUID, advisorName string, eventPayloads []map[string]any) error
	SearchStudents(ctx context.Context, params db.SearchStudentsParams) ([]db.Student, error)
}

// StaffServiceInterface defines methods to interact with Staff Service
type StaffServiceInterface interface {
	ValidateAdvisor(ctx context.Context, advisorID uuid.UUID) error
	GetAdvisorInfo(ctx context.Context, advisorID uuid.UUID) (*AdvisorDetails, error)
	GetInstructorsByDepartment(ctx context.Context, department string) ([]uuid.UUID, error)
}

type StudentService struct {
	studentRepo  StudentRepositoryInterface
	staffService StaffServiceInterface
}

func NewStudentService(studentRepo StudentRepositoryInterface, staffService StaffServiceInterface) *StudentService {
	return &StudentService{
		studentRepo:  studentRepo,
		staffService: staffService,
	}
}

// CreateStudent creates a new student
func (s *StudentService) CreateStudent(ctx context.Context, req dto.CreateStudentRequest) (dto.StudentResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StudentService"),
		zap.String("method", "CreateStudent"),
		zap.String("student_number", req.StudentNumber),
		zap.String("email", req.Email),
	)

	// Check if student number already exists
	existingStudent, err := s.studentRepo.GetStudentByNumber(ctx, req.StudentNumber)
	if err != nil {
		// Check if error is wrapped query failure - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	if existingStudent.StudentNumber != "" {
		serviceLogger.Warn("student number already exists")
		return dto.StudentResponse{}, serviceErrors.ErrStudentNumberExists
	}

	// Check if email already exists
	existingStudent, err = s.studentRepo.GetStudentByEmail(ctx, req.Email)
	if err != nil {
		// Check if error is wrapped query failure - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	if existingStudent.Email != "" {
		serviceLogger.Warn("email already exists")
		return dto.StudentResponse{}, serviceErrors.ErrStudentEmailExists
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
	}

	eventPayload := buildStudentCreatedPayload(req, nil)

	// Validate and set advisor info if provided
	if req.AdvisorID != nil {
		advisorInfo, err := s.staffService.GetAdvisorInfo(ctx, *req.AdvisorID)
		if err != nil {
			serviceLogger.Warn("advisor validation failed",
				zap.Error(err),
				zap.String("advisor_id", req.AdvisorID.String()),
			)
			return dto.StudentResponse{}, serviceErrors.ErrAdvisorNotFound
		}
		params.AdvisorID = utils.UUIDToPgtype(*req.AdvisorID)
		params.AdvisorName = utils.StringToPgText(advisorInfo.Name)
		eventPayload["advisor_id"] = req.AdvisorID.String()
	}

	student, err := s.studentRepo.CreateStudentWithEvent(ctx, params, eventPayload)
	if err != nil {
		// Check for duplicate constraints
		if sharedErrors.Is(err, serviceErrors.ErrStudentNumberExistsRepo) {
			serviceLogger.Warn("duplicate student number detected during creation",
				zap.Error(err),
			)
			return dto.StudentResponse{}, serviceErrors.ErrStudentNumberExists
		}
		if sharedErrors.Is(err, serviceErrors.ErrStudentEmailExistsRepo) {
			serviceLogger.Warn("duplicate email detected during creation",
				zap.Error(err),
			)
			return dto.StudentResponse{}, serviceErrors.ErrStudentEmailExists
		}

		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("student created successfully in database",
		zap.String("student_id", utils.PgtypeToUUIDString(student.ID)),
		zap.String("student_number", student.StudentNumber),
	)

	return s.toStudentResponse(student), nil
}

// GetStudentByID retrieves student by ID
func (s *StudentService) GetStudentByID(ctx context.Context, id string) (dto.StudentResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StudentService"),
		zap.String("method", "GetStudentByID"),
		zap.String("student_id", id),
	)

	studentID, err := uuid.Parse(id)
	if err != nil {
		serviceLogger.Warn("invalid student ID format",
			zap.Error(err),
		)
		return dto.StudentResponse{}, sharedErrors.ErrInvalidID
	}

	student, err := s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		// Check if student not found
		if sharedErrors.Is(err, serviceErrors.ErrStudentNotFoundRepo) {
			serviceLogger.Warn("student not found in database",
				zap.Error(err),
			)
			return dto.StudentResponse{}, serviceErrors.ErrStudentNotFound
		}

		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("student retrieved successfully from database",
		zap.String("student_number", student.StudentNumber),
	)

	return s.toStudentResponse(student), nil
}

// UpdateStudent updates student information
func (s *StudentService) UpdateStudent(ctx context.Context, id string, req dto.UpdateStudentRequest) (dto.StudentResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StudentService"),
		zap.String("method", "UpdateStudent"),
		zap.String("student_id", id),
	)

	studentID, err := uuid.Parse(id)
	if err != nil {
		serviceLogger.Warn("invalid student ID format",
			zap.Error(err),
		)
		return dto.StudentResponse{}, sharedErrors.ErrInvalidID
	}

	// Check if student exists
	_, err = s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		// Check if student not found
		if sharedErrors.Is(err, serviceErrors.ErrStudentNotFoundRepo) {
			serviceLogger.Warn("student not found for update",
				zap.Error(err),
			)
			return dto.StudentResponse{}, serviceErrors.ErrStudentNotFound
		}

		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Validate and get advisor info if provided
	var advisorName string
	if req.AdvisorID != nil {
		advisorInfo, err := s.staffService.GetAdvisorInfo(ctx, *req.AdvisorID)
		if err != nil {
			serviceLogger.Warn("advisor validation failed",
				zap.Error(err),
				zap.String("advisor_id", req.AdvisorID.String()),
			)
			return dto.StudentResponse{}, serviceErrors.ErrAdvisorNotFound
		}
		advisorName = advisorInfo.Name
	}

	// Get current student data for COALESCE defaults
	currentStudent, _ := s.studentRepo.GetStudentByID(ctx, studentID)

	classLevel := currentStudent.ClassLevel
	if req.ClassLevel != nil {
		classLevel = *req.ClassLevel
	}

	params := db.UpdateStudentParams{
		ID:          utils.UUIDToPgtype(studentID),
		FirstName:   utils.PointerStringToPgText(req.FirstName),
		LastName:    utils.PointerStringToPgText(req.LastName),
		Email:       utils.PointerStringToPgText(req.Email),
		ClassLevel:  classLevel,
		AdvisorID:   utils.PointerUUIDToPgtype(req.AdvisorID),
		AdvisorName: utils.PointerStringToPgText(&advisorName),
		Status:      utils.PointerStringToPgText(req.Status),
	}

	changedFields := make(map[string]any)
	if req.FirstName != nil {
		changedFields["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		changedFields["last_name"] = *req.LastName
	}
	if req.Email != nil {
		changedFields["email"] = *req.Email
	}
	if req.ClassLevel != nil {
		changedFields["class_level"] = *req.ClassLevel
	}
	if req.AdvisorID != nil {
		changedFields["advisor_id"] = req.AdvisorID.String()
	}
	if req.Status != nil {
		changedFields["status"] = *req.Status
	}

	firstName := currentStudent.FirstName
	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	lastName := currentStudent.LastName
	if req.LastName != nil {
		lastName = *req.LastName
	}
	email := currentStudent.Email
	if req.Email != nil {
		email = *req.Email
	}

	eventPayload := buildStudentUpdatedPayload(StudentUpdatedInputs{
		ID:              id,
		Current:         currentStudent,
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		ClassLevel:      classLevel,
		ChangedFields:   changedFields,
		StatusOverride:  req.Status,
		AdvisorOverride: req.AdvisorID,
	})

	student, err := s.studentRepo.UpdateStudentWithEvent(ctx, studentID, params, eventPayload)
	if err != nil {
		// Check if student not found during update
		if sharedErrors.Is(err, serviceErrors.ErrStudentNotFoundRepo) {
			serviceLogger.Warn("student not found during update",
				zap.Error(err),
			)
			return dto.StudentResponse{}, serviceErrors.ErrStudentNotFound
		}

		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StudentResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("student updated successfully in database",
		zap.String("student_number", student.StudentNumber),
		zap.Bool("class_level_changed", req.ClassLevel != nil),
		zap.Bool("advisor_changed", req.AdvisorID != nil),
		zap.Bool("status_changed", req.Status != nil),
	)

	return s.toStudentResponse(student), nil
}

// DeleteStudent soft deletes a student
func (s *StudentService) DeleteStudent(ctx context.Context, id string) error {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StudentService"),
		zap.String("method", "DeleteStudent"),
		zap.String("student_id", id),
	)

	studentID, err := uuid.Parse(id)
	if err != nil {
		serviceLogger.Warn("invalid student ID format",
			zap.Error(err),
		)
		return sharedErrors.ErrInvalidID
	}

	// Check if student exists
	student, err := s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		// Check if student not found
		if sharedErrors.Is(err, serviceErrors.ErrStudentNotFoundRepo) {
			serviceLogger.Warn("student not found for deletion",
				zap.Error(err),
			)
			return serviceErrors.ErrStudentNotFound
		}

		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	eventPayload := buildStudentDeactivatedPayload(id, student.StudentNumber)

	err = s.studentRepo.SoftDeleteStudentWithEvent(ctx, studentID, eventPayload)
	if err != nil {
		// Check if student not found during deletion
		if sharedErrors.Is(err, serviceErrors.ErrStudentNotFoundRepo) {
			serviceLogger.Warn("student not found during deletion",
				zap.Error(err),
			)
			return serviceErrors.ErrStudentNotFound
		}

		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("student deleted successfully in database",
		zap.String("student_number", student.StudentNumber),
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
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StudentListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Get total count (we'll use existing CountStudents for now, ideally should count with filters)
	total, err := s.studentRepo.CountStudents(ctx)
	if err != nil {
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StudentListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StudentListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Initialize as empty slice to avoid null in JSON response
	studentResponses := make([]dto.StudentResponse, 0, len(students))
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
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.MyAdviseesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.MyAdviseesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	var studentResponses []dto.StudentResponse
	for _, student := range students {
		studentResponses = append(studentResponses, s.toStudentResponse(student))
	}

	return dto.MyAdviseesResponse{
		Advisor: dto.AdvisorInfo{
			AdvisorID: advisorID.String(),
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
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.OrphanedStudentsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.OrphanedStudentsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Initialize as empty slice to avoid null in JSON response
	studentResponses := make([]dto.StudentResponse, 0, len(students))
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
	// Get advisor info (validates and returns name)
	advisorInfo, err := s.staffService.GetAdvisorInfo(ctx, req.AdvisorID)
	if err != nil {
		logger.Warn("advisor validation failed",
			zap.Error(err),
			zap.String("advisor_id", req.AdvisorID.String()),
		)
		return dto.BulkAdvisorAssignResponse{}, serviceErrors.ErrAdvisorNotFound
	}

	// Create event payloads for each student
	eventPayloads := make([]map[string]any, len(req.StudentIDs))
	for i, studentID := range req.StudentIDs {
		eventPayloads[i] = map[string]any{
			"id":             studentID.String(),
			"student_number": "", // Will be filled from DB if needed
			"changed_fields": map[string]any{
				"advisor_id":   req.AdvisorID.String(),
				"advisor_name": advisorInfo.Name,
			},
		}
	}

	// Bulk assign
	err = s.studentRepo.BulkAssignAdvisor(ctx, req.StudentIDs, req.AdvisorID, advisorInfo.Name, eventPayloads)
	if err != nil {
		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.BulkAdvisorAssignResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.BulkAdvisorAssignResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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
			AdvisorID:   req.AdvisorID.String(),
			AdvisorName: advisorInfo.Name,
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
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.SearchStudentsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.SearchStudentsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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

	var advisorName *string
	if student.AdvisorName.Valid {
		advisorName = &student.AdvisorName.String
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
		AdvisorName:    advisorName,
		Status:         utils.PgTextToString(student.Status),
		CreatedAt:      student.CreatedAt.Time,
		UpdatedAt:      student.UpdatedAt.Time,
	}
}
