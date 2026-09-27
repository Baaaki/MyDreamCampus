package service

import (
	"context"
	"sync"
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

// mockEnrollmentStore implements EnrollmentStore with thread-safe controls.
type mockEnrollmentStore struct {
	EnrollmentStore
	mu              sync.RWMutex
	programs        map[uuid.UUID]db.EnrollmentProgram
	coursesByProg   map[uuid.UUID][]db.EnrollmentProgramCourse
	enrollmentCount map[uuid.UUID]int
	lastRejection   db.CreateRejectionLogParams
	lastApproveEvt  map[string]any
	lastRejectEvt   map[string]any
	getProgramErr   error
	approveErr      error
	rejectErr       error
}

func newMockEnrollmentStore() *mockEnrollmentStore {
	return &mockEnrollmentStore{
		programs:        make(map[uuid.UUID]db.EnrollmentProgram),
		coursesByProg:   make(map[uuid.UUID][]db.EnrollmentProgramCourse),
		enrollmentCount: make(map[uuid.UUID]int),
	}
}

func (m *mockEnrollmentStore) GetEnrollmentProgramByID(ctx context.Context, id uuid.UUID) (db.EnrollmentProgram, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getProgramErr != nil {
		return db.EnrollmentProgram{}, m.getProgramErr
	}
	p, ok := m.programs[id]
	if !ok {
		return db.EnrollmentProgram{}, sharedErrors.ErrNotFound
	}
	return p, nil
}

func (m *mockEnrollmentStore) GetCoursesByProgramID(ctx context.Context, programID uuid.UUID) ([]db.EnrollmentProgramCourse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.coursesByProg[programID], nil
}

func (m *mockEnrollmentStore) ApproveProgramWithEvent(ctx context.Context, programID uuid.UUID, eventPayload map[string]any) (db.EnrollmentProgram, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.approveErr != nil {
		return db.EnrollmentProgram{}, m.approveErr
	}
	p, ok := m.programs[programID]
	if !ok {
		return db.EnrollmentProgram{}, sharedErrors.ErrNotFound
	}
	p.Status = db.NullEnrollmentStatusEnum{
		EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumApproved,
		Valid:                true,
	}
	m.programs[programID] = p
	m.lastApproveEvt = eventPayload
	return p, nil
}

func (m *mockEnrollmentStore) RejectProgramWithEventAndLog(ctx context.Context, programID uuid.UUID, rejectionLogParams db.CreateRejectionLogParams, eventPayload map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rejectErr != nil {
		return m.rejectErr
	}
	_, ok := m.programs[programID]
	if !ok {
		return sharedErrors.ErrNotFound
	}
	delete(m.programs, programID)
	m.lastRejection = rejectionLogParams
	m.lastRejectEvt = eventPayload
	return nil
}

func (m *mockEnrollmentStore) GetPendingProgramsByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]db.EnrollmentProgram, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []db.EnrollmentProgram
	idSet := make(map[uuid.UUID]bool)
	for _, id := range studentIDs {
		idSet[id] = true
	}
	for _, p := range m.programs {
		sid := utils.PgtypeToUUID(p.StudentID)
		if idSet[sid] && p.Status.EnrollmentStatusEnum == db.EnrollmentEnrollmentStatusEnumPending {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockEnrollmentStore) CountEnrollmentForCourse(ctx context.Context, courseID uuid.UUID) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enrollmentCount[courseID], nil
}

// mockStudentClient
type mockStudentClient struct {
	students map[uuid.UUID]contracts.StudentResponse
	getErr   error
	listErr  error
}

func (m *mockStudentClient) GetStudentByID(ctx context.Context, id uuid.UUID) (contracts.StudentResponse, error) {
	if m.getErr != nil {
		return contracts.StudentResponse{}, m.getErr
	}
	st, ok := m.students[id]
	if !ok {
		return contracts.StudentResponse{}, sharedErrors.ErrNotFound
	}
	return st, nil
}

