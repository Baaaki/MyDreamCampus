//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStudentDB(t *testing.T) (*pgxpool.Pool, *StudentRepository) {
	t.Helper()
	pool := pgtest.New(t, "student", "../sql/migrations")
	return pool, NewStudentRepository(pool)
}

func student(number, email string) db.CreateStudentParams {
	return db.CreateStudentParams{
		StudentNumber: number, FirstName: "Zeynep", LastName: "Şahin", Email: email,
		Faculty: "Mühendislik Fakültesi", Department: "Bilgisayar Mühendisliği", EnrollmentYear: 2026, ClassLevel: 1,
	}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

// The service checks number and email before the insert, so the unique
// indexes only fire when two creates race. The repository tells the two
// apart by index name, which only the real schema can confirm.
func TestStudentRepository_TakenNumberOrEmail_MappedToItsConflict(t *testing.T) {
	pool, repo := newStudentDB(t)
	ctx := context.Background()
	_, err := repo.CreateStudentWithEvent(ctx, student("2026001", "zeynep@uni.edu.tr"), map[string]any{})
	require.NoError(t, err)

	_, err = repo.CreateStudentWithEvent(ctx, student("2026001", "baska@uni.edu.tr"), map[string]any{})
	assert.ErrorIs(t, err, serviceErrors.ErrStudentNumberExistsRepo)

	_, err = repo.CreateStudentWithEvent(ctx, student("2026002", "zeynep@uni.edu.tr"), map[string]any{})
	assert.ErrorIs(t, err, serviceErrors.ErrStudentEmailExistsRepo)

	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM student.outbox_events"), "no event for a refused create")
}

func TestStudentRepository_SoftDeleted_FreesNumberAndEmail(t *testing.T) {
	_, repo := newStudentDB(t)
	ctx := context.Background()
	gone, err := repo.CreateStudentWithEvent(ctx, student("2026001", "zeynep@uni.edu.tr"), map[string]any{})
	require.NoError(t, err)
	require.NoError(t, repo.SoftDeleteStudentWithEvent(ctx, utils.PgtypeToUUID(gone.ID), map[string]any{}))

	back, err := repo.CreateStudentWithEvent(ctx, student("2026001", "zeynep@uni.edu.tr"), map[string]any{})

	require.NoError(t, err, "the unique indexes cover active students only")
	found, err := repo.GetStudentByNumber(ctx, "2026001")
	require.NoError(t, err)
	assert.Equal(t, back.ID, found.ID)
}

func TestStudentRepository_ClassLevelOutOfRange_Rejected(t *testing.T) {
	_, repo := newStudentDB(t)
	params := student("2026001", "zeynep@uni.edu.tr")
	params.ClassLevel = 7

	_, err := repo.CreateStudentWithEvent(context.Background(), params, map[string]any{})

	assert.Error(t, err)
}

func TestStudentRepository_Advisor_AssignSkipsInactiveAndUnassignMarksTheEvent(t *testing.T) {
	pool, repo := newStudentDB(t)
	ctx := context.Background()
	active, err := repo.CreateStudentWithEvent(ctx, student("2026001", "a@uni.edu.tr"), map[string]any{})
	require.NoError(t, err)
	inactive, err := repo.CreateStudentWithEvent(ctx, student("2026002", "b@uni.edu.tr"), map[string]any{})
	require.NoError(t, err)
	require.NoError(t, repo.SoftDeleteStudentWithEvent(ctx, utils.PgtypeToUUID(inactive.ID), map[string]any{}))
	advisor := uuid.New()

	require.NoError(t, repo.BulkAssignAdvisor(ctx, []uuid.UUID{utils.PgtypeToUUID(active.ID), utils.PgtypeToUUID(inactive.ID)}, advisor, "Ahmet Yılmaz", []map[string]any{{}}))

	advised, err := repo.ListStudentsByAdvisor(ctx, advisor)
	require.NoError(t, err)
	require.Len(t, advised, 1)
	assert.Equal(t, active.ID, advised[0].ID)
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM student.students WHERE advisor_id IS NOT NULL"), "a deleted student is not assigned")
	orphans, total, err := repo.ListOrphanedStudents(ctx, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, orphans)
	assert.Zero(t, total)

	eventID := uuid.NewString()
	require.NoError(t, repo.UnassignAdvisorByStaffIDWithEventMarking(ctx, advisor, eventID, "staff.deactivated"))

	orphans, total, err = repo.ListOrphanedStudents(ctx, 10, 0)
	require.NoError(t, err)
	require.Len(t, orphans, 1)
	assert.Equal(t, int64(1), total)
	assert.False(t, orphans[0].AdvisorName.Valid, "the snapshot of the old advisor's name goes too")
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM student.processed_events WHERE event_id = $1", eventID))
}
