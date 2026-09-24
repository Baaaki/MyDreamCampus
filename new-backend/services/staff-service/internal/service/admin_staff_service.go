package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/shared/config"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// AdminStaffStore is the persistence AdminStaffService needs;
// repository.AdminStaffRepository implements it.
type AdminStaffStore interface {
	List(ctx context.Context, faculty string) ([]db.AdminStaff, error)
	GetByID(ctx context.Context, id uuid.UUID) (db.AdminStaff, error)
	Create(ctx context.Context, params db.CreateAdminStaffParams) (db.AdminStaff, error)
	Update(ctx context.Context, params db.UpdateAdminStaffParams) (db.AdminStaff, error)
}

// AdminStaffService manages the administrative staff directory. It publishes
// no events: these records never become accounts.
type AdminStaffService struct {
	store  AdminStaffStore
	config *config.Config
}

func NewAdminStaffService(store AdminStaffStore, cfg ...*config.Config) *AdminStaffService {
	var c *config.Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &AdminStaffService{store: store, config: c}
}

const startDateLayout = "2006-01-02"

func (s *AdminStaffService) List(ctx context.Context, faculty string) (dto.AdminStaffListResponse, error) {
	rows, err := s.store.List(ctx, faculty)
	if err != nil {
		return dto.AdminStaffListResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	resp := dto.AdminStaffListResponse{Data: make([]dto.AdminStaffResponse, 0, len(rows))}
	for _, row := range rows {
		resp.Data = append(resp.Data, toAdminStaffResponse(ctx, row))
	}
	return resp, nil
}

func (s *AdminStaffService) Get(ctx context.Context, rawID string) (dto.AdminStaffResponse, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return dto.AdminStaffResponse{}, sharedErrors.ErrInvalidID
	}
	row, err := s.store.GetByID(ctx, id)
	if err != nil {
		return dto.AdminStaffResponse{}, mapAdminStaffError(err)
	}
	return toAdminStaffResponse(ctx, row), nil
}

func (s *AdminStaffService) Create(ctx context.Context, req dto.AdminStaffRequest) (dto.AdminStaffResponse, error) {
	fields, err := adminStaffFields(req)
	if err != nil {
		return dto.AdminStaffResponse{}, err
	}
	row, err := s.store.Create(ctx, fields)
	if err != nil {
		return dto.AdminStaffResponse{}, mapAdminStaffError(err)
	}
	logger.WithContextAndFields(ctx, zap.String("service", "AdminStaffService")).
		Info("admin staff created", zap.String("admin_staff_id", utils.PgtypeToUUIDString(row.ID)))
	return toAdminStaffResponse(ctx, row), nil
}

func (s *AdminStaffService) Update(ctx context.Context, rawID string, req dto.AdminStaffRequest) (dto.AdminStaffResponse, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return dto.AdminStaffResponse{}, sharedErrors.ErrInvalidID
	}

	if s.config != nil {
		existing, err := s.store.GetByID(ctx, id)
		if err != nil {
			return dto.AdminStaffResponse{}, mapAdminStaffError(err)
		}
		if s.config.IsProtectedEmail(existing.Email) && req.Email != "" && !strings.EqualFold(req.Email, existing.Email) {
			return dto.AdminStaffResponse{}, serviceErrors.ErrProtectedAccountEmailChangeForbidden
		}
		if req.Email != "" && !strings.EqualFold(req.Email, existing.Email) && s.config.IsProtectedEmail(req.Email) {
			return dto.AdminStaffResponse{}, serviceErrors.ErrProtectedAccountEmailChangeForbidden
		}
	}

	fields, err := adminStaffFields(req)
	if err != nil {
		return dto.AdminStaffResponse{}, err
	}
	row, err := s.store.Update(ctx, db.UpdateAdminStaffParams{
		ID:               utils.UUIDToPgtype(id),
		Email:            fields.Email,
		Title:            fields.Title,
		FirstName:        fields.FirstName,
		LastName:         fields.LastName,
		Faculty:          fields.Faculty,
		Department:       fields.Department,
		Phone:            fields.Phone,
		ProfileImageUrl:  fields.ProfileImageUrl,
		Position:         fields.Position,
		JobDescription:   fields.JobDescription,
		Responsibilities: fields.Responsibilities,
		WorkingHours:     fields.WorkingHours,
		OfficeLocation:   fields.OfficeLocation,
		StartDate:        fields.StartDate,
	})
	if err != nil {
		return dto.AdminStaffResponse{}, mapAdminStaffError(err)
	}
	return toAdminStaffResponse(ctx, row), nil
}

// adminStaffFields converts a request into the column values create and
// update share. Binding has already checked the date format; parsing again
// here keeps the service safe to call without the handler.
func adminStaffFields(req dto.AdminStaffRequest) (db.CreateAdminStaffParams, error) {
	var start pgtype.Date
	if req.StartDate != "" {
		t, err := time.Parse(startDateLayout, req.StartDate)
		if err != nil {
			return db.CreateAdminStaffParams{}, sharedErrors.ErrValidation
		}
		start = pgtype.Date{Time: t, Valid: true}
	}
	responsibilities := req.Responsibilities
	if responsibilities == nil {
		responsibilities = []string{}
	}
	encoded, err := json.Marshal(responsibilities)
	if err != nil {
		return db.CreateAdminStaffParams{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	return db.CreateAdminStaffParams{
		Email:            req.Email,
		Title:            req.Title,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Faculty:          req.Faculty,
		Department:       req.Department,
		Phone:            req.Phone,
		ProfileImageUrl:  req.ProfileImageURL,
		Position:         req.Position,
		JobDescription:   req.JobDescription,
		Responsibilities: encoded,
		WorkingHours:     req.WorkingHours,
		OfficeLocation:   req.OfficeLocation,
		StartDate:        start,
	}, nil
}

func mapAdminStaffError(err error) error {
	switch {
	case sharedErrors.Is(err, sharedErrors.ErrNotFoundRepo):
		return serviceErrors.ErrAdminStaffNotFound
	case sharedErrors.Is(err, sharedErrors.ErrAlreadyExistsRepo):
		return serviceErrors.ErrEmailExists
	default:
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
}

func toAdminStaffResponse(ctx context.Context, row db.AdminStaff) dto.AdminStaffResponse {
	resp := dto.AdminStaffResponse{
		ID:               utils.PgtypeToUUIDString(row.ID),
		Email:            row.Email,
		Title:            row.Title,
		FirstName:        row.FirstName,
		LastName:         row.LastName,
		Faculty:          row.Faculty,
		Department:       row.Department,
		Phone:            row.Phone,
		ProfileImageURL:  row.ProfileImageUrl,
		Position:         row.Position,
		JobDescription:   row.JobDescription,
		Responsibilities: []string{},
		WorkingHours:     row.WorkingHours,
		OfficeLocation:   row.OfficeLocation,
		Status:           "active",
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
	if !row.IsActive {
		resp.Status = "inactive"
	}
	if row.StartDate.Valid {
		resp.StartDate = row.StartDate.Time.Format(startDateLayout)
	}
	if len(row.Responsibilities) > 0 {
		if err := json.Unmarshal(row.Responsibilities, &resp.Responsibilities); err != nil {
			// A corrupt value degrades to an empty list rather than failing the read.
			logger.WithContext(ctx).Warn("failed to decode admin staff responsibilities",
				zap.String("admin_staff_id", resp.ID), zap.Error(err))
			resp.Responsibilities = []string{}
		}
	}
	return resp
}
