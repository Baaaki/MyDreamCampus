//go:build integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/enrollment/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const itSemester = "2026-2027-Fall"

// catalogByID answers course lookups from a fixed set, the way catalog does:
// an unknown id is a 404.
type catalogByID map[uuid.UUID]contracts.SemesterCourseResponse

func (c catalogByID) GetAvailableCourses(context.Context, string, int16, string) ([]contracts.SemesterCourseListItem, error) {
	return nil, nil
}

func (c catalogByID) GetCoursesByIDs(_ context.Context, _ string, ids []uuid.UUID) ([]contracts.SemesterCourseResponse, error) {
	courses := make([]contracts.SemesterCourseResponse, 0, len(ids))
	for _, id := range ids {
		course, ok := c[id]
		if !ok {
			return nil, fmt.Errorf("course not found: %w", client.ErrNotFound)
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func semesterCourse(code, department string, slot int16, prerequisites ...contracts.Prerequisite) contracts.SemesterCourseResponse {
	return contracts.SemesterCourseResponse{
		ID: uuid.New(), Semester: itSemester, CourseCode: code, CourseName: code,
		Department: department, Credits: 4, ClassLevel: 1, MaxCapacity: 30,
		ScheduleSessions: []contracts.ScheduleSession{{DayOfWeek: "monday", SlotNumbers: []int16{slot}, SessionType: "theory"}},
		Prerequisites:    prerequisites,
	}
}

// newServiceOnDB wires the real repositories to a throwaway database with an
// open enrollment period; only student and catalog are fakes.
func newServiceOnDB(t *testing.T, student contracts.StudentResponse, catalog CourseCatalogClient) (*EnrollmentService, *repository.EnrollmentRepository) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	pool := pgtest.New(t, "enrollment", "../sql/migrations")
	periods := sharedRepo.NewSimplePeriodRepository(pool, "enrollment")
	now := time.Now()
	require.NoError(t, periods.UpsertPeriod(context.Background(), sharedRepo.SimplePeriod{
		ID: uuid.New(), Semester: itSemester, PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), IsActive: true,
	}))
	repo := repository.NewEnrollmentRepository(pool)
	students := &mockStudentClient{students: map[uuid.UUID]contracts.StudentResponse{uuid.MustParse(student.ID): student}}
	return NewEnrollmentService(repo, repository.NewPassedPrerequisitesRepository(pool), students, catalog, periods), repo
}

// The pending program used to be deleted before the new selection was even
// looked at, so a resubmission that failed any check left the student with
// no program at all.
func TestCreateEnrollmentProgram_ResubmissionFailsACheck_PendingProgramStays(t *testing.T) {
	studentID := uuid.New()
	student := contracts.StudentResponse{ID: studentID.String(), Department: "Bilgisayar Mühendisliği", ClassLevel: 2, Status: "active"}
	open := semesterCourse("BLM101", student.Department, 1)
	locked := semesterCourse("BLM202", student.Department, 2, contracts.Prerequisite{ID: uuid.New(), CourseCode: "BLM102"})
	svc, repo := newServiceOnDB(t, student, catalogByID{open.ID: open, locked.ID: locked})
	ctx := context.Background()

	first, err := svc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{open.ID}})
	require.NoError(t, err)

	_, err = svc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{locked.ID}})

	require.ErrorIs(t, err, serviceErrors.ErrPrerequisitesNotMet)
	still, err := repo.GetEnrollmentProgramByStudentAndSemester(ctx, studentID, itSemester)
	require.NoError(t, err, "the failed resubmission must not cost the student the program they had")
	assert.Equal(t, first.ID, utils.PgtypeToUUID(still.ID))
}

func TestCreateEnrollmentProgram_ValidResubmission_ReplacesThePendingProgram(t *testing.T) {
	studentID := uuid.New()
	student := contracts.StudentResponse{ID: studentID.String(), Department: "Bilgisayar Mühendisliği", ClassLevel: 2, Status: "active"}
	a := semesterCourse("BLM101", student.Department, 1)
	b := semesterCourse("BLM103", student.Department, 2)
	svc, repo := newServiceOnDB(t, student, catalogByID{a.ID: a, b.ID: b})
	ctx := context.Background()

	first, err := svc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{a.ID}})
	require.NoError(t, err)
	second, err := svc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{a.ID, b.ID}})
	require.NoError(t, err)

	assert.NotEqual(t, first.ID, second.ID)
	current, err := repo.GetEnrollmentProgramByStudentAndSemester(ctx, studentID, itSemester)
	require.NoError(t, err)
	assert.Equal(t, second.ID, utils.PgtypeToUUID(current.ID))
	courses, err := repo.GetCoursesByProgramID(ctx, second.ID)
	require.NoError(t, err)
	assert.Len(t, courses, 2)
}
