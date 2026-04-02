package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/staff-service/internal/db"
	"github.com/baaaki/mydreamcampus/staff-service/internal/dto"
	"github.com/baaaki/mydreamcampus/staff-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	// Check if staff already exists
	existingStaff, err := s.staffRepo.GetStaffByEmail(ctx, req.Email)
	if err != nil && err != pgx.ErrNoRows {
		logger.Error("failed to check staff existence",
			zap.Error(err),
			zap.String("email", req.Email),
		)
		return dto.StaffResponse{}, errors.ErrInternal
	}
	if err == nil && existingStaff.Email != "" {
		logger.Warn("staff already exists",
			zap.String("email", req.Email),
		)
		return dto.StaffResponse{}, errors.ErrStaffExists
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

	eventPayload := map[string]interface{}{
		"staff_id":   nil, // Will be set after creation
		"email":      req.Email,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"role":       req.Role,
		"department": req.Department,
	}

	staff, err := s.staffRepo.CreateStaffWithEvent(ctx, params, eventPayload)
	if err != nil {
		logger.Error("failed to create staff",
			zap.Error(err),
			zap.String("email", req.Email),
		)
		return dto.StaffResponse{}, errors.ErrInternal
	}

	logger.Info("staff created successfully",
		zap.String("staff_id", uuid.UUID(staff.ID.Bytes).String()),
		zap.String("email", staff.Email),
	)

	return s.toStaffResponse(staff), nil
}

// GetStaffByID retrieves staff by ID
func (s *StaffService) GetStaffByID(ctx context.Context, id string) (dto.StaffResponse, error) {
	staffID, err := uuid.Parse(id)
	if err != nil {
		return dto.StaffResponse{}, errors.ErrInvalidID
	}

	staff, err := s.staffRepo.GetStaffByID(ctx, staffID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.StaffResponse{}, errors.ErrStaffNotFound
		}
		logger.Error("failed to get staff",
			zap.Error(err),
			zap.String("staff_id", id),
		)
		return dto.StaffResponse{}, errors.ErrInternal
	}

	return s.toStaffResponse(staff), nil
}

// UpdateStaff updates staff information
func (s *StaffService) UpdateStaff(ctx context.Context, id string, req dto.UpdateStaffRequest) (dto.StaffResponse, error) {
	staffID, err := uuid.Parse(id)
	if err != nil {
		return dto.StaffResponse{}, errors.ErrInvalidID
	}

	// Check if staff exists
	_, err = s.staffRepo.GetStaffByID(ctx, staffID)
	if err != nil {
		if err.Error() == "staff not found" {
			return dto.StaffResponse{}, errors.ErrStaffNotFound
		}
		return dto.StaffResponse{}, errors.ErrInternal
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

	eventPayload := map[string]interface{}{
		"staff_id":        id,
		"department":      req.Department,
		"phone":           req.Phone,
		"office_location": req.OfficeLocation,
	}

	staff, err := s.staffRepo.UpdateStaffWithEvent(ctx, staffID, params, eventPayload)
	if err != nil {
		logger.Error("failed to update staff",
			zap.Error(err),
			zap.String("staff_id", id),
		)
		return dto.StaffResponse{}, errors.ErrInternal
	}

	logger.Info("staff updated successfully",
		zap.String("staff_id", id),
	)

	return s.toStaffResponse(staff), nil
}

// DeleteStaff soft deletes a staff member
func (s *StaffService) DeleteStaff(ctx context.Context, id string) error {
	staffID, err := uuid.Parse(id)
	if err != nil {
		return errors.ErrInvalidID
	}

	// Check if staff exists
	_, err = s.staffRepo.GetStaffByID(ctx, staffID)
	if err != nil {
		if err.Error() == "staff not found" {
			return errors.ErrStaffNotFound
		}
		return errors.ErrInternal
	}

	eventPayload := map[string]interface{}{
		"staff_id": id,
	}

	err = s.staffRepo.SoftDeleteStaffWithEvent(ctx, staffID, eventPayload)
	if err != nil {
		logger.Error("failed to delete staff",
			zap.Error(err),
			zap.String("staff_id", id),
		)
		return errors.ErrInternal
	}

	logger.Info("staff deleted successfully",
		zap.String("staff_id", id),
	)

	return nil
}

// ListStaff lists staff with pagination
func (s *StaffService) ListStaff(ctx context.Context, query dto.PaginationQuery) (dto.StaffListResponse, error) {
	limit := int32(query.Limit)
	offset := int32((query.Page - 1) * query.Limit)

	staffList, total, err := s.staffRepo.ListStaff(ctx, limit, offset)
	if err != nil {
		logger.Error("failed to list staff",
			zap.Error(err),
		)
		return dto.StaffListResponse{}, errors.ErrInternal
	}

	var staffResponses []dto.StaffResponse
	for _, staff := range staffList {
		staffResponses = append(staffResponses, s.toStaffResponse(staff))
	}

	return dto.StaffListResponse{
		Data: staffResponses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      int(total),
			TotalPages: (int(total) + query.Limit - 1) / query.Limit,
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
