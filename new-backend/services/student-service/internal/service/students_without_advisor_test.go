package service

import (
	"context"
	"errors"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// orphanRepo serves n advisor-less students page by page.
type orphanRepo struct {
	StudentRepositoryInterface
	students []db.Student
	err      error
	calls    int
}

func (r *orphanRepo) ListOrphanedStudents(_ context.Context, limit, offset int32) ([]db.Student, int64, error) {
	r.calls++
	if r.err != nil {
		return nil, 0, r.err
	}
	end := min(int(offset)+int(limit), len(r.students))
	if int(offset) >= end {
		return []db.Student{}, int64(len(r.students)), nil
	}
	return r.students[offset:end], int64(len(r.students)), nil
}

func TestListStudentsWithoutAdvisor_MoreThanAPage_ReturnsAll(t *testing.T) {
	repo := &orphanRepo{}
	for range orphanedPageSize + 3 {
		repo.students = append(repo.students, db.Student{ID: utils.UUIDToPgtype(uuid.New())})
	}

	got, err := NewStudentService(repo, nil).ListStudentsWithoutAdvisor(context.Background())

	require.NoError(t, err)
	assert.Len(t, got, orphanedPageSize+3)
	assert.Equal(t, 2, repo.calls, "a short page ends the walk")
}

func TestListStudentsWithoutAdvisor_RepoFails_ReturnsError(t *testing.T) {
	repo := &orphanRepo{err: errors.New("connection reset")}

	_, err := NewStudentService(repo, nil).ListStudentsWithoutAdvisor(context.Background())

	assert.Error(t, err)
}
