package service

import (
	"context"
	"encoding/json"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	serviceErrors "github.com/baaaki/mydreamcampus/staff-service/internal/errors"
	"github.com/baaaki/mydreamcampus/staff-service/internal/db"
	"github.com/baaaki/mydreamcampus/staff-service/internal/dto"
	"github.com/baaaki/mydreamcampus/staff-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TeacherProfileService struct {
	profileRepo *repository.TeacherProfileRepository
}

func NewTeacherProfileService(profileRepo *repository.TeacherProfileRepository) *TeacherProfileService {
	return &TeacherProfileService{
		profileRepo: profileRepo,
	}
}

// GetTeacherProfileByStaffID retrieves teacher profile by staff ID (public endpoint)
func (s *TeacherProfileService) GetTeacherProfileByStaffID(ctx context.Context, staffID string) (dto.TeacherProfileResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "TeacherProfileService"),
		zap.String("method", "GetTeacherProfileByStaffID"),
		zap.String("staff_id", staffID),
	)

	id, err := uuid.Parse(staffID)
	if err != nil {
		serviceLogger.Warn("invalid staff ID format", zap.Error(err))
		return dto.TeacherProfileResponse{}, sharedErrors.ErrInvalidID
	}

	profile, err := s.profileRepo.GetTeacherProfileByStaffID(ctx, id)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrTeacherProfileNotFoundRepo) {
			serviceLogger.Warn("teacher profile not found", zap.Error(err))
			return dto.TeacherProfileResponse{}, serviceErrors.ErrTeacherProfileNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.TeacherProfileResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.TeacherProfileResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("teacher profile retrieved successfully")
	return s.toTeacherProfileResponse(profile), nil
}

// UpdateTeacherProfile updates teacher profile
func (s *TeacherProfileService) UpdateTeacherProfile(ctx context.Context, staffID string, req dto.UpdateTeacherProfileRequest) (dto.TeacherProfileResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "TeacherProfileService"),
		zap.String("method", "UpdateTeacherProfile"),
		zap.String("staff_id", staffID),
	)

	id, err := uuid.Parse(staffID)
	if err != nil {
		serviceLogger.Warn("invalid staff ID format", zap.Error(err))
		return dto.TeacherProfileResponse{}, sharedErrors.ErrInvalidID
	}

	// Build update params
	params := db.UpdateTeacherProfileParams{
		StaffID: utils.UUIDToPgtype(id),
	}

	if req.AcademicTitle != nil {
		params.AcademicTitle = utils.StringToPgText(*req.AcademicTitle)
	}
	if req.Faculty != nil {
		params.Faculty = utils.StringToPgText(*req.Faculty)
	}
	if req.ProfileImageURL != nil {
		params.ProfileImageUrl = utils.StringToPgText(*req.ProfileImageURL)
	}
	if req.Education != nil {
		data, _ := json.Marshal(req.Education)
		params.Education = data
	}
	if req.Articles != nil {
		data, _ := json.Marshal(req.Articles)
		params.Articles = data
	}
	if req.Bulletins != nil {
		data, _ := json.Marshal(req.Bulletins)
		params.Bulletins = data
	}
	if req.Projects != nil {
		data, _ := json.Marshal(req.Projects)
		params.Projects = data
	}
	if req.Awards != nil {
		data, _ := json.Marshal(req.Awards)
		params.Awards = data
	}
	if req.Scholarships != nil {
		data, _ := json.Marshal(req.Scholarships)
		params.Scholarships = data
	}
	if req.AdminAssignments != nil {
		data, _ := json.Marshal(req.AdminAssignments)
		params.AdminAssignments = data
	}

	_, err = s.profileRepo.UpdateTeacherProfile(ctx, params)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrTeacherProfileNotFoundRepo) {
			serviceLogger.Warn("teacher profile not found for update", zap.Error(err))
			return dto.TeacherProfileResponse{}, serviceErrors.ErrTeacherProfileNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.TeacherProfileResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.TeacherProfileResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Fetch updated profile with staff info
	updatedProfile, err := s.profileRepo.GetTeacherProfileByStaffID(ctx, id)
	if err != nil {
		return dto.TeacherProfileResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("teacher profile updated successfully")
	return s.toTeacherProfileResponse(updatedProfile), nil
}

// ListTeacherProfiles lists teacher profiles with pagination
func (s *TeacherProfileService) ListTeacherProfiles(ctx context.Context, query dto.PaginationQuery) (dto.TeacherProfileListResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "TeacherProfileService"),
		zap.String("method", "ListTeacherProfiles"),
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
	)

	limit := int32(query.Limit)
	offset := int32((query.Page - 1) * query.Limit)

	profiles, total, err := s.profileRepo.ListTeacherProfiles(ctx, limit, offset)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.TeacherProfileListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.TeacherProfileListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	var responses []dto.TeacherProfileResponse
	for _, profile := range profiles {
		responses = append(responses, s.listRowToTeacherProfileResponse(profile))
	}

	totalPages := (int(total) + query.Limit - 1) / query.Limit

	serviceLogger.Info("teacher profiles list retrieved",
		zap.Int("total_records", int(total)),
		zap.Int("returned_records", len(responses)),
	)

	return dto.TeacherProfileListResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

