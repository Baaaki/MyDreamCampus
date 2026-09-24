package dto

import "time"

// SimulateTimeRequest is the request body for POST /admin/time/simulate.
type SimulateTimeRequest struct {
	Time time.Time `json:"time" binding:"required"`
}

// TimeStatusResponse is the response body for GET /admin/time/status and
// for simulate/reset. CurrentTime and RealTime come from one snapshot, so
// their difference is exactly the offset; comparing it across services
// shows drift without the request latency in between.
type TimeStatusResponse struct {
	Service       string     `json:"service"`
	Active        bool       `json:"active"`
	CurrentTime   time.Time  `json:"current_time"`
	RealTime      time.Time  `json:"real_time"`
	OffsetSeconds int64      `json:"offset_seconds"`
	Until         *time.Time `json:"until,omitempty"`
}
