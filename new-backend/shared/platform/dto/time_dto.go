package dto

import "time"

// SimulateTimeRequest is the request body for POST /admin/time/simulate.
type SimulateTimeRequest struct {
	Time time.Time `json:"time" binding:"required"`
}

// TimeStatusResponse is the response body for GET /admin/time/status.
type TimeStatusResponse struct {
	Mode          string     `json:"mode"`
	CurrentTime   time.Time  `json:"current_time"`
	SimulatedTime *time.Time `json:"simulated_time,omitempty"`
}
