package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"go.uber.org/zap"
)

// HTTPLogger calls the catalog service to write audit log entries.
// Used by enrollment, grades, and meal services.
type HTTPLogger struct {
	catalogBaseURL string
	httpClient     *http.Client
	serviceName    string
}

func NewHTTPLogger(catalogBaseURL, serviceName string) *HTTPLogger {
	return &HTTPLogger{
		catalogBaseURL: catalogBaseURL,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		serviceName:    serviceName,
	}
}

func (l *HTTPLogger) Log(ctx context.Context, event AuditEvent) error {
	event.Service = l.serviceName

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	url := fmt.Sprintf("%s/api/catalog/internal/audit-log", l.catalogBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		// Log but don't fail the main operation
		logger.Warn("failed to send audit log",
			zap.String("service", l.serviceName),
			zap.String("action", event.Action),
			zap.Error(err),
		)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		logger.Warn("audit log returned non-201 status",
			zap.String("service", l.serviceName),
			zap.Int("status", resp.StatusCode),
		)
	}

	return nil
}
