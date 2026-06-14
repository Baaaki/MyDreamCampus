package dto

import "time"

// CreateCafeteriaRequest represents the request body for creating a cafeteria
type CreateCafeteriaRequest struct {
	Name          string `json:"name" binding:"required"`
	Location      string `json:"location" binding:"required"`
	HasVeganMenu  bool   `json:"has_vegan_menu"`
	ServesDinner  bool   `json:"serves_dinner"`
	IsActive      bool   `json:"is_active"`
}

// UpdateCafeteriaRequest represents the request body for updating a cafeteria
type UpdateCafeteriaRequest struct {
	Name          string `json:"name" binding:"required"`
	Location      string `json:"location" binding:"required"`
	HasVeganMenu  bool   `json:"has_vegan_menu"`
	ServesDinner  bool   `json:"serves_dinner"`
	IsActive      bool   `json:"is_active"`
}

// CafeteriaResponse represents cafeteria response
type CafeteriaResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Location     string    `json:"location"`
	HasVeganMenu bool      `json:"has_vegan_menu"`
	ServesDinner bool      `json:"serves_dinner"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CafeteriaListResponse represents list of cafeterias
type CafeteriaListResponse struct {
	Cafeterias []CafeteriaResponse `json:"cafeterias"`
}

// GenerateQRRequest query parameters for QR generation
type GenerateQRRequest struct {
	Date     string `form:"date"`                                  // YYYY-MM-DD, defaults to today
	MealTime string `form:"meal_time" binding:"required,oneof=lunch dinner"`
}

// QRResponse represents QR code response
type QRResponse struct {
	CafeteriaID       string            `json:"cafeteria_id"`
	CafeteriaName     string            `json:"cafeteria_name"`
	Date              string            `json:"date"`
	MealTime          string            `json:"meal_time"`
	QRPayload         string            `json:"qr_payload"`
	ValidTimeWindow   ValidTimeWindow   `json:"valid_time_window"`
}

// ValidTimeWindow represents the time window when QR is valid
type ValidTimeWindow struct {
	Start string `json:"start"` // "11:00"
	End   string `json:"end"`   // "13:00"
}
