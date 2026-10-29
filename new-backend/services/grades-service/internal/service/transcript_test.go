package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	gradeErrors "github.com/baaaki/mydreamcampus/grades/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStudentClient answers like HTTPStudentClient: an unknown id is
// ErrStudentNotFound, and err stands in for an unreachable student service.
type fakeStudentClient struct {
	students map[uuid.UUID]contracts.StudentResponse
	err      error
	calls    *int
}

func (f fakeStudentClient) GetStudentByID(_ context.Context, id uuid.UUID) (contracts.StudentResponse, error) {
	*f.calls++
	if f.err != nil {
		return contracts.StudentResponse{}, f.err
	}
	s, ok := f.students[id]
	if !ok {
		return contracts.StudentResponse{}, gradeErrors.ErrStudentNotFound
	}
	return s, nil
}

func TestGetTranscript_WhoMaySeeWhom(t *testing.T) {
	advisor, otherTeacher := uuid.New(), uuid.New()
	advisee, unadvised, unknown := uuid.New(), uuid.New(), uuid.New()
	advisorID := advisor.String()
	students := map[uuid.UUID]contracts.StudentResponse{
		advisee:   {ID: advisee.String(), AdvisorID: &advisorID},
		unadvised: {ID: unadvised.String()},
	}
	outage := fmt.Errorf("student service: %w", client.ErrUnavailable)

	cases := []struct {
		name        string
		requester   uuid.UUID
		role        string
		student     uuid.UUID
		clientErr   error
		want        error
		asksStudent bool
	}{
		{name: "admin sees anyone", requester: uuid.New(), role: "admin", student: unadvised},
		{name: "student sees their own", requester: advisee, role: "student", student: advisee},
		{name: "student cannot see another", requester: advisee, role: "student", student: unadvised, want: gradeErrors.ErrTranscriptForbidden},
		{name: "advisor sees an advisee", requester: advisor, role: "teacher", student: advisee, asksStudent: true},
		{name: "teacher cannot see someone else's advisee", requester: otherTeacher, role: "teacher", student: advisee, want: gradeErrors.ErrNotStudentAdvisor, asksStudent: true},
		{name: "teacher cannot see a student with no advisor", requester: advisor, role: "teacher", student: unadvised, want: gradeErrors.ErrNotStudentAdvisor, asksStudent: true},
		{name: "a student the student service does not know", requester: advisor, role: "teacher", student: unknown, want: gradeErrors.ErrStudentNotFound, asksStudent: true},
		{name: "student service down: refused, not waved through", requester: advisor, role: "teacher", student: advisee, clientErr: outage, want: gradeErrors.ErrStudentInfoUnavailable, asksStudent: true},
		{name: "any other role sees nothing", requester: uuid.New(), role: "guest", student: advisee, want: gradeErrors.ErrTranscriptForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, _ := newGradesWorld(t)
			for _, id := range []uuid.UUID{advisee, unadvised, unknown} {
				w.students[id] = db.GradesStudentsView{ID: id, StudentNumber: id.String()[:8], IsActive: pgtype.Bool{Bool: true, Valid: true}}
			}
			calls := 0
			svc := &StudentGradesService{
				cacheRepo: cacheView{w}, registrationRepo: regView{w: w}, scoreRepo: scoreView{w: w}, completedRepo: completedView{w: w},
				studentClient: fakeStudentClient{students: students, err: c.clientErr, calls: &calls},
			}

			transcript, err := svc.GetTranscript(context.Background(), c.requester, c.role, c.student)

			if c.want != nil {
				assert.ErrorIs(t, err, c.want)
				assert.Nil(t, transcript)
			} else {
				require.NoError(t, err)
				require.NotNil(t, transcript)
			}
			assert.Equal(t, c.asksStudent, calls > 0, "only a teacher's access needs the student service")
		})
	}
}

func TestCompareSemesters_MixedFormats_SortsChronologically(t *testing.T) {
	want := []string{
		"2024_fall",   // 2024-2025 Güz
		"2025_spring", // 2024-2025 Bahar
		"2025-2026-Fall",
		"2025-2026-Spring",
		"2025-2026-Summer",
		"2026-2027-Fall",
		"bilinmeyen",
		"zzz",
	}
	for range 20 {
		got := slices.Clone(want)
		rand.Shuffle(len(got), func(i, j int) { got[i], got[j] = got[j], got[i] })
		slices.SortFunc(got, compareSemesters)
		require.Equal(t, want, got)
	}
}

func TestGetTranscript_Semesters_ReturnedInChronologicalOrder(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 90, 90, false)
	svc := &StudentGradesService{cacheRepo: cacheView{w}, registrationRepo: regView{w: w}, scoreRepo: scoreView{w: w}, completedRepo: completedView{w: w}}
	for _, sem := range []string{"2026-2027-Fall", "2025-2026-Spring", "2025-2026-Fall", "2026-2027-Spring"} {
		w.completed = append(w.completed, db.GradesStudentCompletedCourse{
			StudentID: reg.StudentID, Semester: sem, CourseCode: "C-" + sem, Credits: 3, GradePoint: db.GradesGradePointEnum300,
		})
	}

	// The semesters are grouped through a map, so one lucky run proves nothing.
	for range 20 {
		tr, err := svc.GetTranscript(context.Background(), uuid.New(), "admin", reg.StudentID)
		require.NoError(t, err)

		var got []string
		for _, sem := range tr.Semesters {
			got = append(got, sem.SemesterDisplay)
		}
		require.Equal(t, []string{"2025-2026 Güz", "2025-2026 Bahar", "2026-2027 Güz", "2026-2027 Bahar"}, got)
	}
}

// generated_at was never set and every transcript claimed to be from year 1.
func TestGetTranscript_StampsWhenItWasGenerated(t *testing.T) {
	w, courseID := newGradesWorld(t)
	reg := w.enroll(courseID, "1", 90, 90, false)
	svc := &StudentGradesService{cacheRepo: cacheView{w}, registrationRepo: regView{w: w}, scoreRepo: scoreView{w: w}, completedRepo: completedView{w: w}}

	tr, err := svc.GetTranscript(context.Background(), uuid.New(), "admin", reg.StudentID)

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), tr.GeneratedAt, time.Minute)
}