func (m *mockStudentClient) GetStudentsByAdvisorID(ctx context.Context, advisorID uuid.UUID) ([]contracts.StudentResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	advStr := advisorID.String()
	var res []contracts.StudentResponse
	for _, st := range m.students {
		if st.AdvisorID != nil && *st.AdvisorID == advStr {
			res = append(res, st)
		}
	}
	return res, nil
}

// mockCourseCatalogClient
type mockCourseCatalogClient struct {
	availableCourses []contracts.SemesterCourseListItem
	getAvailableErr  error
	coursesByIDs     []contracts.SemesterCourseResponse
	getByIDsErr      error
}

func (m *mockCourseCatalogClient) GetAvailableCourses(ctx context.Context, department string, classLevel int16, semester string) ([]contracts.SemesterCourseListItem, error) {
	if m.getAvailableErr != nil {
		return nil, m.getAvailableErr
	}
	return m.availableCourses, nil
}

func (m *mockCourseCatalogClient) GetCoursesByIDs(ctx context.Context, semester string, ids []uuid.UUID) ([]contracts.SemesterCourseResponse, error) {
	if m.getByIDsErr != nil {
		return nil, m.getByIDsErr
	}
	return m.coursesByIDs, nil
}

// Helper to set up logger
func initTestLogger(t *testing.T) {
	t.Helper()
	_ = logger.Init("test")
}

func TestApproveEnrollmentProgram_Success(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	studentID := uuid.New()
	programID := uuid.New()
	courseID := uuid.New()
	advStr := advisorID.String()

	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID:        utils.UUIDToPgtype(programID),
		StudentID: utils.UUIDToPgtype(studentID),
		Semester:  "2026-2027-Fall",
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending,
			Valid:                true,
		},
	}
	store.coursesByProg[programID] = []db.EnrollmentProgramCourse{
		{
			CourseID:   utils.UUIDToPgtype(courseID),
			CourseCode: "CENG101",
			CourseName: "Intro to Programming",
			Credits:    4,
		},
	}

	stClient := &mockStudentClient{
		students: map[uuid.UUID]contracts.StudentResponse{
			studentID: {
				ID:        studentID.String(),
				AdvisorID: &advStr,
				Status:    "active",
			},
		},
	}

	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  stClient,
	}

	resp, err := svc.ApproveEnrollmentProgram(ctx, programID, advisorID)
	require.NoError(t, err)
	assert.Equal(t, programID, resp.ID)
	assert.Equal(t, studentID, resp.StudentID)
	assert.Equal(t, "approved", resp.Status)
	assert.Len(t, resp.Courses, 1)
	assert.Equal(t, "CENG101", resp.Courses[0].CourseCode)
	assert.NotNil(t, store.lastApproveEvt)
}

func TestApproveEnrollmentProgram_ProgramNotFound(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	store := newMockEnrollmentStore()
	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  &mockStudentClient{},
	}

	_, err := svc.ApproveEnrollmentProgram(ctx, uuid.New(), advisorID)
	require.ErrorIs(t, err, serviceErrors.ErrProgramNotFound)
}

func TestApproveEnrollmentProgram_AlreadyApproved(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	programID := uuid.New()
	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID: utils.UUIDToPgtype(programID),
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumApproved,
			Valid:                true,
		},
	}

	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  &mockStudentClient{},
	}

	_, err := svc.ApproveEnrollmentProgram(ctx, programID, advisorID)
	require.ErrorIs(t, err, serviceErrors.ErrCannotModifyApproved)
}

