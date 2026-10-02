package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errPeerDown is what shared/client returns when the target cannot answer.
var errPeerDown = fmt.Errorf("internal client: GET /internal/x: %w: %w", client.ErrUnavailable, errors.New("dial tcp: connection refused"))

func requireStatus(t *testing.T, err error, status int) {
	t.Helper()
	appErr, ok := sharedErrors.As(err)
	require.True(t, ok, "expected an AppError, got %v", err)
	assert.Equal(t, status, appErr.HTTPStatus)
}

func TestUpstreamError_PeerThatCannotAnswerIsA503(t *testing.T) {
	requireStatus(t, upstreamError(errPeerDown), http.StatusServiceUnavailable)
	requireStatus(t, upstreamError(&client.StatusError{StatusCode: http.StatusBadGateway}), http.StatusServiceUnavailable)
	requireStatus(t, upstreamError(errors.New("decode response: unexpected EOF")), http.StatusInternalServerError)
}

func TestGetAvailableCourses_StudentServiceDown_Returns503NotStudentNotFound(t *testing.T) {
	initTestLogger(t)
	svc := &EnrollmentService{studentClient: &mockStudentClient{getErr: errPeerDown}}

	_, err := svc.GetAvailableCourses(context.Background(), uuid.New(), "2026-2027-Fall")

	assert.NotErrorIs(t, err, serviceErrors.ErrStudentNotFound)
	requireStatus(t, err, http.StatusServiceUnavailable)
}

func TestGetAvailableCourses_CatalogDown_Returns503(t *testing.T) {
	initTestLogger(t)
	studentID := uuid.New()
	svc := &EnrollmentService{
		studentClient:       &mockStudentClient{students: map[uuid.UUID]contracts.StudentResponse{studentID: {ID: studentID.String(), Status: "active"}}},
		courseCatalogClient: &mockCourseCatalogClient{getAvailableErr: errPeerDown},
	}

	_, err := svc.GetAvailableCourses(context.Background(), studentID, "2026-2027-Fall")

	requireStatus(t, err, http.StatusServiceUnavailable)
}

func TestApproveEnrollmentProgram_StudentServiceDown_Returns503(t *testing.T) {
	initTestLogger(t)
	programID := uuid.New()
	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID:        utils.UUIDToPgtype(programID),
		StudentID: utils.UUIDToPgtype(uuid.New()),
		Status:    db.NullEnrollmentStatusEnum{EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending, Valid: true},
	}
	svc := &EnrollmentService{enrollmentRepo: store, studentClient: &mockStudentClient{getErr: errPeerDown}}

	_, err := svc.ApproveEnrollmentProgram(context.Background(), programID, uuid.New())

	requireStatus(t, err, http.StatusServiceUnavailable)
}
