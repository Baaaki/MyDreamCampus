//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	catalogErrors "github.com/baaaki/mydreamcampus/catalog/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fall = "2026-2027-Fall"

func newCatalogDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := pgtest.New(t, "course_catalog", "../sql/migrations")
	for _, code := range []string{"BLM101", "BLM102", "BLM103"} {
		_, err := pool.Exec(context.Background(), `INSERT INTO course_catalog.course_catalog
			(course_code, name, faculty, department, class_level, credits)
			VALUES ($1, $1, 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 1, 4)`, code)
		require.NoError(t, err)
	}
	return pool
}

func courseParams(semester, code string, instructor uuid.UUID) db.CreateSemesterCourseParams {
	return db.CreateSemesterCourseParams{
		Semester: semester, CourseCode: code, Department: "Bilgisayar Mühendisliği", Credits: 4, ClassLevel: 1,
		InstructorID: utils.UUIDToPgtype(instructor), InstructorFullname: "Ahmet Yılmaz", ClassroomLocation: "A-101",
		MaxCapacity: 60, AssessmentSchema: []byte(`[{"slug":"final","weight":100}]`), Prerequisites: []byte(`[]`),
	}
}

func openCourse(t *testing.T, repo *SemesterRepository, semester, code string, instructor uuid.UUID) pgtype.UUID {
	t.Helper()
	c, err := repo.CreateSemesterCourse(context.Background(), courseParams(semester, code, instructor))
	require.NoError(t, err)
	return c.ID
}

func mondaySlot(course pgtype.UUID, slot int16) db.CreateScheduleSessionParams {
	return db.CreateScheduleSessionParams{SemesterCourseID: course, DayOfWeek: db.DayOfWeekEnum("monday"), SlotNumber: slot, SessionType: db.ScheduleSessionTypeEnum("theory")}
}

func TestSemesterRepository_CourseOpenedTwice_Conflict(t *testing.T) {
	repo := NewSemesterRepository(newCatalogDB(t))
	openCourse(t, repo, fall, "BLM101", uuid.New())

	_, err := repo.CreateSemesterCourse(context.Background(), courseParams(fall, "BLM101", uuid.New()))

	assert.ErrorIs(t, err, catalogErrors.ErrCourseAlreadyOpenedRepo)
}

// The service checks the instructor's slots before it writes; the trigger
// is what stops a second course that takes the slot in between. The
// schedule is written in the course's transaction, so the loser must come
// back as the conflict, not as a failed query.
func TestScheduleRepository_InstructorDoubleBooked_RefusedAsAConflict(t *testing.T) {
	pool := newCatalogDB(t)
	ctx := context.Background()
	semesters, schedule := NewSemesterRepository(pool), NewScheduleRepository(pool)
	instructor := uuid.New()
	first := openCourse(t, semesters, fall, "BLM101", instructor)
	second := openCourse(t, semesters, fall, "BLM102", instructor)
	require.NoError(t, schedule.BulkCreateScheduleSessions(ctx, []db.CreateScheduleSessionParams{mondaySlot(first, 1), mondaySlot(first, 2)}))

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	err = schedule.WithTx(tx).BulkCreateScheduleSessions(ctx, []db.CreateScheduleSessionParams{mondaySlot(second, 3), mondaySlot(second, 2)})

	assert.ErrorIs(t, err, catalogErrors.ErrInstructorScheduleConflict)
	require.NoError(t, tx.Rollback(ctx))

	nextTerm := openCourse(t, semesters, "2026-2027-Spring", "BLM102", instructor)
	assert.NoError(t, schedule.BulkCreateScheduleSessions(ctx, []db.CreateScheduleSessionParams{mondaySlot(nextTerm, 2)}), "the same slot in another semester")
	colleague := openCourse(t, semesters, fall, "BLM103", uuid.New())
	assert.NoError(t, schedule.BulkCreateScheduleSessions(ctx, []db.CreateScheduleSessionParams{mondaySlot(colleague, 2)}), "another instructor in the same slot")
}

func TestSemesterStatusRepository_OnlyForwardAndOnlyOneActive(t *testing.T) {
	pool := pgtest.New(t, "course_catalog", "../sql/migrations")
	repo := NewSemesterStatusRepository(pool, nil)
	ctx := context.Background()
	deadline := time.Now().Add(90 * 24 * time.Hour)
	fallSem, err := repo.CreateSemester(ctx, fall, deadline)
	require.NoError(t, err)
	springSem, err := repo.CreateSemester(ctx, "2026-2027-Spring", deadline)
	require.NoError(t, err)

	_, err = repo.CreateSemester(ctx, fall, deadline)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr, "the handler answers 409 on this")
	assert.Equal(t, "23505", pgErr.Code)

	active, err := repo.ActivateSemester(ctx, utils.PgtypeToUUID(fallSem.ID))
	require.NoError(t, err)
	assert.True(t, active.ActivatedAt.Valid, "the trigger stamps the activation")

	_, err = pool.Exec(ctx, "UPDATE course_catalog.semesters SET status = 'active' WHERE id = $1", springSem.ID)
	assert.Error(t, err, "a second active semester is refused even past the query's own guard")

	done, err := repo.CompleteSemester(ctx, utils.PgtypeToUUID(fallSem.ID))
	require.NoError(t, err)
	assert.True(t, done.CompletedAt.Valid)
	_, err = repo.ActivateSemester(ctx, utils.PgtypeToUUID(fallSem.ID))
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	_, err = pool.Exec(ctx, "UPDATE course_catalog.semesters SET status = 'active' WHERE id = $1", fallSem.ID)
	assert.Error(t, err, "a completed semester never comes back")

	_, err = repo.ActivateSemester(ctx, utils.PgtypeToUUID(springSem.ID))
	assert.NoError(t, err, "with the first completed, the next may start")
}

func TestAuditRepository_EntriesAreFinalAndEventsCountOnce(t *testing.T) {
	pool := pgtest.New(t, "course_catalog", "../sql/migrations")
	repo := NewAuditRepository(pool)
	ctx := context.Background()
	entry, err := repo.InsertAuditLog(ctx, db.InsertAuditLogParams{
		Service: "catalog", ActorID: utils.UUIDToPgtype(uuid.New()), ActorRole: "admin", Action: "course.create", ResourceType: "course",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "UPDATE course_catalog.audit_log SET action = 'x' WHERE id = $1", entry.ID)
	assert.Error(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM course_catalog.audit_log WHERE id = $1", entry.ID)
	assert.Error(t, err)

	fromEvent := db.InsertAuditLogFromEventParams{
		EventID: utils.UUIDToPgtype(uuid.New()), Service: "grades", ActorID: utils.UUIDToPgtype(uuid.New()), ActorRole: "teacher",
		Action: "grade.finalize", ResourceType: "course",
	}
	require.NoError(t, repo.InsertAuditLogFromEvent(ctx, fromEvent))
	require.NoError(t, repo.InsertAuditLogFromEvent(ctx, fromEvent), "a redelivered event is not an error")

	n, err := repo.CountAuditLog(ctx, db.CountAuditLogParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}
