package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment/internal/dto"

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

func (m *mockEnrollmentStore) GetEnrollmentProgramsByStudent(ctx context.Context, studentID uuid.UUID, semester *string, status *string) ([]db.EnrollmentProgram, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []db.EnrollmentProgram
	for _, p := range m.programs {
		if utils.PgtypeToUUID(p.StudentID) == studentID {
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
		// What HTTPStudentClient turns a 404 into.
		return contracts.StudentResponse{}, serviceErrors.ErrStudentNotFound
	}
	return st, nil
}

func (m *mockStudentClient) GetStudentsWithoutAdvisor(ctx context.Context) ([]contracts.StudentResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []contracts.StudentResponse
	for _, st := range m.students {
		if st.AdvisorID == nil {
			res = append(res, st)
		}
	}
	return res, nil
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

	catalog := &mockCourseCatalogClient{coursesByIDs: []contracts.SemesterCourseResponse{{
		ID: courseID, InstructorFullname: "Ahmet Yılmaz",
		ScheduleSessions: []contracts.ScheduleSession{{DayOfWeek: "monday", SlotNumbers: []int16{1}, SessionType: "theory"}},
	}}}
	svc := &EnrollmentService{
		enrollmentRepo:      store,
		studentClient:       stClient,
		courseCatalogClient: catalog,
	}

	resp, err := svc.ApproveEnrollmentProgram(ctx, programID, advisorID, false)
	require.NoError(t, err)
	assert.Equal(t, programID, resp.ID)
	assert.Equal(t, studentID, resp.StudentID)
	assert.Equal(t, "approved", resp.Status)
	assert.Len(t, resp.Courses, 1)
	assert.Equal(t, "CENG101", resp.Courses[0].CourseCode)
	assert.Equal(t, "Ahmet Yılmaz", resp.Courses[0].InstructorName)
	assert.Len(t, resp.Courses[0].ScheduleSessions, 1, "the approved program comes back as my-enrollments draws it")
	assert.NotNil(t, store.lastApproveEvt)
}

// The route lets admins in; without the advisor check giving way to them, a
// student whose advisor was deactivated had a program no one could decide.
func TestApproveEnrollmentProgram_WhoMayDecide(t *testing.T) {
	initTestLogger(t)
	ctx := context.Background()
	advisorID, otherTeacher, adminID := uuid.New(), uuid.New(), uuid.New()
	advStr := advisorID.String()
	withAdvisor, withoutAdvisor := uuid.New(), uuid.New()

	cases := []struct {
		name    string
		student uuid.UUID
		caller  uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"the student's advisor", withAdvisor, advisorID, false, nil},
		{"another teacher", withAdvisor, otherTeacher, false, sharedErrors.ErrForbidden},
		{"an admin, for a student with an advisor", withAdvisor, adminID, true, nil},
		{"an admin, for a student without one", withoutAdvisor, adminID, true, nil},
		{"a teacher, for a student without an advisor", withoutAdvisor, otherTeacher, false, sharedErrors.ErrForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			programID := uuid.New()
			store := newMockEnrollmentStore()
			store.programs[programID] = db.EnrollmentProgram{
				ID: utils.UUIDToPgtype(programID), StudentID: utils.UUIDToPgtype(tc.student), Semester: "2026-2027-Spring",
				Status: db.NullEnrollmentStatusEnum{EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending, Valid: true},
			}
			svc := &EnrollmentService{
				enrollmentRepo: store,
				studentClient: &mockStudentClient{students: map[uuid.UUID]contracts.StudentResponse{
					withAdvisor:    {ID: withAdvisor.String(), AdvisorID: &advStr, Status: "active"},
					withoutAdvisor: {ID: withoutAdvisor.String(), Status: "active"},
				}},
				courseCatalogClient: &mockCourseCatalogClient{},
			}

			_, err := svc.ApproveEnrollmentProgram(ctx, programID, tc.caller, tc.isAdmin)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, store.lastApproveEvt)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.caller.String(), store.lastApproveEvt["data"].(map[string]any)["approved_by"])
		})
	}
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

	_, err := svc.ApproveEnrollmentProgram(ctx, uuid.New(), advisorID, false)
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

	_, err := svc.ApproveEnrollmentProgram(ctx, programID, advisorID, false)
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

	advisorName := "Dr. Mehmet Yilmaz"
	stClient := &mockStudentClient{
		students: map[uuid.UUID]contracts.StudentResponse{
			studentID: {
				ID:          studentID.String(),
				AdvisorID:   &advStr,
				AdvisorName: &advisorName,
				Status:      "active",
			},
		},
	}

	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient:  stClient,
		courseCatalogClient: &mockCourseCatalogClient{coursesByIDs: []contracts.SemesterCourseResponse{
			{ID: courseID, InstructorFullname: "Ayşe Demir"},
		}},
	}

	err := svc.RejectEnrollmentProgram(ctx, programID, advisorID, false, "Eksik kredi")
	require.NoError(t, err)

	// The token carries no names: the rejection used to say "Reddeden: •".
	assert.Equal(t, "Dr. Mehmet Yilmaz", store.lastRejection.AdvisorFullname)
	var snapshot dto.RejectedCoursesData
	require.NoError(t, json.Unmarshal(store.lastRejection.RejectedCourses, &snapshot))
	require.Len(t, snapshot.Courses, 1)
	assert.Equal(t, "Ayşe Demir", snapshot.Courses[0].Instructor)
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

	err := svc.RejectEnrollmentProgram(ctx, uuid.New(), advisorID, false, "Sebep")
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

	err := svc.RejectEnrollmentProgram(ctx, programID, advisorID, false, "Sebep")
	require.ErrorIs(t, err, serviceErrors.ErrCannotModifyApproved)
}

