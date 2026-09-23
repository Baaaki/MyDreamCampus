package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pendingProgramStore struct {
	EnrollmentStore
	program    db.EnrollmentProgram
	approveErr error
}

func (f *pendingProgramStore) GetEnrollmentProgramByID(context.Context, uuid.UUID) (db.EnrollmentProgram, error) {
	return f.program, nil
}

func (f *pendingProgramStore) GetCoursesByProgramID(context.Context, uuid.UUID) ([]db.EnrollmentProgramCourse, error) {
	return nil, nil
}

func (f *pendingProgramStore) ApproveProgramWithEvent(context.Context, uuid.UUID, map[string]any) (db.EnrollmentProgram, error) {
	if f.approveErr != nil {
		return db.EnrollmentProgram{}, f.approveErr
	}
	return f.program, nil
}

type advisedStudent struct {
	StudentClient
	advisorID string
}

func (f advisedStudent) GetStudentByID(context.Context, uuid.UUID) (contracts.StudentResponse, error) {
	return contracts.StudentResponse{AdvisorID: &f.advisorID}, nil
}

func newAdvisorTestService(t *testing.T, advisorID uuid.UUID) (*EnrollmentService, *pendingProgramStore) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	store := &pendingProgramStore{program: db.EnrollmentProgram{
		ID:        utils.UUIDToPgtype(uuid.New()),
		StudentID: utils.UUIDToPgtype(uuid.New()),
		Semester:  "2026-2027-Fall",
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending,
			Valid:                true,
		},
	}}
	return &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  advisedStudent{advisorID: advisorID.String()},
	}, store
}

func httpStatusOf(t *testing.T, err error) int {
	t.Helper()
	if appErr, ok := sharedErrors.As(err); ok {
		return appErr.HTTPStatus
	}
	appErr, ok := serviceErrors.MapSentinel(err)
	require.True(t, ok, "unmapped error %v", err)
	return appErr.HTTPStatus
}

func TestApproveEnrollmentProgram_WrongAdvisor_Returns403(t *testing.T) {
	svc, _ := newAdvisorTestService(t, uuid.New())

	_, err := svc.ApproveEnrollmentProgram(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, httpStatusOf(t, err))
}

func TestRejectEnrollmentProgram_WrongAdvisor_Returns403(t *testing.T) {
	svc, _ := newAdvisorTestService(t, uuid.New())

	err := svc.RejectEnrollmentProgram(context.Background(), uuid.New(), uuid.New(), "Dr. Baska", "Yanlis ders secimi yapilmis")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, httpStatusOf(t, err))
}

func TestApproveEnrollmentProgram_NotPending_Returns409(t *testing.T) {
	advisorID := uuid.New()
	svc, store := newAdvisorTestService(t, advisorID)
	store.approveErr = serviceErrors.ErrProgramNotPending

	_, err := svc.ApproveEnrollmentProgram(context.Background(), uuid.New(), advisorID)

	require.ErrorIs(t, err, serviceErrors.ErrProgramNotPending)
	assert.Equal(t, http.StatusConflict, httpStatusOf(t, err))
}
