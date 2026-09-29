//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStaffDB(t *testing.T) (*pgxpool.Pool, *StaffRepository) {
	t.Helper()
	pool := pgtest.New(t, "staff", "../sql/migrations")
	return pool, NewStaffRepository(pool)
}

func teacher(email, department string) db.CreateStaffParams {
	return db.CreateStaffParams{Email: email, FirstName: "Ahmet", LastName: "Yılmaz", Role: "teacher", Department: utils.StringToPgText(department)}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

func TestStaffRepository_CreateTeacher_WritesProfileAndEventWithTheNewID(t *testing.T) {
	pool, repo := newStaffDB(t)

	staff, err := repo.CreateStaffWithEvent(context.Background(), teacher("ahmet@uni.edu.tr", "Bilgisayar"), map[string]any{"id": nil})

	require.NoError(t, err)
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM staff.teacher_profiles WHERE staff_id = $1", staff.ID))
	var payload []byte
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT payload FROM staff.outbox_events").Scan(&payload))
	var got map[string]any
	require.NoError(t, json.Unmarshal(payload, &got))
	assert.Equal(t, utils.PgtypeToUUIDString(staff.ID), got["id"], "auth creates the login under this id")
}

// The email check before the insert only sees active members; the unique
// constraint is what stops the rest, and it must come back as a conflict
// with nothing half-written.
func TestStaffRepository_TakenEmail_ConflictWithNothingWritten(t *testing.T) {
	pool, repo := newStaffDB(t)
	ctx := context.Background()
	gone, err := repo.CreateStaffWithEvent(ctx, teacher("ahmet@uni.edu.tr", "Bilgisayar"), map[string]any{})
	require.NoError(t, err)
	require.NoError(t, repo.SoftDeleteStaffWithEvent(ctx, utils.PgtypeToUUID(gone.ID), map[string]any{}))

	active, err := repo.GetStaffByEmail(ctx, "ahmet@uni.edu.tr")
	require.NoError(t, err)
	assert.False(t, active.ID.Valid, "a deactivated member does not show up in the pre-check")
	_, err = repo.CreateStaffWithEvent(ctx, teacher("ahmet@uni.edu.tr", "Fizik"), map[string]any{})

	assert.ErrorIs(t, err, serviceErrors.ErrStaffExistsRepo, "a deactivated member's email stays taken")
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM staff.teacher_profiles"))
	assert.Equal(t, 2, count(t, pool, "SELECT count(*) FROM staff.outbox_events"), "created and deactivated, nothing for the failed insert")
}

func TestStaffRepository_SoftDelete_HidesTheMemberEverywhere(t *testing.T) {
	_, repo := newStaffDB(t)
	ctx := context.Background()
	kept, err := repo.CreateStaffWithEvent(ctx, teacher("ahmet@uni.edu.tr", "Bilgisayar"), map[string]any{})
	require.NoError(t, err)
	gone, err := repo.CreateStaffWithEvent(ctx, teacher("mehmet@uni.edu.tr", "Bilgisayar"), map[string]any{})
	require.NoError(t, err)

	require.NoError(t, repo.SoftDeleteStaffWithEvent(ctx, utils.PgtypeToUUID(gone.ID), map[string]any{}))

	list, total, err := repo.ListStaff(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, kept.ID, list[0].ID)
	instructors, err := repo.GetInstructorsByDepartment(ctx, "Bilgisayar")
	require.NoError(t, err)
	assert.Len(t, instructors, 1, "a deactivated teacher is not offered for new courses")
	_, err = repo.UpdateStaffWithEvent(ctx, utils.PgtypeToUUID(gone.ID), db.UpdateStaffParams{ID: gone.ID, Phone: utils.StringToPgText("555")}, map[string]any{})
	assert.ErrorIs(t, err, serviceErrors.ErrStaffNotFoundRepo)
}

func TestAdminStaffRepository_EmailClash_OnCreateAndUpdate(t *testing.T) {
	pool := pgtest.New(t, "staff", "../sql/migrations")
	repo := NewAdminStaffRepository(pool)
	ctx := context.Background()
	member := func(email string) db.CreateAdminStaffParams {
		return db.CreateAdminStaffParams{Email: email, FirstName: "Ayşe", LastName: "Kaya", Faculty: "Fen Fakültesi", Position: "Sekreter", Responsibilities: []byte(`[]`)}
	}
	first, err := repo.Create(ctx, member("ayse@uni.edu.tr"))
	require.NoError(t, err)
	second, err := repo.Create(ctx, member("fatma@uni.edu.tr"))
	require.NoError(t, err)

	_, err = repo.Create(ctx, member("ayse@uni.edu.tr"))
	assert.ErrorIs(t, err, sharedErrors.ErrAlreadyExistsRepo)

	_, err = repo.Update(ctx, db.UpdateAdminStaffParams{
		ID: second.ID, Email: first.Email, FirstName: "Fatma", LastName: "Demir", Faculty: "Fen Fakültesi", Position: "Sekreter", Responsibilities: []byte(`[]`),
	})
	assert.ErrorIs(t, err, sharedErrors.ErrAlreadyExistsRepo)

	_, err = repo.Update(ctx, db.UpdateAdminStaffParams{
		ID: utils.UUIDToPgtype(uuid.New()), Email: "yeni@uni.edu.tr", FirstName: "X", LastName: "Y", Faculty: "Fen Fakültesi", Position: "Sekreter", Responsibilities: []byte(`[]`),
	})
	assert.ErrorIs(t, err, sharedErrors.ErrNotFoundRepo)
}
