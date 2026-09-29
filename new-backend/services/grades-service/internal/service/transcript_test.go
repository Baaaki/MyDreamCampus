package service

import (
	"context"
	"fmt"
	"testing"

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