func TestRejectEnrollmentProgram_ByAdmin_SignedAsAdmin(t *testing.T) {
	initTestLogger(t)
	programID, studentID, adminID := uuid.New(), uuid.New(), uuid.New()
	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID: utils.UUIDToPgtype(programID), StudentID: utils.UUIDToPgtype(studentID), Semester: "2026-2027-Spring",
		Status: db.NullEnrollmentStatusEnum{EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending, Valid: true},
	}
	svc := &EnrollmentService{
		enrollmentRepo:      store,
		studentClient:       &mockStudentClient{students: map[uuid.UUID]contracts.StudentResponse{studentID: {ID: studentID.String(), Status: "active"}}},
		courseCatalogClient: &mockCourseCatalogClient{getByIDsErr: errors.New("catalog down")},
	}

	err := svc.RejectEnrollmentProgram(context.Background(), programID, adminID, true, "Danışman atanana kadar")

	require.NoError(t, err, "a catalog outage only leaves the instructors out")
	assert.Equal(t, "Yönetici", store.lastRejection.AdvisorFullname)
	assert.Equal(t, utils.UUIDToPgtype(adminID), store.lastRejection.AdvisorID)
}

// The student's schedule screens list each course with its instructor; the
// catalog answer that carried the sessions carried the name too, and it was
// dropped.
func TestGetMyEnrollments_CatalogAnswers_CourseCarriesInstructorAndSessions(t *testing.T) {
	initTestLogger(t)
	studentID, programID, courseID := uuid.New(), uuid.New(), uuid.New()
	store := newMockEnrollmentStore()
	store.programs[programID] = db.EnrollmentProgram{
		ID: utils.UUIDToPgtype(programID), StudentID: utils.UUIDToPgtype(studentID), Semester: "2026-2027-Fall",
		Status: db.NullEnrollmentStatusEnum{EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumApproved, Valid: true},
	}
	store.coursesByProg[programID] = []db.EnrollmentProgramCourse{
		{CourseID: utils.UUIDToPgtype(courseID), CourseCode: "CENG301", CourseName: "İşletim Sistemleri", Credits: 3},
	}
	catalog := &mockCourseCatalogClient{coursesByIDs: []contracts.SemesterCourseResponse{{
		ID: courseID, InstructorFullname: "Ahmet Yılmaz",
		ScheduleSessions: []contracts.ScheduleSession{{DayOfWeek: "wednesday", SlotNumbers: []int16{1, 2}, SessionType: "theory"}},
	}}}
	svc := &EnrollmentService{enrollmentRepo: store, courseCatalogClient: catalog}

	resp, err := svc.GetMyEnrollments(context.Background(), studentID, nil, nil)
	require.NoError(t, err)
	require.Len(t, resp.Programs, 1)
	require.Len(t, resp.Programs[0].Courses, 1)
	course := resp.Programs[0].Courses[0]
	assert.Equal(t, "Ahmet Yılmaz", course.InstructorName)
	assert.Equal(t, []dto.ScheduleSession{{DayOfWeek: "wednesday", SlotNumbers: []int{1, 2}, SessionType: "theory"}}, course.ScheduleSessions)

	// Without the catalog the program still lists its courses.
	catalog.getByIDsErr = errors.New("catalog down")
	resp, err = svc.GetMyEnrollments(context.Background(), studentID, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Programs[0].Courses[0].InstructorName)
	assertSessionsMarshalEmpty(t, resp.Programs[0].Courses[0].ScheduleSessions)
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

		resp, err := svc.GetPendingProgramsByAdvisor(ctx, advisorID, false)
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
					ID:            s1.String(),
					StudentNumber: "2024001",
					FirstName:     "Ahmet",
					LastName:      "Kaya",
					Department:    "Fizik",
					ClassLevel:    2,
					AdvisorID:     &advStr,
					Status:        "active",
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

		catalog := &mockCourseCatalogClient{
			coursesByIDs: []contracts.SemesterCourseResponse{
				{ID: c1, InstructorFullname: "Prof. Dr. Marie Curie"},
			},
		}
		svc := &EnrollmentService{
			enrollmentRepo:      store,
			studentClient:       stClient,
			courseCatalogClient: catalog,
		}

		resp, err := svc.GetPendingProgramsByAdvisor(ctx, advisorID, false)
		require.NoError(t, err)
		assert.Equal(t, advisorID, resp.AdvisorID)
		require.Len(t, resp.Programs, 1)
		got := resp.Programs[0]
		assert.Equal(t, p1, got.ID)
		assert.Equal(t, s1, got.StudentID)
		assert.Equal(t, "Ahmet Kaya", got.StudentName)
		assert.Equal(t, "2024001", got.StudentNumber)
		assert.Equal(t, "Fizik", got.Department)
		assert.Equal(t, int16(2), got.ClassLevel)
		require.Len(t, got.Courses, 1)
		assert.Equal(t, "PHYS101", got.Courses[0].CourseCode)
		assert.Equal(t, "Prof. Dr. Marie Curie", got.Courses[0].InstructorName)
		assertSessionsMarshalEmpty(t, got.Courses[0].ScheduleSessions)

		// The instructor is a label: a catalog outage must not hide the
		// programs waiting for the advisor's decision.
		catalog.getByIDsErr = errors.New("catalog down")
		resp, err = svc.GetPendingProgramsByAdvisor(ctx, advisorID, false)
		require.NoError(t, err)
		require.Len(t, resp.Programs, 1)
		assert.Equal(t, "2024001", resp.Programs[0].StudentNumber)
		assert.Empty(t, resp.Programs[0].Courses[0].InstructorName)
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
		assertSessionsMarshalEmpty(t, c2.ScheduleSessions)
	})
}

