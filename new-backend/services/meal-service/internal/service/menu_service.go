package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	"github.com/baaaki/mydreamcampus/meal/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"go.uber.org/zap"
)

// MenuStore is the slice of MenuRepository the service uses.
type MenuStore interface {
	UpsertMonthlyMenu(ctx context.Context, params db.UpsertMonthlyMenuParams) (db.MonthlyMenu, error)
	GetMonthlyMenu(ctx context.Context, params db.GetMonthlyMenuParams) (db.MonthlyMenu, error)
}

var _ MenuStore = (*repository.MenuRepository)(nil)

type MenuService struct {
	menuRepo MenuStore
	logger   *zap.Logger
}

func NewMenuService(menuRepo MenuStore, logger *zap.Logger) *MenuService {
	return &MenuService{
		menuRepo: menuRepo,
		logger:   logger,
	}
}

// CreateOrUpdateMonthlyMenu creates or updates monthly menu (Admin only)
func (s *MenuService) CreateOrUpdateMonthlyMenu(ctx context.Context, req dto.CreateMonthlyMenuRequest) (*dto.MonthlyMenuResponse, error) {
	toSave := withDatedLunches(req.MenuData, req.Year, req.Month)
	if isWeekMenu(req.MenuData) {
		stored, err := s.storedMenuData(ctx, req.Year, req.Month)
		if err != nil {
			return nil, err
		}
		toSave = withDinnersKept(toSave, stored)
	}

	// Marshal menu data to JSONB
	menuDataJSON, err := json.Marshal(toSave)
	if err != nil {
		s.logger.Error("failed to marshal menu data", zap.Error(err))
		return nil, sharedErrors.ErrBadRequest
	}

	menu, err := s.menuRepo.UpsertMonthlyMenu(ctx, db.UpsertMonthlyMenuParams{
		Year:     utils.ClampToInt16(req.Year),
		Month:    utils.ClampToInt16(req.Month),
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

// storedMenuData is the month's saved menu, or nil before its first save.
func (s *MenuService) storedMenuData(ctx context.Context, year, month int) (map[string]any, error) {
	menu, err := s.menuRepo.GetMonthlyMenu(ctx, db.GetMonthlyMenuParams{
		Year:  utils.ClampToInt16(year),
		Month: utils.ClampToInt16(month),
	})
	if errors.Is(err, sharedErrors.ErrNotFoundRepo) {
		return nil, nil
	}
	if err != nil {
		s.logger.Error("failed to read monthly menu before saving", zap.Error(err))
		return nil, err
	}
	var stored map[string]any
	if err := json.Unmarshal(menu.MenuData, &stored); err != nil {
		s.logger.Error("failed to unmarshal stored menu data", zap.Error(err))
		return nil, sharedErrors.ErrQueryFailed
	}
	return stored, nil
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
		Year:  utils.ClampToInt16(year),
		Month: utils.ClampToInt16(month),
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
