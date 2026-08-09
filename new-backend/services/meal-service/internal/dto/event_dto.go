package dto

import "time"

// ============================================================================
// CONSUMED EVENTS (Inbound from other services)
// ============================================================================

// StudentCreatedEvent represents student.created event from student service
type StudentCreatedEvent struct {
	EventType string                  `json:"event_type"`
	EventID   string                  `json:"event_id"`
	Timestamp time.Time               `json:"timestamp"`
	Data      StudentCreatedEventData `json:"data"`
}

type StudentCreatedEventData struct {
	ID            string `json:"id"`
	StudentNumber string `json:"student_number"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

// StudentUpdatedEvent represents student.updated event from student service
type StudentUpdatedEvent struct {
	EventType string                  `json:"event_type"`
	EventID   string                  `json:"event_id"`
	Timestamp time.Time               `json:"timestamp"`
	Data      StudentUpdatedEventData `json:"data"`
}

type StudentUpdatedEventData struct {
	ID            string `json:"id"`
	StudentNumber string `json:"student_number"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

// StudentDeactivatedEvent represents student.deactivated event from student service
type StudentDeactivatedEvent struct {
	EventType string                      `json:"event_type"`
	EventID   string                      `json:"event_id"`
	Timestamp time.Time                   `json:"timestamp"`
	Data      StudentDeactivatedEventData `json:"data"`
}

type StudentDeactivatedEventData struct {
	ID string `json:"id"`
}

// PaymentCompletedEvent represents payment.completed event from payment service
type PaymentCompletedEvent struct {
	EventType string                    `json:"event_type"`
	EventID   string                    `json:"event_id"`
	Timestamp time.Time                 `json:"timestamp"`
	Data      PaymentCompletedEventData `json:"data"`
}

type PaymentCompletedEventData struct {
	PaymentID   string  `json:"payment_id"`
	ReferenceID string  `json:"reference_id"` // "res_uuid" or "bat_uuid"
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
}

// PaymentFailedEvent represents payment.failed event from payment service
type PaymentFailedEvent struct {
	EventType string                 `json:"event_type"`
	EventID   string                 `json:"event_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      PaymentFailedEventData `json:"data"`
}

type PaymentFailedEventData struct {
	PaymentID   string `json:"payment_id"`
	ReferenceID string `json:"reference_id"` // "res_uuid" or "bat_uuid"
	Reason      string `json:"reason"`
}

// ============================================================================
// PUBLISHED EVENTS (Outbound to other services)
// ============================================================================

// MealReservationCreatedEvent represents meal.reservation.created event
type MealReservationCreatedEvent struct {
	EventType string                          `json:"event_type"`
	EventID   string                          `json:"event_id"`
	Timestamp time.Time                       `json:"timestamp"`
	Data      MealReservationCreatedEventData `json:"data"`
}

type MealReservationCreatedEventData struct {
	ReservationID string  `json:"reservation_id"`
	StudentID     string  `json:"student_id"`
	StudentNumber string  `json:"student_number"`
	Date          string  `json:"date"`
	MealTime      string  `json:"meal_time"`
	MenuType      string  `json:"menu_type"`
	CafeteriaID   string  `json:"cafeteria_id"`
	CafeteriaName string  `json:"cafeteria_name"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
}

// MealReservationCancelledEvent represents meal.reservation.cancelled event
type MealReservationCancelledEvent struct {
	EventType string                            `json:"event_type"`
	EventID   string                            `json:"event_id"`
	Timestamp time.Time                         `json:"timestamp"`
	Data      MealReservationCancelledEventData `json:"data"`
}

type MealReservationCancelledEventData struct {
	ReservationID string  `json:"reservation_id"`
	StudentID     string  `json:"student_id"`
	StudentNumber string  `json:"student_number"`
	Date          string  `json:"date"`
	MealTime      string  `json:"meal_time"`
	RefundAmount  float64 `json:"refund_amount"`
	Currency      string  `json:"currency"`
}
