package service

import (
	"context"
	"net/url"

	"github.com/baaaki/mydreamcampus/shared/client"
)

// ClosedDay is one non-serving day pushed to the meal service when a
// semester is created or edited.
type ClosedDay struct {
	Date   string `json:"date"`
	Reason string `json:"reason"`
}

// MealClient is the fan-out catalog performs on semester changes. Closed
// days live in meal's schema; catalog only announces them.
type MealClient interface {
	CreateClosedDays(ctx context.Context, days []ClosedDay) error
	ReplaceClosedDays(ctx context.Context, semester string, days []ClosedDay) error
	DeleteClosedDays(ctx context.Context, semester string) error
}

// HTTPMealClient rides the shared transport, so the timeout, the internal
// secret, the request-ID hand-off and the breaker are the same ones every
// other internal call gets.
type HTTPMealClient struct {
	base *client.Base
}

func NewHTTPMealClient(base *client.Base) *HTTPMealClient {
	return &HTTPMealClient{base: base}
}

type closedDaysPayload struct {
	ClosedDays []ClosedDay `json:"closed_days"`
}

func (c *HTTPMealClient) CreateClosedDays(ctx context.Context, days []ClosedDay) error {
	return c.base.Post(ctx, "/internal/closed-days/batch", closedDaysPayload{ClosedDays: days}, nil)
}

func (c *HTTPMealClient) ReplaceClosedDays(ctx context.Context, semester string, days []ClosedDay) error {
	return c.base.Put(ctx, "/internal/closed-days/by-semester/"+url.PathEscape(semester), closedDaysPayload{ClosedDays: days}, nil)
}

func (c *HTTPMealClient) DeleteClosedDays(ctx context.Context, semester string) error {
	return c.base.Delete(ctx, "/internal/closed-days/by-semester/"+url.PathEscape(semester))
}

var _ MealClient = (*HTTPMealClient)(nil)
