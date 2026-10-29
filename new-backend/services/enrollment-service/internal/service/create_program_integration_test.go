//go:build integration

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	"github.com/jackc/pgx/v5/pgxpool"
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
	svc, repo, _ := newServiceOnDBWithActivity(t, student, catalog)
	return svc, repo
}

func newServiceOnDBWithActivity(t *testing.T, student contracts.StudentResponse, catalog CourseCatalogClient, activity ...SemesterActivityClient) (*EnrollmentService, *repository.EnrollmentRepository, *pgxpool.Pool) {
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
	return NewEnrollmentService(repo, repository.NewPassedPrerequisitesRepository(pool), students, catalog, periods, activity), repo, pool
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

// downCatalog is a catalog that cannot answer at all.
type downCatalog struct{}

func (downCatalog) GetAvailableCourses(context.Context, string, int16, string) ([]contracts.SemesterCourseListItem, error) {
	return nil, errPeerDown
}

func (downCatalog) GetCoursesByIDs(context.Context, string, []uuid.UUID) ([]contracts.SemesterCourseResponse, error) {
	return nil, errPeerDown
}

func TestCreateEnrollmentProgram_UnknownCourse_CourseNotFound(t *testing.T) {
	studentID := uuid.New()
	student := contracts.StudentResponse{ID: studentID.String(), Department: "Bilgisayar Mühendisliği", ClassLevel: 2, Status: "active"}
	svc, _ := newServiceOnDB(t, student, catalogByID{})

	_, err := svc.CreateEnrollmentProgram(context.Background(), dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{uuid.New()}})

	assert.ErrorIs(t, err, serviceErrors.ErrCourseNotFound, "a 404 from catalog is the course's answer, not a server error")
}

func TestCreateEnrollmentProgram_CatalogDown_Returns503(t *testing.T) {
	studentID := uuid.New()
	student := contracts.StudentResponse{ID: studentID.String(), Department: "Bilgisayar Mühendisliği", ClassLevel: 2, Status: "active"}
	svc, _ := newServiceOnDB(t, student, downCatalog{})

	_, err := svc.CreateEnrollmentProgram(context.Background(), dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{uuid.New()}})

	requireStatus(t, err, http.StatusServiceUnavailable)
}

// activityAnswer is attendance or grades answering the semester-activity read.
type activityAnswer struct {
	has bool
	err error
}

func (a activityAnswer) HasSemesterActivity(context.Context, uuid.UUID, string) (bool, error) {
	return a.has, a.err
}

// submitAndApprove gives the student an approved program of the courses.
func submitAndApprove(t *testing.T, svc *EnrollmentService, repo *repository.EnrollmentRepository, studentID uuid.UUID, courses ...uuid.UUID) uuid.UUID {
	t.Helper()
	program, err := svc.CreateEnrollmentProgram(context.Background(), dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: courses})
	require.NoError(t, err)
	_, err = repo.ApproveProgramWithEvent(context.Background(), program.ID, map[string]any{})
	require.NoError(t, err)
	return program.ID
}

// The window is open, so an approved program may still change: the new
// selection goes back to the advisor, and the old one's cancelled event
// tells attendance and grades which courses to drop.
func TestCreateEnrollmentProgram_ApprovedProgramChanged_PendingAgainWithCancelEvent(t *testing.T) {
	studentID := uuid.New()
	student := contracts.StudentResponse{ID: studentID.String(), Department: "Bilgisayar Mühendisliği", ClassLevel: 2, Status: "active"}
	kept := semesterCourse("BLM101", student.Department, 1)
	kept.MaxCapacity = 1 // the student's own seat must not count against keeping it
	dropped := semesterCourse("BLM103", student.Department, 2)
	added := semesterCourse("BLM105", student.Department, 3)
	svc, repo, pool := newServiceOnDBWithActivity(t, student,
		catalogByID{kept.ID: kept, dropped.ID: dropped, added.ID: added}, activityAnswer{}, activityAnswer{})
	ctx := context.Background()
	approved := submitAndApprove(t, svc, repo, studentID, kept.ID, dropped.ID)

	changed, err := svc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{kept.ID, added.ID}})

	require.NoError(t, err)
	assert.Equal(t, "pending", changed.Status, "the change goes back to the advisor")
	current, err := repo.GetEnrollmentProgramByStudentAndSemester(ctx, studentID, itSemester)
	require.NoError(t, err)
	assert.Equal(t, changed.ID, utils.PgtypeToUUID(current.ID))

	var payload []byte
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT payload FROM enrollment.outbox_events WHERE event_type = 'enrollment.program.cancelled'").Scan(&payload))
	var cancelled struct {
		ProgramID uuid.UUID   `json:"program_id"`
		CourseIDs []uuid.UUID `json:"course_ids"`
	}
	require.NoError(t, json.Unmarshal(payload, &cancelled))
	assert.Equal(t, approved, cancelled.ProgramID)
	assert.ElementsMatch(t, []uuid.UUID{kept.ID, dropped.ID}, cancelled.CourseIDs)
}

func TestCreateEnrollmentProgram_ApprovedProgramChange_Refused(t *testing.T) {
	for name, tc := range map[string]struct {
		grades     activityAnswer
		wantErr    error
		wantStatus int
	}{
		"a score exists": {grades: activityAnswer{has: true}, wantErr: serviceErrors.ErrApprovedProgramInUse},
		"grades is down": {grades: activityAnswer{err: errPeerDown}, wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			studentID := uuid.New()
			student := contracts.StudentResponse{ID: studentID.String(), Department: "Bilgisayar Mühendisliği", ClassLevel: 2, Status: "active"}
			a := semesterCourse("BLM101", student.Department, 1)
			b := semesterCourse("BLM103", student.Department, 2)
			svc, repo, _ := newServiceOnDBWithActivity(t, student, catalogByID{a.ID: a, b.ID: b}, activityAnswer{}, tc.grades)
			ctx := context.Background()
			approved := submitAndApprove(t, svc, repo, studentID, a.ID)

			_, err := svc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{StudentID: studentID, Semester: itSemester, CourseIDs: []uuid.UUID{b.ID}})

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				requireStatus(t, err, tc.wantStatus)
			}
			current, err := repo.GetEnrollmentProgramByStudentAndSemester(ctx, studentID, itSemester)
			require.NoError(t, err)
			assert.Equal(t, approved, utils.PgtypeToUUID(current.ID), "the approved program stands")
			assert.Equal(t, "approved", string(current.Status.EnrollmentStatusEnum))
		})
	}
}
