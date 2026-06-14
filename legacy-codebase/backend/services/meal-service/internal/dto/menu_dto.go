package dto

import (
	"time"
)

// CreateMonthlyMenuRequest represents request to create/update monthly menu
type CreateMonthlyMenuRequest struct {
	Year     int            `json:"year" binding:"required,min=2020,max=2100"`
	Month    int            `json:"month" binding:"required,min=1,max=12"`
	MenuData map[string]any `json:"menu_data" binding:"required"`
}

// MonthlyMenuResponse represents monthly menu response
type MonthlyMenuResponse struct {
	Year      int            `json:"year"`
	Month     int            `json:"month"`
	MenuData  map[string]any `json:"menu_data"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// GetMonthlyMenuQuery represents query parameters for fetching monthly menu
type GetMonthlyMenuQuery struct {
	Year  int `form:"year"`
	Month int `form:"month" binding:"omitempty,min=1,max=12"`
}
