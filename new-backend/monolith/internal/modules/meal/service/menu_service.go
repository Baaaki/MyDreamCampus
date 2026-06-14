package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"go.uber.org/zap"
)

type MenuService struct {
	menuRepo *repository.MenuRepository
	logger   *zap.Logger
}

func NewMenuService(menuRepo *repository.MenuRepository, logger *zap.Logger) *MenuService {
	return &MenuService{
		menuRepo: menuRepo,
		logger:   logger,
	}
}

// CreateOrUpdateMonthlyMenu creates or updates monthly menu (Admin only)
func (s *MenuService) CreateOrUpdateMonthlyMenu(ctx context.Context, req dto.CreateMonthlyMenuRequest) (*dto.MonthlyMenuResponse, error) {
	// Marshal menu data to JSONB
	menuDataJSON, err := json.Marshal(req.MenuData)
	if err != nil {
		s.logger.Error("failed to marshal menu data", zap.Error(err))
		return nil, sharedErrors.ErrBadRequest
	}

	menu, err := s.menuRepo.UpsertMonthlyMenu(ctx, db.UpsertMonthlyMenuParams{
		Year:     int16(req.Year),
		Month:    int16(req.Month),
		MenuData: menuDataJSON,
	})
	if err != nil {
		s.logger.Error("failed to upsert monthly menu", zap.Error(err))
		return nil, err
	}

	// Unmarshal menu data for response
	var menuData map[string]any
	if err := json.Unmarshal(menu.MenuData, &menuData); err != nil {
		s.logger.Error("failed to unmarshal menu data", zap.Error(err))
		return nil, sharedErrors.ErrQueryFailed
	}

	s.logger.Info("monthly menu saved", zap.Int("year", req.Year), zap.Int("month", req.Month))

	return &dto.MonthlyMenuResponse{
		Year:      int(menu.Year),
		Month:     int(menu.Month),
		MenuData:  menuData,
		CreatedAt: menu.CreatedAt.Time,
		UpdatedAt: menu.UpdatedAt.Time,
	}, nil
}

// GetMonthlyMenu returns monthly menu (Public - no auth required)
func (s *MenuService) GetMonthlyMenu(ctx context.Context, year, month int) (*dto.MonthlyMenuResponse, error) {
	// If year/month not provided, use current date
	if year == 0 || month == 0 {
		now := clock.Now()
		if year == 0 {
			year = now.Year()
		}
		if month == 0 {
			month = int(now.Month())
		}
	}

	menu, err := s.menuRepo.GetMonthlyMenu(ctx, db.GetMonthlyMenuParams{
		Year:  int16(year),
		Month: int16(month),
	})
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFoundRepo) {
			return nil, sharedErrors.ErrNotFound
		}
		s.logger.Error("failed to get monthly menu", zap.Error(err))
		return nil, err
	}

	// Unmarshal menu data for response
	var menuData map[string]any
	if err := json.Unmarshal(menu.MenuData, &menuData); err != nil {
		s.logger.Error("failed to unmarshal menu data", zap.Error(err))
		return nil, sharedErrors.ErrQueryFailed
	}

	return &dto.MonthlyMenuResponse{
		Year:     int(menu.Year),
		Month:    int(menu.Month),
		MenuData: menuData,
	}, nil
}
