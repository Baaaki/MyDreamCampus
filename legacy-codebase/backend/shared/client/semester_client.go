package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"go.uber.org/zap"
)

// SemesterInfo contains the essential semester data needed by other services for enforcement.
type SemesterInfo struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	HardDeadline   time.Time `json:"hard_deadline"`
	IsPastDeadline bool      `json:"is_past_deadline"`
}

type cacheEntry struct {
	info      *SemesterInfo
	expiresAt time.Time
}

// SemesterClient calls catalog service to fetch semester info (hard_deadline etc.)
// and caches responses in-memory to avoid repeated HTTP calls.
type SemesterClient struct {
	catalogBaseURL string
	httpClient     *http.Client
	cacheTTL       time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

func NewSemesterClient(catalogBaseURL string) *SemesterClient {
	return &SemesterClient{
		catalogBaseURL: catalogBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cacheTTL: 5 * time.Minute,
		cache:    make(map[string]cacheEntry),
	}
}

// GetSemesterInfo fetches semester info from catalog service, with in-memory caching.
func (c *SemesterClient) GetSemesterInfo(ctx context.Context, semesterName string) (*SemesterInfo, error) {
	// Check cache first
	c.mu.RLock()
	if entry, ok := c.cache[semesterName]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.info, nil
	}
	c.mu.RUnlock()

	// Fetch from catalog service
	url := fmt.Sprintf("%s/api/catalog/internal/semesters/%s/info", c.catalogBaseURL, semesterName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Forward request ID so a single trace spans caller → catalog.
	if rid := logger.GetRequestID(ctx); rid != "" {
		req.Header.Set("X-Request-ID", rid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch semester info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("semester not found: %s", semesterName)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from catalog service: %d", resp.StatusCode)
	}

	var info SemesterInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode semester info: %w", err)
	}

	// Update cache
	c.mu.Lock()
	c.cache[semesterName] = cacheEntry{
		info:      &info,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
	c.mu.Unlock()

	logger.Debug("semester info fetched from catalog service",
		zap.String("semester", semesterName),
		zap.String("status", info.Status),
		zap.Time("hard_deadline", info.HardDeadline),
	)

	return &info, nil
}

// InvalidateCache removes a semester from the cache.
func (c *SemesterClient) InvalidateCache(semesterName string) {
	c.mu.Lock()
	delete(c.cache, semesterName)
	c.mu.Unlock()
}