func TestRejectEnrollmentProgram_Success(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	studentID := uuid.New()
	programID := uuid.New()
	courseID := uuid.New()
	advStr := advisorID.String()

	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID:        utils.UUIDToPgtype(programID),
		StudentID: utils.UUIDToPgtype(studentID),
		Semester:  "2026-2027-Fall",
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending,
			Valid:                true,
		},
	}
	store.coursesByProg[programID] = []db.EnrollmentProgramCourse{
		{
			CourseID:   utils.UUIDToPgtype(courseID),
			CourseCode: "MATH101",
			CourseName: "Calculus I",
			Credits:    3,
		},
	}

	stClient := &mockStudentClient{
		students: map[uuid.UUID]contracts.StudentResponse{
			studentID: {
				ID:        studentID.String(),
				AdvisorID: &advStr,
				Status:    "active",
			},
		},
	}

	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  stClient,
	}

	err := svc.RejectEnrollmentProgram(ctx, programID, advisorID, "Dr. Mehmet Yilmaz", "Eksik kredi")
	require.NoError(t, err)

	assert.Equal(t, "Dr. Mehmet Yilmaz", store.lastRejection.AdvisorFullname)
	assert.Equal(t, "Eksik kredi", store.lastRejection.RejectionReason)
	assert.NotNil(t, store.lastRejectEvt)
	assert.Equal(t, "Eksik kredi", store.lastRejectEvt["rejection_reason"])
}

func TestRejectEnrollmentProgram_ProgramNotFound(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	store := newMockEnrollmentStore()
	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  &mockStudentClient{},
	}

	err := svc.RejectEnrollmentProgram(ctx, uuid.New(), advisorID, "Dr. Ali", "Sebep")
	require.ErrorIs(t, err, serviceErrors.ErrProgramNotFound)
}

func TestRejectEnrollmentProgram_AlreadyApproved(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	programID := uuid.New()
	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID: utils.UUIDToPgtype(programID),
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumApproved,
			Valid:                true,
		},
	}

	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  &mockStudentClient{},
	}

	err := svc.RejectEnrollmentProgram(ctx, programID, advisorID, "Dr. Ali", "Sebep")
	require.ErrorIs(t, err, serviceErrors.ErrCannotModifyApproved)
}

func TestGetPendingProgramsByAdvisor(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	advisorID := uuid.New()
	advStr := advisorID.String()

	t.Run("returns empty when advisor has no students", func(t *testing.T) {
		svc := &EnrollmentService{
			enrollmentRepo: newMockEnrollmentStore(),
			studentClient:  &mockStudentClient{},
		}

		resp, err := svc.GetPendingProgramsByAdvisor(ctx, advisorID)
		require.NoError(t, err)
		assert.Equal(t, advisorID, resp.AdvisorID)
		assert.Empty(t, resp.Programs)
	})

	t.Run("returns pending programs with mapped student details and courses", func(t *testing.T) {
		s1 := uuid.New()
		s2 := uuid.New()
		p1 := uuid.New()
		c1 := uuid.New()

		store := newMockEnrollmentStore()
		store.programs[p1] = db.EnrollmentProgram{
			ID:        utils.UUIDToPgtype(p1),
			StudentID: utils.UUIDToPgtype(s1),
			Semester:  "2026-2027-Fall",
			Status: db.NullEnrollmentStatusEnum{
				EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending,
				Valid:                true,
			},
		}
		store.coursesByProg[p1] = []db.EnrollmentProgramCourse{
			{
				CourseID:   utils.UUIDToPgtype(c1),
				CourseCode: "PHYS101",
				CourseName: "Physics I",
				Credits:    4,
			},
		}

		stClient := &mockStudentClient{
			students: map[uuid.UUID]contracts.StudentResponse{
				s1: {
					ID:        s1.String(),
					FirstName: "Ahmet",
					LastName:  "Kaya",
					AdvisorID: &advStr,
					Status:    "active",
				},
				s2: {
					ID:        s2.String(),
					FirstName: "Ayse",
					LastName:  "Demir",
					AdvisorID: &advStr,
					Status:    "active",
				},
			},
		}

		svc := &EnrollmentService{
			enrollmentRepo: store,
			studentClient:  stClient,
		}

		resp, err := svc.GetPendingProgramsByAdvisor(ctx, advisorID)
		require.NoError(t, err)
		assert.Equal(t, advisorID, resp.AdvisorID)
		require.Len(t, resp.Programs, 1)
		assert.Equal(t, p1, resp.Programs[0].ID)
		assert.Equal(t, s1, resp.Programs[0].StudentID)
		assert.Equal(t, "Ahmet Kaya", resp.Programs[0].StudentName)
		require.Len(t, resp.Programs[0].Courses, 1)
		assert.Equal(t, "PHYS101", resp.Programs[0].Courses[0].CourseCode)
	})
}

