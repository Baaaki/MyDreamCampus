package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/clock/clocktest"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePeriodStore struct {
	periods []sharedRepo.SimplePeriod
	err     error
}

func (f *fakePeriodStore) GetActivePeriodBySemester(ctx context.Context, semester string) (*sharedRepo.SimplePeriod, error) {
	return nil, errors.New("not used")
}

func (f *fakePeriodStore) GetAllPeriods(ctx context.Context) ([]sharedRepo.SimplePeriod, error) {
	return f.periods, f.err
}

type programsByStudentStore struct {
	EnrollmentStore
	programs []db.EnrollmentProgram
}

func (s *programsByStudentStore) GetEnrollmentProgramsByStudent(ctx context.Context, studentID uuid.UUID, semester *string, status *string) ([]db.EnrollmentProgram, error) {
	return s.programs, nil
}

func activePeriod(semester string, start, end time.Time) sharedRepo.SimplePeriod {
	return sharedRepo.SimplePeriod{Semester: semester, PeriodStart: start, PeriodEnd: end, IsActive: true}
}

// openView keeps the assertions on the fields that decide the screen.
type openView struct {
	Semester       string
	EnrollmentOpen bool
	HasProgram     bool
}

func view(t *testing.T, periods []sharedRepo.SimplePeriod, hasProgram map[string]bool, at time.Time) []openView {
	t.Helper()
	got := openSemesters(periods, hasProgram, at)
	out := make([]openView, len(got))
	for i, s := range got {
		out[i] = openView{s.Semester, s.EnrollmentOpen, s.HasProgram}
	}
	return out
}

var testNow = time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

func TestOpenSemesters_WindowOpen_ListsItWithProgramFlag(t *testing.T) {
	periods := []sharedRepo.SimplePeriod{activePeriod("2025-2026-Spring", testNow.AddDate(0, 0, -3), testNow.AddDate(0, 0, 4))}

	got := view(t, periods, map[string]bool{"2025-2026-Spring": true}, testNow)

	assert.Equal(t, []openView{{"2025-2026-Spring", true, true}}, got)
}

func TestOpenSemesters_WindowBoundaries_AreInclusive(t *testing.T) {
	start := activePeriod("A", testNow, testNow.AddDate(0, 0, 7))
	end := activePeriod("B", testNow.AddDate(0, 0, -7), testNow)

	got := view(t, []sharedRepo.SimplePeriod{start, end}, nil, testNow)

	assert.Equal(t, []openView{{"B", true, false}, {"A", true, false}}, got)
}

func TestOpenSemesters_CurrentSemesterWithProgram_IsListedClosed(t *testing.T) {
	periods := []sharedRepo.SimplePeriod{
		activePeriod("2024-2025-Spring", testNow.AddDate(-1, 0, 0), testNow.AddDate(-1, 0, 7)),
		activePeriod("2025-2026-Fall", testNow.AddDate(0, -5, 0), testNow.AddDate(0, -5, 7)),
		activePeriod("2025-2026-Spring", testNow.AddDate(0, 0, -2), testNow.AddDate(0, 0, 5)),
	}
	hasProgram := map[string]bool{"2024-2025-Spring": true, "2025-2026-Fall": true}

	got := view(t, periods, hasProgram, testNow)

	// An older semester the student also has a program in is not current.
	assert.Equal(t, []openView{
		{"2025-2026-Fall", false, true},
		{"2025-2026-Spring", true, false},
	}, got)
}

func TestOpenSemesters_CurrentSemesterWithoutProgram_IsLeftOut(t *testing.T) {
	periods := []sharedRepo.SimplePeriod{activePeriod("2025-2026-Fall", testNow.AddDate(0, -5, 0), testNow.AddDate(0, -5, 7))}

	got := view(t, periods, map[string]bool{"2024-2025-Fall": true}, testNow)

	assert.Empty(t, got)
}

func TestOpenSemesters_FutureOrInactiveWindow_IsLeftOut(t *testing.T) {
	future := activePeriod("future", testNow.AddDate(0, 0, 1), testNow.AddDate(0, 0, 8))
	inactive := activePeriod("inactive", testNow.AddDate(0, 0, -1), testNow.AddDate(0, 0, 6))
	inactive.IsActive = false

	got := view(t, []sharedRepo.SimplePeriod{future, inactive}, map[string]bool{"future": true, "inactive": true}, testNow)

	assert.Empty(t, got)
}

func TestOpenSemesters_NoPeriods_MarshalsEmptyArray(t *testing.T) {
	raw, err := json.Marshal(openSemesters(nil, nil, testNow))

	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(raw))
}

func TestGetOpenSemesters_UsesServiceClockAndStudentPrograms(t *testing.T) {
	initTestLogger(t)
	clocktest.Freeze(t, testNow)
	periods := &fakePeriodStore{periods: []sharedRepo.SimplePeriod{
		activePeriod("2025-2026-Fall", testNow.AddDate(0, -5, 0), testNow.AddDate(0, -5, 7)),
		activePeriod("2025-2026-Spring", testNow.AddDate(0, 0, -1), testNow.AddDate(0, 0, 6)),
	}}
	store := &programsByStudentStore{programs: []db.EnrollmentProgram{{Semester: "2025-2026-Fall"}}}
	svc := NewEnrollmentService(store, nil, nil, nil, periods)

	got, err := svc.GetOpenSemesters(context.Background(), uuid.New())

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "2025-2026-Fall", got[0].Semester)
	assert.False(t, got[0].EnrollmentOpen)
	assert.True(t, got[0].HasProgram)
	assert.Equal(t, "2025-2026-Spring", got[1].Semester)
	assert.True(t, got[1].EnrollmentOpen)
	assert.False(t, got[1].HasProgram)
}

func TestGetOpenSemesters_PeriodStoreFails_ReturnsInternal(t *testing.T) {
	initTestLogger(t)
	svc := NewEnrollmentService(&programsByStudentStore{}, nil, nil, nil, &fakePeriodStore{err: errors.New("db down")})

	_, err := svc.GetOpenSemesters(context.Background(), uuid.New())

	appErr, ok := sharedErrors.As(err)
	require.True(t, ok)
	assert.Equal(t, sharedErrors.ErrInternal.Code, appErr.Code)
}
