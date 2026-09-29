//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
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
