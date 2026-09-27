package service

import (
	"context"
	"errors"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	"github.com/baaaki/mydreamcampus/student/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCRUDStudentRepo struct {
	StudentRepositoryInterface
	studentMap    map[uuid.UUID]db.Student
	byNumberMap   map[string]db.Student
	byEmailMap    map[string]db.Student
	createErr     error
	getErr        error
	updateErr     error
	deleteErr     error
	bulkAssignErr error
}

func newMockCRUDStudentRepo() *mockCRUDStudentRepo {
	return &mockCRUDStudentRepo{
		studentMap:  make(map[uuid.UUID]db.Student),
		byNumberMap: make(map[string]db.Student),
		byEmailMap:  make(map[string]db.Student),
	}
}

func (m *mockCRUDStudentRepo) GetStudentByNumber(ctx context.Context, studentNumber string) (db.Student, error) {
	if m.getErr != nil {
		return db.Student{}, m.getErr
	}
	if s, ok := m.byNumberMap[studentNumber]; ok {
		return s, nil
	}
	return db.Student{}, nil
}

func (m *mockCRUDStudentRepo) GetStudentByEmail(ctx context.Context, email string) (db.Student, error) {
	if m.getErr != nil {
		return db.Student{}, m.getErr
	}
	if s, ok := m.byEmailMap[email]; ok {
		return s, nil
	}
	return db.Student{}, nil
}

func (m *mockCRUDStudentRepo) GetStudentByID(ctx context.Context, id uuid.UUID) (db.Student, error) {
	if m.getErr != nil {
		return db.Student{}, m.getErr
	}
	if s, ok := m.studentMap[id]; ok {
		return s, nil
	}
	return db.Student{}, serviceErrors.ErrStudentNotFoundRepo
}

func (m *mockCRUDStudentRepo) CreateStudentWithEvent(ctx context.Context, params db.CreateStudentParams, eventPayload map[string]any) (db.Student, error) {
	if m.createErr != nil {
		return db.Student{}, m.createErr
	}
	id := uuid.New()
	st := db.Student{
		ID:             utils.UUIDToPgtype(id),
		StudentNumber:  params.StudentNumber,
		FirstName:      params.FirstName,
		LastName:       params.LastName,
		Email:          params.Email,
		Faculty:        params.Faculty,
		Department:     params.Department,
		EnrollmentYear: params.EnrollmentYear,
		ClassLevel:     params.ClassLevel,
		Status:         utils.StringToPgText("active"),
	}
	m.studentMap[id] = st
	m.byNumberMap[params.StudentNumber] = st
	m.byEmailMap[params.Email] = st
	return st, nil
}

func (m *mockCRUDStudentRepo) UpdateStudentWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStudentParams, eventPayload map[string]any) (db.Student, error) {
	if m.updateErr != nil {
		return db.Student{}, m.updateErr
	}
	st, ok := m.studentMap[id]
	if !ok {
		return db.Student{}, serviceErrors.ErrStudentNotFoundRepo
	}
	if params.FirstName.Valid {
		st.FirstName = params.FirstName.String
	}
	if params.LastName.Valid {
		st.LastName = params.LastName.String
	}
	m.studentMap[id] = st
	return st, nil
}

func (m *mockCRUDStudentRepo) SoftDeleteStudentWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]any) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.studentMap[id]; !ok {
		return serviceErrors.ErrStudentNotFoundRepo
	}
	delete(m.studentMap, id)
	return nil
}

func (m *mockCRUDStudentRepo) BulkAssignAdvisor(ctx context.Context, studentIDs []uuid.UUID, advisorID uuid.UUID, advisorName string, eventPayloads []map[string]any) error {
	if m.bulkAssignErr != nil {
		return m.bulkAssignErr
	}
	return nil
}

type mockStaffSvc struct {
	StaffServiceInterface
	err error
}

func (m *mockStaffSvc) GetAdvisorInfo(ctx context.Context, advisorID uuid.UUID) (*AdvisorDetails, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &AdvisorDetails{
		ID:   advisorID.String(),
		Name: "Prof. Ahmet Yilmaz",
	}, nil
}

func TestStudentService_CreateStudent_Success(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	staff := &mockStaffSvc{}
	svc := NewStudentService(repo, staff)

	req := dto.CreateStudentRequest{
		StudentNumber:  "2024001",
		FirstName:      "Ali",
		LastName:       "Can",
		Email:          "ali.can@uni.edu.tr",
		Faculty:        "Engineering",
		Department:     "CENG",
		EnrollmentYear: 2024,
		ClassLevel:     1,
	}

	resp, err := svc.CreateStudent(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "2024001", resp.StudentNumber)
	assert.Equal(t, "Ali", resp.FirstName)
	assert.Equal(t, "Can", resp.LastName)
	assert.Equal(t, "ali.can@uni.edu.tr", resp.Email)
}

func TestStudentService_CreateStudent_WithAdvisor(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	advisorID := uuid.New()
	staff := &mockStaffSvc{}
	svc := NewStudentService(repo, staff)

	req := dto.CreateStudentRequest{
		StudentNumber:  "2024002",
		FirstName:      "Ayse",
		LastName:       "Kaya",
		Email:          "ayse.kaya@uni.edu.tr",
		Faculty:        "Engineering",
		Department:     "CENG",
		EnrollmentYear: 2024,
		ClassLevel:     1,
		AdvisorID:      &advisorID,
	}

	resp, err := svc.CreateStudent(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "2024002", resp.StudentNumber)
}