func TestGetAvailableCourses(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()

	studentID := uuid.New()
	course1ID := uuid.New()
	course2ID := uuid.New()

	t.Run("returns ErrStudentNotFound when student does not exist", func(t *testing.T) {
		svc := &EnrollmentService{
			studentClient: &mockStudentClient{},
		}
		_, err := svc.GetAvailableCourses(ctx, studentID, "2026-2027-Fall")
		require.ErrorIs(t, err, serviceErrors.ErrStudentNotFound)
	})

	t.Run("returns ErrStudentDeactivated when student is not active", func(t *testing.T) {
		svc := &EnrollmentService{
			studentClient: &mockStudentClient{
				students: map[uuid.UUID]contracts.StudentResponse{
					studentID: {
						ID:     studentID.String(),
						Status: "suspended",
					},
				},
			},
		}
		_, err := svc.GetAvailableCourses(ctx, studentID, "2026-2027-Fall")
		require.ErrorIs(t, err, serviceErrors.ErrStudentDeactivated)
	})

	t.Run("calculates available seats correctly and maps sessions", func(t *testing.T) {
		store := newMockEnrollmentStore()
		store.enrollmentCount[course1ID] = 15 // 50 - 15 = 35 available
		store.enrollmentCount[course2ID] = 30 // 30 - 30 = 0 available

		stClient := &mockStudentClient{
			students: map[uuid.UUID]contracts.StudentResponse{
				studentID: {
					ID:         studentID.String(),
					Department: "Computer Engineering",
					ClassLevel: 2,
					Status:     "active",
				},
			},
		}

		catalogClient := &mockCourseCatalogClient{
			availableCourses: []contracts.SemesterCourseListItem{
				{
					ID:                 course1ID,
					CourseCode:         "CENG201",
					CourseName:         "Data Structures",
					Credits:            4,
					MaxCapacity:        50,
					InstructorFullname: "Prof. Dr. Ada Lovelace",
					ScheduleSessions: []contracts.ScheduleSession{
						{
							DayOfWeek:   "monday",
							SlotNumbers: []int16{1, 2},
							SessionType: "theory",
						},
					},
				},
				{
					ID:                 course2ID,
					CourseCode:         "CENG203",
					CourseName:         "Discrete Math",
					Credits:            3,
					MaxCapacity:        30,
					InstructorFullname: "Dr. Alan Turing",
				},
			},
		}

		svc := &EnrollmentService{
			enrollmentRepo:      store,
			studentClient:       stClient,
			courseCatalogClient: catalogClient,
		}

		resp, err := svc.GetAvailableCourses(ctx, studentID, "2026-2027-Fall")
		require.NoError(t, err)
		assert.Equal(t, studentID, resp.StudentID)
		assert.Equal(t, "Computer Engineering", resp.Department)
		assert.Equal(t, int16(2), resp.ClassLevel)
		require.Len(t, resp.AvailableCourses, 2)

		c1 := resp.AvailableCourses[0]
		assert.Equal(t, "CENG201", c1.CourseCode)
		assert.Equal(t, int16(50), c1.MaxCapacity)
		assert.Equal(t, int16(15), c1.CurrentEnrollment)
		assert.Equal(t, int16(35), c1.AvailableSeats)
		require.Len(t, c1.ScheduleSessions, 1)
		assert.Equal(t, "monday", c1.ScheduleSessions[0].DayOfWeek)
		assert.Equal(t, []int{1, 2}, c1.ScheduleSessions[0].SlotNumbers)

		c2 := resp.AvailableCourses[1]
		assert.Equal(t, "CENG203", c2.CourseCode)
		assert.Equal(t, int16(30), c2.MaxCapacity)
		assert.Equal(t, int16(30), c2.CurrentEnrollment)
		assert.Equal(t, int16(0), c2.AvailableSeats)
	})
}
