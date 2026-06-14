package service

import (
	"context"
	"errors"

	"github.com/baaaki/mydreamcampus/meal-service/internal/db"
	"github.com/baaaki/mydreamcampus/meal-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/meal-service/internal/errors"
	"github.com/baaaki/mydreamcampus/meal-service/internal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CafeteriaService struct {
	cafeteriaRepo *repository.CafeteriaRepository
	logger        *zap.Logger
}

func NewCafeteriaService(cafeteriaRepo *repository.CafeteriaRepository, logger *zap.Logger) *CafeteriaService {
	return &CafeteriaService{
		cafeteriaRepo: cafeteriaRepo,
		logger:        logger,
	}
}

// GetActiveCafeterias returns all active cafeterias
func (s *CafeteriaService) GetActiveCafeterias(ctx context.Context) (*dto.CafeteriaListResponse, error) {
	cafeterias, err := s.cafeteriaRepo.GetActiveCafeterias(ctx)
	if err != nil {
		s.logger.Error("failed to get active cafeterias", zap.Error(err))
		return nil, err
	}

	response := &dto.CafeteriaListResponse{
		Cafeterias: make([]dto.CafeteriaResponse, 0, len(cafeterias)),
	}

	for _, c := range cafeterias {
		response.Cafeterias = append(response.Cafeterias, dto.CafeteriaResponse{
			ID:           c.ID.String(),
			Name:         c.Name,
			Location:     c.Location,
			HasVeganMenu: c.HasVeganMenu,
			ServesDinner: c.ServesDinner,
			IsActive:     c.IsActive,
			CreatedAt:    c.CreatedAt.Time,
			UpdatedAt:    c.UpdatedAt.Time,
		})
	}

	return response, nil
}

// GetAllCafeterias returns all cafeterias including inactive ones (for admin)
func (s *CafeteriaService) GetAllCafeterias(ctx context.Context) (*dto.CafeteriaListResponse, error) {
	cafeterias, err := s.cafeteriaRepo.GetAllCafeterias(ctx)
	if err != nil {
		s.logger.Error("failed to get all cafeterias", zap.Error(err))
		return nil, err
	}

	response := &dto.CafeteriaListResponse{
		Cafeterias: make([]dto.CafeteriaResponse, 0, len(cafeterias)),
	}

	for _, c := range cafeterias {
		response.Cafeterias = append(response.Cafeterias, dto.CafeteriaResponse{
			ID:           c.ID.String(),
			Name:         c.Name,
			Location:     c.Location,
			HasVeganMenu: c.HasVeganMenu,
			ServesDinner: c.ServesDinner,
			IsActive:     c.IsActive,
			CreatedAt:    c.CreatedAt.Time,
			UpdatedAt:    c.UpdatedAt.Time,
		})
	}

	return response, nil
}

// GetCafeteriaByID returns cafeteria by ID
func (s *CafeteriaService) GetCafeteriaByID(ctx context.Context, id string) (*dto.CafeteriaResponse, error) {
	cafeteriaID, err := uuid.Parse(id)
	if err != nil {
		return nil, sharedErrors.ErrBadRequest
	}

	cafeteria, err := s.cafeteriaRepo.GetCafeteriaByID(ctx, cafeteriaID)
	if err != nil {
		if errors.Is(err, serviceErrors.ErrCafeteriaNotFoundRepo) {
			return nil, serviceErrors.ErrCafeteriaNotFound
		}
		s.logger.Error("failed to get cafeteria", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	return &dto.CafeteriaResponse{
		ID:           cafeteria.ID.String(),
		Name:         cafeteria.Name,
		Location:     cafeteria.Location,
		HasVeganMenu: cafeteria.HasVeganMenu,
		ServesDinner: cafeteria.ServesDinner,
		IsActive:     cafeteria.IsActive,
		CreatedAt:    cafeteria.CreatedAt.Time,
		UpdatedAt:    cafeteria.UpdatedAt.Time,
	}, nil
}

// CreateCafeteria creates a new cafeteria (Admin only)
func (s *CafeteriaService) CreateCafeteria(ctx context.Context, req dto.CreateCafeteriaRequest) (*dto.CafeteriaResponse, error) {
	cafeteria, err := s.cafeteriaRepo.CreateCafeteria(ctx, db.CreateCafeteriaParams{
		Name:         req.Name,
		Location:     req.Location,
		HasVeganMenu: req.HasVeganMenu,
		ServesDinner: req.ServesDinner,
		IsActive:     req.IsActive,
	})
	if err != nil {
		s.logger.Error("failed to create cafeteria", zap.Error(err))
		return nil, err
	}

	s.logger.Info("cafeteria created", zap.String("id", cafeteria.ID.String()), zap.String("name", cafeteria.Name))

	return &dto.CafeteriaResponse{
		ID:           cafeteria.ID.String(),
		Name:         cafeteria.Name,
		Location:     cafeteria.Location,
		HasVeganMenu: cafeteria.HasVeganMenu,
		ServesDinner: cafeteria.ServesDinner,
		IsActive:     cafeteria.IsActive,
		CreatedAt:    cafeteria.CreatedAt.Time,
		UpdatedAt:    cafeteria.UpdatedAt.Time,
	}, nil
}

// UpdateCafeteria updates a cafeteria (Admin only)
func (s *CafeteriaService) UpdateCafeteria(ctx context.Context, id string, req dto.UpdateCafeteriaRequest) (*dto.CafeteriaResponse, error) {
	cafeteriaID, err := uuid.Parse(id)
	if err != nil {
		return nil, sharedErrors.ErrBadRequest
	}

	cafeteria, err := s.cafeteriaRepo.UpdateCafeteria(ctx, db.UpdateCafeteriaParams{
		ID:           utils.UUIDToPgtype(cafeteriaID),
		Name:         req.Name,
		Location:     req.Location,
		HasVeganMenu: req.HasVeganMenu,
		ServesDinner: req.ServesDinner,
		IsActive:     req.IsActive,
	})
	if err != nil {
		if errors.Is(err, serviceErrors.ErrCafeteriaNotFoundRepo) {
			return nil, serviceErrors.ErrCafeteriaNotFound
		}
		s.logger.Error("failed to update cafeteria", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	s.logger.Info("cafeteria updated", zap.String("id", cafeteria.ID.String()), zap.String("name", cafeteria.Name))

	return &dto.CafeteriaResponse{
		ID:           cafeteria.ID.String(),
		Name:         cafeteria.Name,
		Location:     cafeteria.Location,
		HasVeganMenu: cafeteria.HasVeganMenu,
		ServesDinner: cafeteria.ServesDinner,
		IsActive:     cafeteria.IsActive,
		CreatedAt:    cafeteria.CreatedAt.Time,
		UpdatedAt:    cafeteria.UpdatedAt.Time,
	}, nil
}

// DeactivateCafeteria soft deletes a cafeteria (Admin only)
func (s *CafeteriaService) DeactivateCafeteria(ctx context.Context, id string) error {
	cafeteriaID, err := uuid.Parse(id)
	if err != nil {
		return sharedErrors.ErrBadRequest
	}

	err = s.cafeteriaRepo.DeactivateCafeteria(ctx, cafeteriaID)
	if err != nil {
		if errors.Is(err, serviceErrors.ErrCafeteriaNotFoundRepo) {
			return serviceErrors.ErrCafeteriaNotFound
		}
		s.logger.Error("failed to deactivate cafeteria", zap.Error(err), zap.String("id", id))
		return err
	}

	s.logger.Info("cafeteria deactivated", zap.String("id", id))
	return nil
}