// toTeacherProfileResponse converts db row to dto response
func (s *TeacherProfileService) toTeacherProfileResponse(row db.GetTeacherProfileByStaffIDRow) dto.TeacherProfileResponse {
	response := dto.TeacherProfileResponse{
		ID:              utils.PgtypeToUUIDString(row.ID),
		StaffID:         utils.PgtypeToUUIDString(row.StaffID),
		AcademicTitle:   utils.PgTextToString(row.AcademicTitle),
		FirstName:       row.FirstName,
		LastName:        row.LastName,
		Faculty:         utils.PgTextToString(row.Faculty),
		Department:      utils.PgTextToString(row.Department),
		Email:           row.Email,
		Phone:           utils.PgTextToString(row.Phone),
		OfficeLocation:  utils.PgTextToString(row.OfficeLocation),
		ProfileImageURL: utils.PgTextToString(row.ProfileImageUrl),
		Education:       []dto.Education{},
		Articles:        []dto.Article{},
		Bulletins:       []dto.Bulletin{},
		Projects:        []dto.Project{},
		Awards:          []dto.Award{},
		Scholarships:    []dto.Scholarship{},
		AdminAssignments: []dto.AdminAssignment{},
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	// Parse JSONB fields
	if len(row.Education) > 0 {
		json.Unmarshal(row.Education, &response.Education)
	}
	if len(row.Articles) > 0 {
		json.Unmarshal(row.Articles, &response.Articles)
	}
	if len(row.Bulletins) > 0 {
		json.Unmarshal(row.Bulletins, &response.Bulletins)
	}
	if len(row.Projects) > 0 {
		json.Unmarshal(row.Projects, &response.Projects)
	}
	if len(row.Awards) > 0 {
		json.Unmarshal(row.Awards, &response.Awards)
	}
	if len(row.Scholarships) > 0 {
		json.Unmarshal(row.Scholarships, &response.Scholarships)
	}
	if len(row.AdminAssignments) > 0 {
		json.Unmarshal(row.AdminAssignments, &response.AdminAssignments)
	}

	return response
}

// listRowToTeacherProfileResponse converts list row to dto response
func (s *TeacherProfileService) listRowToTeacherProfileResponse(row db.ListTeacherProfilesRow) dto.TeacherProfileResponse {
	response := dto.TeacherProfileResponse{
		ID:              utils.PgtypeToUUIDString(row.ID),
		StaffID:         utils.PgtypeToUUIDString(row.StaffID),
		AcademicTitle:   utils.PgTextToString(row.AcademicTitle),
		FirstName:       row.FirstName,
		LastName:        row.LastName,
		Faculty:         utils.PgTextToString(row.Faculty),
		Department:      utils.PgTextToString(row.Department),
		Email:           row.Email,
		Phone:           utils.PgTextToString(row.Phone),
		OfficeLocation:  utils.PgTextToString(row.OfficeLocation),
		ProfileImageURL: utils.PgTextToString(row.ProfileImageUrl),
		Education:       []dto.Education{},
		Articles:        []dto.Article{},
		Bulletins:       []dto.Bulletin{},
		Projects:        []dto.Project{},
		Awards:          []dto.Award{},
		Scholarships:    []dto.Scholarship{},
		AdminAssignments: []dto.AdminAssignment{},
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	// Parse JSONB fields
	if len(row.Education) > 0 {
		json.Unmarshal(row.Education, &response.Education)
	}
	if len(row.Articles) > 0 {
		json.Unmarshal(row.Articles, &response.Articles)
	}
	if len(row.Bulletins) > 0 {
		json.Unmarshal(row.Bulletins, &response.Bulletins)
	}
	if len(row.Projects) > 0 {
		json.Unmarshal(row.Projects, &response.Projects)
	}
	if len(row.Awards) > 0 {
		json.Unmarshal(row.Awards, &response.Awards)
	}
	if len(row.Scholarships) > 0 {
		json.Unmarshal(row.Scholarships, &response.Scholarships)
	}
	if len(row.AdminAssignments) > 0 {
		json.Unmarshal(row.AdminAssignments, &response.AdminAssignments)
	}

	return response
}
