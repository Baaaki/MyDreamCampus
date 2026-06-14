package service

import (
	"testing"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveFailedType(t *testing.T) {
	cases := []struct {
		name           string
		theoryFailed   bool
		labFailed      bool
		want           string
		whyItMatters   string
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

func TestMergeFailureRows_SeparateStudents(t *testing.T) {
	studentA := uuid.New()
	studentB := uuid.New()

	theory := []db.GetFailingStudentsByCourseByTypeRow{
		failingRow(t, studentA, "20210001", "Ahmet", "Yilmaz", "a@x.x", 3, 7),
	}
	lab := []db.GetFailingStudentsByCourseByTypeRow{
		failingRow(t, studentB, "20210002", "Mehmet", "Demir", "b@x.x", 2, 5),
	}

	out := mergeFailureRows(theory, lab)
	require.Len(t, out, 2)

	a := out[studentA]
	require.NotNil(t, a)
	assert.True(t, a.TheoryFailed)
	assert.False(t, a.LabFailed)
	assert.Equal(t, 3, a.TheoryPresent)
	assert.Equal(t, 7, a.TheoryAbsent)
	assert.Equal(t, 0, a.LabPresent, "untouched lab counts must remain zero, not inherit theory's")

	b := out[studentB]
	require.NotNil(t, b)
	assert.False(t, b.TheoryFailed)
	assert.True(t, b.LabFailed)
	assert.Equal(t, 2, b.LabPresent)
}

func TestMergeFailureRows_StudentInBothLists(t *testing.T) {
	// The merge contract: a student appearing in BOTH theory and lab failure
	// rows ends up with one entry that has both flags set. Without this
	// behaviour the publisher would emit two events for the same student and
	// grades-service would double-count them.
	id := uuid.New()
	theory := []db.GetFailingStudentsByCourseByTypeRow{
		failingRow(t, id, "20210001", "Ahmet", "Yilmaz", "a@x.x", 3, 7),
	}
	lab := []db.GetFailingStudentsByCourseByTypeRow{
		failingRow(t, id, "20210001", "Ahmet", "Yilmaz", "a@x.x", 1, 5),
	}

	out := mergeFailureRows(theory, lab)
	require.Len(t, out, 1, "the same student must collapse into a single failure record")

	info := out[id]
	require.NotNil(t, info)
	assert.True(t, info.TheoryFailed)
	assert.True(t, info.LabFailed)
	assert.Equal(t, 3, info.TheoryPresent)
	assert.Equal(t, 7, info.TheoryAbsent)
	assert.Equal(t, 1, info.LabPresent)
	assert.Equal(t, 5, info.LabAbsent)
	assert.Equal(t, "Ahmet Yilmaz", info.StudentName,
		"name must be set from the theory pass — the lab pass must NOT overwrite it (it would still be the same name in practice, but checking that the helper doesn't reset other fields)")
}

func TestMergeFailureRows_EmptyInputs(t *testing.T) {
	out := mergeFailureRows(nil, nil)
	assert.Empty(t, out, "no failures means an empty (not nil) map for safe iteration")
}

func TestMergeFailureRows_StudentNamePopulation(t *testing.T) {
	// Defensive — exercises the fmt.Sprintf path so a regression that drops
	// the space between first/last name shows up in CI rather than in a UI bug.
	id := uuid.New()
	theory := []db.GetFailingStudentsByCourseByTypeRow{
		failingRow(t, id, "20210001", "Ali Baba", "Yilmaz Demir", "ali@x.x", 0, 10),
	}
	out := mergeFailureRows(theory, nil)
	assert.Equal(t, "Ali Baba Yilmaz Demir", out[id].StudentName)
	assert.Equal(t, "ali@x.x", out[id].StudentEmail)
}

// failingRow is a tiny constructor that hides the pgtype noise — every field
// is required by the merge code path, so the helpers are easier to read with
// it factored out.
func failingRow(
	t *testing.T,
	studentID uuid.UUID,
	number, first, last, email string,
	present, absent int,
) db.GetFailingStudentsByCourseByTypeRow {
	t.Helper()
	return db.GetFailingStudentsByCourseByTypeRow{
		StudentID:     utils.UUIDToPgUUID(studentID),
		StudentNumber: number,
		FirstName:     pgtype.Text{String: first, Valid: true},
		LastName:      pgtype.Text{String: last, Valid: true},
		Email:         pgtype.Text{String: email, Valid: true},
		PresentCount:  int64(present),
		AbsentCount:   int64(absent),
	}
}
