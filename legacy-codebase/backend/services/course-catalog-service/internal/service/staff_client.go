package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	catalogErrors "github.com/baaaki/mydreamcampus/course-catalog-service/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StaffClient interface for staff service communication
type StaffClient interface {
	GetInstructor(ctx context.Context, instructorID uuid.UUID, department string) (*InstructorInfo, error)
	GetInstructorsByDepartment(ctx context.Context, department string) ([]InstructorInfo, error)
}

// InstructorInfo represents instructor information from Staff Service
type InstructorInfo struct {
	ID         uuid.UUID `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	FullName   string    `json:"-"` // Computed field
	Department string    `json:"department"`
	Status     string    `json:"status"`
}

// HTTPStaffClient implements StaffClient using HTTP
type HTTPStaffClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPStaffClient creates a new HTTP staff client
func NewHTTPStaffClient(baseURL string) *HTTPStaffClient {
	return &HTTPStaffClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetInstructor retrieves and validates instructor information
func (c *HTTPStaffClient) GetInstructor(ctx context.Context, instructorID uuid.UUID, department string) (*InstructorInfo, error) {
	logger := logger.WithContextAndFields(ctx,
		zap.String("client", "HTTPStaffClient"),
		zap.String("method", "GetInstructor"),
		zap.String("instructor_id", instructorID.String()),
		zap.String("department", department),
	)

	// Fetch instructor by ID (using internal endpoint - no auth required)
	url := fmt.Sprintf("%s/internal/staff/%s", c.baseURL, instructorID.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.Error("failed to create HTTP request",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		logger.Warn("instructor not found in Staff Service",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, catalogErrors.ErrInstructorNotFound
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("unexpected status code from Staff Service",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed to read response body",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var instructor InstructorInfo
	if err := json.Unmarshal(body, &instructor); err != nil {
		logger.Error("failed to unmarshal response",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Validate instructor is active
	if instructor.Status != "active" {
		logger.Warn("instructor is not active",
			zap.String("status", instructor.Status),
		)
		return nil, catalogErrors.ErrInstructorNotActive
	}

	// Validate instructor is in the correct department
	if instructor.Department != department {
		logger.Warn("instructor not in department",
			zap.String("instructor_department", instructor.Department),
			zap.String("expected_department", department),
		)
		return nil, catalogErrors.ErrInstructorNotInDepartment
	}

	// Compute full name
	instructor.FullName = instructor.FirstName + " " + instructor.LastName

	logger.Info("instructor validated successfully",
		zap.String("full_name", instructor.FullName),
	)

	return &instructor, nil
}

// GetInstructorsByDepartment retrieves all active instructors in a department
func (c *HTTPStaffClient) GetInstructorsByDepartment(ctx context.Context, department string) ([]InstructorInfo, error) {
	logger := logger.WithContextAndFields(ctx,
		zap.String("client", "HTTPStaffClient"),
		zap.String("method", "GetInstructorsByDepartment"),
		zap.String("department", department),
	)

	// Fetch instructors by department (using internal endpoint - no auth required)
	url := fmt.Sprintf("%s/internal/staff/instructors?department=%s&status=active", c.baseURL, department)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.Error("failed to create HTTP request",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("unexpected status code from Staff Service",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed to read response body",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response struct {
		Data []InstructorInfo `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		logger.Error("failed to unmarshal response",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Compute full names
	for i := range response.Data {
		response.Data[i].FullName = response.Data[i].FirstName + " " + response.Data[i].LastName
	}

	logger.Info("instructors retrieved successfully",
		zap.Int("count", len(response.Data)),
	)

	return response.Data, nil
}
