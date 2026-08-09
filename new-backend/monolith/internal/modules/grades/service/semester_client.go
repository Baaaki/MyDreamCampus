package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/shared/contracts"
)

// SemesterInfo is catalog's payload, not grades' own shape — the definition
// lives in shared/contracts so provider and consumer cannot drift.
type SemesterInfo = contracts.SemesterInfo

type SemesterClient interface {
	GetSemesterInfo(ctx context.Context, semester string) (*SemesterInfo, error)
}