// No advisor's list holds a student without one; the admin's does.
func TestGetPendingProgramsByAdvisor_Admin_ListsStudentsWithoutAdvisor(t *testing.T) {
	initTestLogger(t)
	advisorID, adminID := uuid.New(), uuid.New()
	advStr := advisorID.String()
	advised, orphan := uuid.New(), uuid.New()
	store := newMockEnrollmentStore()
	for _, student := range []uuid.UUID{advised, orphan} {
		programID := uuid.New()
		store.programs[programID] = db.EnrollmentProgram{
			ID: utils.UUIDToPgtype(programID), StudentID: utils.UUIDToPgtype(student), Semester: "2026-2027-Spring",
			Status: db.NullEnrollmentStatusEnum{EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending, Valid: true},
		}
	}
	svc := &EnrollmentService{
		enrollmentRepo: store,
		studentClient: &mockStudentClient{students: map[uuid.UUID]contracts.StudentResponse{
			advised: {ID: advised.String(), AdvisorID: &advStr, Status: "active"},
			orphan:  {ID: orphan.String(), StudentNumber: "2024001", Status: "active"},
		}},
		courseCatalogClient: &mockCourseCatalogClient{},
	}

	resp, err := svc.GetPendingProgramsByAdvisor(context.Background(), adminID, true)

	require.NoError(t, err)
	require.Len(t, resp.Programs, 1)
	assert.Equal(t, orphan, resp.Programs[0].StudentID)
}
