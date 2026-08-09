package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/baaaki/mydreamcampus/shared/client"
)

// HTTPSemesterClient reads semester info from the catalog service over
// internal REST.
type HTTPSemesterClient struct {
	base *client.Base
}

func NewHTTPSemesterClient(base *client.Base) *HTTPSemesterClient {
	return &HTTPSemesterClient{base: base}
}

// GetSemesterInfo fetches name/status/hard_deadline for enforcement.
// A 404 stays a plain error — the caller degrades on it. Everything else
// keeps client.ErrUnavailable in the chain, which the caller must not
// degrade on: an unreachable catalog is not a semester without a deadline.
func (c *HTTPSemesterClient) GetSemesterInfo(ctx context.Context, semester string) (*SemesterInfo, error) {
	var info SemesterInfo
	if err := c.base.Get(ctx, "/internal/semesters/"+url.PathEscape(semester)+"/info", &info); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return nil, fmt.Errorf("semester %q not found", semester)
		}
		return nil, err
	}
	return &info, nil
}

var _ SemesterClient = (*HTTPSemesterClient)(nil)
