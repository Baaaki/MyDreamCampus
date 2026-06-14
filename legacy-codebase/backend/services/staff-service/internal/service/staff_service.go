package service

import (
	"context"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/staff-service/internal/db"
	"github.com/baaaki/mydreamcampus/staff-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/staff-service/internal/errors"
	"github.com/baaaki/mydreamcampus/staff-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type StaffService struct {
	staffRepo *repository.StaffRepository
}

func NewStaffService(staffRepo *repository.StaffRepository) *StaffService {
	return &StaffService{
		staffRepo: staffRepo,
	}
}

// CreateStaff creates a new staff member
func (s *StaffService) CreateStaff(ctx context.Context, req dto.CreateStaffRequest) (dto.StaffResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffService"),
		zap.String("method", "CreateStaff"),
		zap.String("email", req.Email),
	)

	// Check if staff already exists
	existingStaff, err := s.staffRepo.GetStaffByEmail(ctx, req.Email)
	if err != nil {
		// Check if error is wrapped query failure - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// If staff exists (email not empty), return conflict error
	if existingStaff.Email != "" {
		serviceLogger.Warn("staff already exists",
			zap.String("existing_email", existingStaff.Email),
		)
		return dto.StaffResponse{}, serviceErrors.ErrStaffExists
	}

	// Create staff with outbox event
	params := db.CreateStaffParams{
		Email:          req.Email,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Role:           req.Role,
		Department:     utils.StringToPgText(req.Department),
		Phone:          utils.StringToPgText(req.Phone),
		OfficeLocation: utils.StringToPgText(req.OfficeLocation),
	}

	eventPayload := buildStaffCreatedPayload(req)

	staff, err := s.staffRepo.CreateStaffWithEvent(ctx, params, eventPayload)
	if err != nil {
		// Check for duplicate email constraint violation
		if sharedErrors.Is(err, serviceErrors.ErrStaffExistsRepo) {
			serviceLogger.Warn("duplicate email detected during creation",
				zap.Error(err),
			)
			return dto.StaffResponse{}, serviceErrors.ErrEmailExists
		}

		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("staff created successfully in database",
		zap.String("staff_id", uuid.UUID(staff.ID.Bytes).String()),
		zap.String("role", staff.Role),
	)

	return s.toStaffResponse(staff), nil
}

// GetStaffByID retrieves staff by ID
func (s *StaffService) GetStaffByID(ctx context.Context, id string) (dto.StaffResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffService"),
		zap.String("method", "GetStaffByID"),
		zap.String("staff_id", id),
	)

	staffID, err := uuid.Parse(id)
	if err != nil {
		serviceLogger.Warn("invalid staff ID format",
			zap.Error(err),
		)
		return dto.StaffResponse{}, sharedErrors.ErrInvalidID
	}

	staff, err := s.staffRepo.GetStaffByID(ctx, staffID)
	if err != nil {
		// Check if staff not found
		if sharedErrors.Is(err, serviceErrors.ErrStaffNotFoundRepo) {
			serviceLogger.Warn("staff not found in database",
				zap.Error(err),
			)
			return dto.StaffResponse{}, serviceErrors.ErrStaffNotFound
		}

		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("staff retrieved successfully from database",
		zap.String("email", staff.Email),
	)

	return s.toStaffResponse(staff), nil
}

// UpdateStaff updates staff information
func (s *StaffService) UpdateStaff(ctx context.Context, id string, req dto.UpdateStaffRequest) (dto.StaffResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffService"),
		zap.String("method", "UpdateStaff"),
		zap.String("staff_id", id),
	)

	staffID, err := uuid.Parse(id)
	if err != nil {
		serviceLogger.Warn("invalid staff ID format",
			zap.Error(err),
		)
		return dto.StaffResponse{}, sharedErrors.ErrInvalidID
	}

	// Check if staff exists
	existingStaff, err := s.staffRepo.GetStaffByID(ctx, staffID)
	if err != nil {
		// Check if staff not found
		if sharedErrors.Is(err, serviceErrors.ErrStaffNotFoundRepo) {
			serviceLogger.Warn("staff not found for update",
				zap.Error(err),
			)
			return dto.StaffResponse{}, serviceErrors.ErrStaffNotFound
		}

		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	params := db.UpdateStaffParams{
		ID: pgtype.UUID{
			Bytes: staffID,
			Valid: true,
		},
		Department:     utils.PointerStringToPgText(req.Department),
		Phone:          utils.PointerStringToPgText(req.Phone),
		OfficeLocation: utils.PointerStringToPgText(req.OfficeLocation),
	}

	eventPayload := buildStaffUpdatedPayload(StaffUpdatedInputs{
		ID:             id,
		Department:     req.Department,
		Phone:          req.Phone,
		OfficeLocation: req.OfficeLocation,
	})

	staff, err := s.staffRepo.UpdateStaffWithEvent(ctx, staffID, params, eventPayload)
	if err != nil {
		// Check if staff not found during update
		if sharedErrors.Is(err, serviceErrors.ErrStaffNotFoundRepo) {
			serviceLogger.Warn("staff not found during update",
				zap.Error(err),
			)
			return dto.StaffResponse{}, serviceErrors.ErrStaffNotFound
		}

		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StaffResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("staff updated successfully in database",
		zap.String("email", existingStaff.Email),
		zap.Bool("department_changed", req.Department != nil),
		zap.Bool("phone_changed", req.Phone != nil),
		zap.Bool("office_changed", req.OfficeLocation != nil),
	)

	return s.toStaffResponse(staff), nil
}

// DeleteStaff soft deletes a staff member
func (s *StaffService) DeleteStaff(ctx context.Context, id string) error {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffService"),
		zap.String("method", "DeleteStaff"),
		zap.String("staff_id", id),
	)

	staffID, err := uuid.Parse(id)
	if err != nil {
		serviceLogger.Warn("invalid staff ID format",
			zap.Error(err),
		)
		return sharedErrors.ErrInvalidID
	}

	// Check if staff exists
	existingStaff, err := s.staffRepo.GetStaffByID(ctx, staffID)
	if err != nil {
		// Check if staff not found
		if sharedErrors.Is(err, serviceErrors.ErrStaffNotFoundRepo) {
			serviceLogger.Warn("staff not found for deletion",
				zap.Error(err),
			)
			return serviceErrors.ErrStaffNotFound
		}

		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	eventPayload := buildStaffDeactivatedPayload(id)

	err = s.staffRepo.SoftDeleteStaffWithEvent(ctx, staffID, eventPayload)
	if err != nil {
		// Check if staff not found during deletion
		if sharedErrors.Is(err, serviceErrors.ErrStaffNotFoundRepo) {
			serviceLogger.Warn("staff not found during deletion",
				zap.Error(err),
			)
			return serviceErrors.ErrStaffNotFound
		}

		// Check for transaction or query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrTransactionFailed) || sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("staff deleted successfully in database",
		zap.String("email", existingStaff.Email),
		zap.String("role", existingStaff.Role),
	)

	return nil
}

// ListStaff lists staff with pagination
func (s *StaffService) ListStaff(ctx context.Context, query dto.PaginationQuery) (dto.StaffListResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffService"),
		zap.String("method", "ListStaff"),
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
	)

	limit := int32(query.Limit)
	offset := int32((query.Page - 1) * query.Limit)

	staffList, total, err := s.staffRepo.ListStaff(ctx, limit, offset)
	if err != nil {
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StaffListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	var staffResponses []dto.StaffResponse
	for _, staff := range staffList {
		staffResponses = append(staffResponses, s.toStaffResponse(staff))
	}

	totalPages := (int(total) + query.Limit - 1) / query.Limit

	serviceLogger.Info("staff list retrieved successfully from database",
		zap.Int("total_records", int(total)),
		zap.Int("returned_records", len(staffResponses)),
		zap.Int("total_pages", totalPages),
	)

	return dto.StaffListResponse{
		Data: staffResponses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

// GetInstructorsByDepartment retrieves instructors by department
func (s *StaffService) GetInstructorsByDepartment(ctx context.Context, department string) (dto.StaffListResponse, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffService"),
		zap.String("method", "GetInstructorsByDepartment"),
		zap.String("department", department),
	)

	instructors, err := s.staffRepo.GetInstructorsByDepartment(ctx, department)
	if err != nil {
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.StaffListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		// Unexpected error - wrap and return, handler will log
		return dto.StaffListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	var instructorResponses []dto.StaffResponse
	for _, instructor := range instructors {
		instructorResponses = append(instructorResponses, s.toStaffResponse(instructor))
	}

	serviceLogger.Info("instructors retrieved successfully from database",
		zap.Int("count", len(instructorResponses)),
	)

	return dto.StaffListResponse{
		Data: instructorResponses,
		Pagination: dto.PaginationResponse{
			Page:       1,
			Limit:      len(instructorResponses),
			Total:      len(instructorResponses),
			TotalPages: 1,
		},
	}, nil
}

// toStaffResponse converts db.Staff to dto.StaffResponse
func (s *StaffService) toStaffResponse(staff db.Staff) dto.StaffResponse {
	status := "active"
	if !staff.IsActive {
		status = "inactive"
	}

	return dto.StaffResponse{
		ID:             utils.PgtypeToUUIDString(staff.ID),
		Email:          staff.Email,
		FirstName:      staff.FirstName,
		LastName:       staff.LastName,
		Role:           staff.Role,
		Department:     utils.PgTextToString(staff.Department),
		Phone:          utils.PgTextToString(staff.Phone),
		OfficeLocation: utils.PgTextToString(staff.OfficeLocation),
		Status:         status,
		CreatedAt:      staff.CreatedAt.Time,
		UpdatedAt:      staff.UpdatedAt.Time,
	}
}
