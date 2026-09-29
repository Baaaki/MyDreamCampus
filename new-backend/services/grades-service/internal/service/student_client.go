package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/google/uuid"
)

// StudentClient reads a student from the student service, which owns the
// advisor assignment. Grades keeps no copy of it: an advisor change or
// unassignment would reach a projection late, and the transcript check must
// not act on a stale advisor.
type StudentClient interface {
	GetStudentByID(ctx context.Context, id uuid.UUID) (contracts.StudentResponse, error)
}
