package semester

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPChecker calls the catalog service to verify semester status.
// Used by enrollment, grades, and meal services.
type HTTPChecker struct {
	catalogBaseURL string
	httpClient     *http.Client
}

func NewHTTPChecker(catalogBaseURL string) *HTTPChecker {
	return &HTTPChecker{
		catalogBaseURL: catalogBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *HTTPChecker) IsSemesterActive(ctx context.Context, semesterName string) (bool, error) {
	url := fmt.Sprintf("%s/api/catalog/internal/semesters/%s/status", c.catalogBaseURL, semesterName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check semester status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Active, nil
}
