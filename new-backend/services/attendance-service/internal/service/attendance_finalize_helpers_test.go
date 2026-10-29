package service

import (
	"testing"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveFailedType(t *testing.T) {
	cases := []struct {
		name         string
		theoryFailed bool
		labFailed    bool
		want         string
		whyItMatters string
	}{
		{"only theory", true, false, "theory",
			"if grades-service sees 'both' it will mark the lab as a fail too, awarding the student a course-level FF — wrong"},
		{"only lab", false, true, "lab",
			"same drift in the opposite direction — student loses the theory pass they earned"},
		{"both", true, true, "both", ""},
		{"both pass", false, false, "both",
			"this branch is unreachable in production (callers only invoke for failures), but the helper must not silently return empty string — leaving 'both' as the safe default keeps downstream consumers from receiving an unknown value"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveFailedType(tc.theoryFailed, tc.labFailed)
			assert.Equal(t, tc.want, got, tc.whyItMatters)
		})
	}
}

func TestFailingStudents_EachTypeOnItsOwn(t *testing.T) {
	studentA := uuid.New()
	studentB := uuid.New()

	// 14 theory and 14 lab sessions: 10 and 11 are the minimums.
	theory := []db.GetCourseAttendanceByStudentAndTypeRow{
		attendanceRow(t, studentA, "20210001", "Ahmet", "Yilmaz", "a@x.x", 14, 7),
		attendanceRow(t, studentB, "20210002", "Mehmet", "Demir", "b@x.x", 14, 12),
	}
	lab := []db.GetCourseAttendanceByStudentAndTypeRow{
		attendanceRow(t, studentA, "20210001", "Ahmet", "Yilmaz", "a@x.x", 14, 13),
		attendanceRow(t, studentB, "20210002", "Mehmet", "Demir", "b@x.x", 14, 5),
	}

	out := failingStudents(theory, lab)
	require.Len(t, out, 2)

	a := out[studentA]
	require.NotNil(t, a)
	assert.Equal(t, TypeAttendance{Held: 14, Present: 7, MinRequired: 10, Failed: true}, a.Theory)
	assert.Equal(t, TypeAttendance{Held: 14, Present: 13, MinRequired: 11}, a.Lab, "the passed type keeps its real standing")

	b := out[studentB]
	require.NotNil(t, b)
	assert.False(t, b.Theory.Failed)
	assert.True(t, b.Lab.Failed)
	assert.Equal(t, 9, b.Lab.response().AbsentCount)
}

func TestFailingStudents_BothTypes_OneRecord(t *testing.T) {
	// A student failing BOTH theory and lab ends up with one entry that has
	// both flags set. Two entries would make the publisher emit two events
	// for the same student and grades-service would double-count them.
	id := uuid.New()
	theory := []db.GetCourseAttendanceByStudentAndTypeRow{attendanceRow(t, id, "20210001", "Ahmet", "Yilmaz", "a@x.x", 14, 3)}
	lab := []db.GetCourseAttendanceByStudentAndTypeRow{attendanceRow(t, id, "20210001", "Ahmet", "Yilmaz", "a@x.x", 14, 1)}

	out := failingStudents(theory, lab)
	require.Len(t, out, 1, "the same student must collapse into a single failure record")
	assert.True(t, out[id].Theory.Failed)
	assert.True(t, out[id].Lab.Failed)
	assert.Equal(t, "Ahmet Yilmaz", out[id].StudentName)
}

// A student approved late is held to the 4 sessions run since: attending 3
// meets their minimum of ceil(4*10/14)=3. The same 3 out of the course's 8
// sessions fails a student enrolled from the start (minimum 6).
func TestFailingStudents_LateEnrollment_HeldToTheirOwnSessions(t *testing.T) {
	late, early := uuid.New(), uuid.New()
	theory := []db.GetCourseAttendanceByStudentAndTypeRow{
		attendanceRow(t, late, "20210001", "Geç", "Kayıt", "g@x.x", 4, 3),
		attendanceRow(t, early, "20210002", "Erken", "Kayıt", "e@x.x", 8, 3),
	}

	out := failingStudents(theory, nil)

	assert.NotContains(t, out, late, "3 of the 4 sessions held since approval meets ceil(4*10/14)=3")
	require.Contains(t, out, early)
	assert.Equal(t, 6, out[early].Theory.MinRequired)
}

func TestFailingStudents_NothingHeldForThem_CannotFail(t *testing.T) {
	id := uuid.New()
	out := failingStudents([]db.GetCourseAttendanceByStudentAndTypeRow{attendanceRow(t, id, "1", "A", "B", "a@x.x", 0, 0)}, nil)
	assert.Empty(t, out)
}

func TestFailingStudents_EmptyInputs(t *testing.T) {
	out := failingStudents(nil, nil)
	assert.Empty(t, out, "no failures means an empty (not nil) map for safe iteration")
}

func TestFailingStudents_StudentNamePopulation(t *testing.T) {
	// Defensive — exercises the fmt.Sprintf path so a regression that drops
	// the space between first/last name shows up in CI rather than in a UI bug.
	id := uuid.New()
	theory := []db.GetCourseAttendanceByStudentAndTypeRow{attendanceRow(t, id, "20210001", "Ali Baba", "Yilmaz Demir", "ali@x.x", 10, 0)}
	out := failingStudents(theory, nil)
	assert.Equal(t, "Ali Baba Yilmaz Demir", out[id].StudentName)
	assert.Equal(t, "ali@x.x", out[id].StudentEmail)
}

// attendanceRow hides the pgtype noise of one student's row.
func attendanceRow(
	t *testing.T,
	studentID uuid.UUID,
	number, first, last, email string,
	held, present int,
) db.GetCourseAttendanceByStudentAndTypeRow {
	t.Helper()
	return db.GetCourseAttendanceByStudentAndTypeRow{
		StudentID:     utils.UUIDToPgUUID(studentID),
		StudentNumber: number,
		FirstName:     pgtype.Text{String: first, Valid: true},
		LastName:      pgtype.Text{String: last, Valid: true},
		Email:         pgtype.Text{String: email, Valid: true},
		HeldCount:     int64(held),
		PresentCount:  int64(present),
	}
}
