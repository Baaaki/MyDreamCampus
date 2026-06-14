package dto

import "time"

// CreateReservationRequest represents single reservation request
type CreateReservationRequest struct {
	CafeteriaID string `json:"cafeteria_id" binding:"required,uuid"`
	Date        string `json:"date" binding:"required"` // YYYY-MM-DD
	MealTime    string `json:"meal_time" binding:"required,oneof=lunch dinner"`
	MenuType    string `json:"menu_type" binding:"required,oneof=normal vegan"`
}

// BatchReservationRequest represents batch reservation request
type BatchReservationRequest struct {
	Reservations []CreateReservationRequest `json:"reservations" binding:"required,min=1,max=10,dive"`
}

// ReservationResponse represents reservation details
type ReservationResponse struct {
	ID            string         `json:"id"`
	Date          string         `json:"date"`
	MealTime      string         `json:"meal_time"`
	MenuType      string         `json:"menu_type"`
	CafeteriaName string         `json:"cafeteria_name"`
	Cafeteria     *CafeteriaInfo `json:"cafeteria,omitempty"`
	Status        string         `json:"status"`
	IsUsed        bool           `json:"is_used"`
	CreatedAt     time.Time      `json:"created_at"`
}

// CafeteriaInfo embedded cafeteria info
type CafeteriaInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

// CreateReservationResponse represents response after creating reservation
type CreateReservationResponse struct {
	ReservationID string              `json:"reservation_id"`
	PaymentURL    string              `json:"payment_url"`
	Amount        float64             `json:"amount"`
	Currency      string              `json:"currency"`
	ExpiresAt     time.Time           `json:"expires_at"`
	Reservation   ReservationResponse `json:"reservation"`
}

// CreateBatchReservationResponse represents response after creating batch reservations
type CreateBatchReservationResponse struct {
	BatchID      string                `json:"batch_id"`
	PaymentURL   string                `json:"payment_url"`
	TotalAmount  float64               `json:"total_amount"`
	Currency     string                `json:"currency"`
	ExpiresAt    time.Time             `json:"expires_at"`
	Reservations []ReservationResponse `json:"reservations"`
}

// MyReservationsQuery represents query parameters for fetching user's reservations
type MyReservationsQuery struct {
	FromDate string `form:"from_date"` // YYYY-MM-DD
	ToDate   string `form:"to_date"`   // YYYY-MM-DD
	Status   string `form:"status" binding:"omitempty,oneof=pending confirmed cancelled expired"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// MyReservationsResponse represents user's reservations with summary
type MyReservationsResponse struct {
	Reservations []ReservationResponse `json:"reservations"`
	Summary      ReservationSummary    `json:"summary"`
	Pagination   *PaginationInfo       `json:"pagination,omitempty"`
}

// ReservationSummary represents summary of user's reservations
type ReservationSummary struct {
	Total     int `json:"total"`
	Confirmed int `json:"confirmed"`
	Pending   int `json:"pending"`
	Used      int `json:"used"`
	Cancelled int `json:"cancelled"`
}

// CancelReservationResponse represents response after cancelling reservation
type CancelReservationResponse struct {
	ReservationID string  `json:"reservation_id"`
	RefundAmount  float64 `json:"refund_amount"`
	Currency      string  `json:"currency"`
	RefundStatus  string  `json:"refund_status"`
}

// UseReservationRequest represents QR scan request
type UseReservationRequest struct {
	QRPayload string `json:"qr_payload" binding:"required"`
}

// UseReservationResponse represents response after using reservation
type UseReservationResponse struct {
	Message       string `json:"message"`
	ReservationID string `json:"reservation_id"`
	CafeteriaName string `json:"cafeteria_name"`
	MealTime      string `json:"meal_time"`
	MenuType      string `json:"menu_type"`
}

// ValidationError represents a single validation error in batch request
type ValidationError struct {
	Index    int    `json:"index"`
	Date     string `json:"date"`
	MealTime string `json:"meal_time"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// ReservationConflict represents a reservation conflict in batch request
type ReservationConflict struct {
	Date                  string `json:"date"`
	MealTime              string `json:"meal_time"`
	ExistingReservationID string `json:"existing_reservation_id"`
	CafeteriaName         string `json:"cafeteria_name"`
	Status                string `json:"status"`
}

// BatchValidationErrorResponse represents response when batch validation fails
type BatchValidationErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors"`
}

// BatchConflictErrorResponse represents response when batch has conflicts
type BatchConflictErrorResponse struct {
	Code      string                `json:"code"`
	Message   string                `json:"message"`
	Conflicts []ReservationConflict `json:"conflicts"`
}
