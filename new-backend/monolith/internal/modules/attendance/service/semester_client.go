package service

import (
	"context"
	"time"

	ccService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
)

// SemesterInfo contains the essential semester data needed by attendance for enforcement.
type SemesterInfo struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	HardDeadline   time.Time `json:"hard_deadline"`
	IsPastDeadline bool      `json:"is_past_deadline"`
}

type SemesterClient interface {
	GetSemesterInfo(ctx context.Context, semester string) (*SemesterInfo, error)
}

type InProcessSemesterClient struct {
	semesterSvc *ccService.SemesterService
}

func NewInProcessSemesterClient(semesterSvc *ccService.SemesterService) *InProcessSemesterClient {
	return &InProcessSemesterClient{
		semesterSvc: semesterSvc,
	}
}

func (c *InProcessSemesterClient) GetSemesterInfo(ctx context.Context, semester string) (*SemesterInfo, error) {
	s, err := c.semesterSvc.GetSemesterByName(ctx, semester)
	if err != nil {
		return nil, err
	}
	
	// Convert cc db.Semester to SemesterInfo
	var hardDeadline time.Time
	if s.HardDeadline.Valid {
		hardDeadline = s.HardDeadline.Time
	}
	
	isPast := time.Now().After(hardDeadline)

	return &SemesterInfo{
		Name:           s.Name,
		Status:         string(s.Status),
		HardDeadline:   hardDeadline,
		IsPastDeadline: isPast,
	}, nil
}
