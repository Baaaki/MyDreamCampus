//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	catalogErrors "github.com/baaaki/mydreamcampus/catalog/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
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
