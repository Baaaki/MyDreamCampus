package service

import (
	"context"
	"time"

	ccService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	"github.com/baaaki/mydreamcampus/shared/contracts"
)

// SemesterInfo is catalog's payload, not attendance's own shape — the
// definition lives in shared/contracts so provider and consumer cannot drift.
type SemesterInfo = contracts.SemesterInfo

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

// Compile-time assertion — the in-process and HTTP clients must stay
// interchangeable.
var _ SemesterClient = (*InProcessSemesterClient)(nil)