func TestStudentService_CreateStudent_DuplicateNumber(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	repo.byNumberMap["2024001"] = db.Student{StudentNumber: "2024001"}
	svc := NewStudentService(repo, &mockStaffSvc{})

	req := dto.CreateStudentRequest{
		StudentNumber:  "2024001",
		FirstName:      "Veli",
		LastName:       "Demir",
		Email:          "veli@uni.edu.tr",
		Faculty:        "Engineering",
		Department:     "CENG",
		EnrollmentYear: 2024,
		ClassLevel:     1,
	}

	_, err := svc.CreateStudent(context.Background(), req)
	assert.ErrorIs(t, err, serviceErrors.ErrStudentNumberExists)
}

func TestStudentService_CreateStudent_DuplicateEmail(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	repo.byEmailMap["existing@uni.edu.tr"] = db.Student{Email: "existing@uni.edu.tr"}
	svc := NewStudentService(repo, &mockStaffSvc{})

	req := dto.CreateStudentRequest{
		StudentNumber:  "2024003",
		FirstName:      "Veli",
		LastName:       "Demir",
		Email:          "existing@uni.edu.tr",
		Faculty:        "Engineering",
		Department:     "CENG",
		EnrollmentYear: 2024,
		ClassLevel:     1,
	}

	_, err := svc.CreateStudent(context.Background(), req)
	assert.ErrorIs(t, err, serviceErrors.ErrStudentEmailExists)
}

func TestStudentService_CreateStudent_AdvisorNotFound(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	advisorID := uuid.New()
	staff := &mockStaffSvc{err: errors.New("advisor not found in staff service")}
	svc := NewStudentService(repo, staff)

	req := dto.CreateStudentRequest{
		StudentNumber:  "2024004",
		FirstName:      "Fatma",
		LastName:       "Ozturk",
		Email:          "fatma@uni.edu.tr",
		Faculty:        "Engineering",
		Department:     "CENG",
		EnrollmentYear: 2024,
		ClassLevel:     1,
		AdvisorID:      &advisorID,
	}

	_, err := svc.CreateStudent(context.Background(), req)
	assert.ErrorIs(t, err, serviceErrors.ErrAdvisorNotFound)
}

func TestStudentService_GetStudentByID(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	svc := NewStudentService(repo, &mockStaffSvc{})

	studentID := uuid.New()
	repo.studentMap[studentID] = db.Student{
		ID:            utils.UUIDToPgtype(studentID),
		StudentNumber: "2024005",
		FirstName:     "Hasan",
		LastName:      "Celik",
		Email:         "hasan@uni.edu.tr",
		Status:        utils.StringToPgText("active"),
	}

	t.Run("valid ID returns student", func(t *testing.T) {
		resp, err := svc.GetStudentByID(context.Background(), studentID.String())
		require.NoError(t, err)
		assert.Equal(t, "2024005", resp.StudentNumber)
		assert.Equal(t, "Hasan", resp.FirstName)
	})

	t.Run("invalid UUID returns error", func(t *testing.T) {
		_, err := svc.GetStudentByID(context.Background(), "invalid-uuid")
		assert.ErrorIs(t, err, sharedErrors.ErrInvalidID)
	})

	t.Run("not found ID returns error", func(t *testing.T) {
		_, err := svc.GetStudentByID(context.Background(), uuid.New().String())
		assert.ErrorIs(t, err, serviceErrors.ErrStudentNotFound)
	})
}

func TestStudentService_DeleteStudent(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	svc := NewStudentService(repo, &mockStaffSvc{})

	studentID := uuid.New()
	repo.studentMap[studentID] = db.Student{
		ID:            utils.UUIDToPgtype(studentID),
		StudentNumber: "2024006",
		Email:         "test.delete@uni.edu.tr",
		Status:        utils.StringToPgText("active"),
	}

	t.Run("successful deletion", func(t *testing.T) {
		err := svc.DeleteStudent(context.Background(), studentID.String())
		require.NoError(t, err)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		err := svc.DeleteStudent(context.Background(), "not-a-uuid")
		assert.ErrorIs(t, err, sharedErrors.ErrInvalidID)
	})

	t.Run("not found student", func(t *testing.T) {
		err := svc.DeleteStudent(context.Background(), uuid.New().String())
		assert.ErrorIs(t, err, serviceErrors.ErrStudentNotFound)
	})
}

func TestStudentService_BulkAssignAdvisor(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	repo := newMockCRUDStudentRepo()
	advisorID := uuid.New()
	s1 := uuid.New()
	s2 := uuid.New()

	t.Run("success", func(t *testing.T) {
		staff := &mockStaffSvc{}
		svc := NewStudentService(repo, staff)

		resp, err := svc.BulkAssignAdvisor(context.Background(), dto.BulkAdvisorAssignRequest{
			StudentIDs: []uuid.UUID{s1, s2},
			AdvisorID:  advisorID,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.UpdatedCount)
		assert.Equal(t, advisorID.String(), resp.Advisor.AdvisorID)
		assert.Equal(t, "Prof. Ahmet Yilmaz", resp.Advisor.AdvisorName)
	})

	t.Run("advisor validation failure", func(t *testing.T) {
		staff := &mockStaffSvc{err: errors.New("advisor inactive")}
		svc := NewStudentService(repo, staff)

		_, err := svc.BulkAssignAdvisor(context.Background(), dto.BulkAdvisorAssignRequest{
			StudentIDs: []uuid.UUID{s1},
			AdvisorID:  advisorID,
		})
		assert.ErrorIs(t, err, serviceErrors.ErrAdvisorNotFound)
	})
}
