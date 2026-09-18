package service

import (
	"context"
	"encoding/json"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/baaaki/mydreamcampus/staff/internal/repository"
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

// decodeJSONBField decodes a denormalized JSONB column. A corrupt value
// degrades to the field's zero value instead of failing the whole read.
func decodeJSONBField(raw []byte, dst any, field string) {
	if len(raw) == 0 {
		return
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		logger.Warn("failed to decode teacher profile JSONB field",
			zap.String("field", field), zap.Error(err))
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

// toTeacherProfileResponse converts db row to dto response
func (s *TeacherProfileService) toTeacherProfileResponse(row db.GetTeacherProfileByStaffIDRow) dto.TeacherProfileResponse {
	response := dto.TeacherProfileResponse{
		ID:               utils.PgtypeToUUIDString(row.ID),
		StaffID:          utils.PgtypeToUUIDString(row.StaffID),
		AcademicTitle:    utils.PgTextToString(row.AcademicTitle),
		FirstName:        row.FirstName,
		LastName:         row.LastName,
		Faculty:          utils.PgTextToString(row.Faculty),
		Department:       utils.PgTextToString(row.Department),
		Email:            row.Email,
		Phone:            utils.PgTextToString(row.Phone),
		OfficeLocation:   utils.PgTextToString(row.OfficeLocation),
		ProfileImageURL:  utils.PgTextToString(row.ProfileImageUrl),
		Education:        []dto.Education{},
		Articles:         []dto.Article{},
		Bulletins:        []dto.Bulletin{},
		Projects:         []dto.Project{},
		Awards:           []dto.Award{},
		Scholarships:     []dto.Scholarship{},
		AdminAssignments: []dto.AdminAssignment{},
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}

	// Parse JSONB fields
	decodeJSONBField(row.Education, &response.Education, "education")
	decodeJSONBField(row.Articles, &response.Articles, "articles")
	decodeJSONBField(row.Bulletins, &response.Bulletins, "bulletins")
	decodeJSONBField(row.Projects, &response.Projects, "projects")
	decodeJSONBField(row.Awards, &response.Awards, "awards")
	decodeJSONBField(row.Scholarships, &response.Scholarships, "scholarships")
	decodeJSONBField(row.AdminAssignments, &response.AdminAssignments, "admin_assignments")

	return response
}
