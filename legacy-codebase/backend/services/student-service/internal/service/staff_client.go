package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StaffClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewStaffClient(baseURL string) *StaffClient {
	return &StaffClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// AdvisorDetails contains advisor information returned from staff service
type AdvisorDetails struct {
	ID   string
	Name string
}

// GetAdvisorInfo validates advisor and returns their full name
func (c *StaffClient) GetAdvisorInfo(ctx context.Context, advisorID uuid.UUID) (*AdvisorDetails, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffClient"),
		zap.String("method", "GetAdvisorInfo"),
	)

	// Use internal endpoint for service-to-service communication (no auth required)
	url := fmt.Sprintf("%s/internal/staff/%s", c.baseURL, advisorID.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("failed to create request",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to call staff service",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, fmt.Errorf("staff service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("advisor not found")
	}

	if resp.StatusCode != http.StatusOK {
		log.Error("staff service returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
		)
		return nil, fmt.Errorf("staff service error: status %d", resp.StatusCode)
	}

	// Parse response to check if staff is active and has teacher role
	var staffResponse struct {
		ID        string `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		Status    string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&staffResponse); err != nil {
		log.Error("failed to decode staff response",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if staffResponse.Role != "teacher" {
		return nil, fmt.Errorf("staff is not a teacher")
	}

	if staffResponse.Status != "active" {
		return nil, fmt.Errorf("advisor is not active")
	}

	return &AdvisorDetails{
		ID:   staffResponse.ID,
		Name: staffResponse.FirstName + " " + staffResponse.LastName,
	}, nil
}

// ValidateAdvisor validates if advisor exists and is active in Staff Service (backward compatible)
func (c *StaffClient) ValidateAdvisor(ctx context.Context, advisorID uuid.UUID) error {
	_, err := c.GetAdvisorInfo(ctx, advisorID)
	return err
}

// GetInstructorsByDepartment retrieves list of instructor IDs for a department
func (c *StaffClient) GetInstructorsByDepartment(ctx context.Context, department string) ([]uuid.UUID, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "StaffClient"),
		zap.String("method", "GetInstructorsByDepartment"),
	)

	url := fmt.Sprintf("%s/api/v1/staff/instructors?department=%s", c.baseURL, department)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("failed to create request",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to call staff service",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, fmt.Errorf("staff service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("staff service returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
		)
		return nil, fmt.Errorf("staff service error: status %d", resp.StatusCode)
	}

	// Parse response
	var instructorsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&instructorsResponse); err != nil {
		log.Error("failed to decode instructors response",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to UUID array
	var instructorIDs []uuid.UUID
	for _, instructor := range instructorsResponse.Data {
		id, err := uuid.Parse(instructor.ID)
		if err != nil {
			log.Warn("invalid instructor ID",
				zap.String("id", instructor.ID),
			)
			continue
		}
		instructorIDs = append(instructorIDs, id)
	}

	log.Info("fetched instructors for department",
		zap.String("department", department),
		zap.Int("count", len(instructorIDs)),
	)

	return instructorIDs, nil
}
