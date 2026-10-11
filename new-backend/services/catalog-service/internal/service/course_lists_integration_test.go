//go:build integration

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	"github.com/baaaki/mydreamcampus/catalog/internal/dto"
	"github.com/baaaki/mydreamcampus/catalog/internal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const itSemester = "2026-2027-Fall"

func newListServices(t *testing.T) (*CatalogService, *SemesterService, *pgxpool.Pool) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	pool := pgtest.New(t, "course_catalog", "../sql/migrations")
	catalogRepo := repository.NewCatalogRepository(pool)
	semesters := NewSemesterService(catalogRepo, repository.NewSemesterRepository(pool), repository.NewScheduleRepository(pool), nil, nil, nil, nil)
	return NewCatalogService(catalogRepo), semesters, pool
}

func jsonField(t *testing.T, v any, field string) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj))
	return string(obj[field])
}

// The admin and teacher pages iterate these lists; null instead of [] on an
// empty page crashed them.
func TestCourseLists_NoRows_MarshalEmptyArrays(t *testing.T) {
	catalog, semesters, _ := newListServices(t)
	ctx := context.Background()
	page := dto.PaginationRequest{Page: 1, Limit: 10}

	courses, err := catalog.ListCourses(ctx, dto.ListCoursesRequest{PaginationRequest: page})
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, jsonField(t, courses, "data"), "catalog courses")

	offered, err := semesters.ListSemesterCourses(ctx, itSemester, dto.ListSemesterCoursesRequest{PaginationRequest: page})
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, jsonField(t, offered, "data"), "semester courses")

	teacher, err := semesters.GetTeacherCourses(ctx, uuid.New(), nil)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, jsonField(t, teacher, "courses"), "teacher courses")
}

func openTeacherCourse(t *testing.T, pool *pgxpool.Pool, instructor uuid.UUID, code string) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO course_catalog.course_catalog
		(course_code, name, faculty, department, class_level, credits)
		VALUES ($1, $1, 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 1, 4)`, code)
	require.NoError(t, err)
	course, err := repository.NewSemesterRepository(pool).CreateSemesterCourse(ctx, db.CreateSemesterCourseParams{
		Semester: itSemester, CourseCode: code, Department: "Bilgisayar Mühendisliği", Credits: 4, ClassLevel: 1,
		InstructorID: utils.UUIDToPgtype(instructor), InstructorFullname: "Ahmet Yılmaz", ClassroomLocation: "A-101",
		MaxCapacity: 60, AssessmentSchema: []byte(`[{"slug":"final","weight":100}]`), Prerequisites: []byte(`[]`),
	})
	require.NoError(t, err)
	return course.ID
}

func TestGetTeacherCourses_ReadsEachCoursesOwnSchedule(t *testing.T) {
	_, semesters, pool := newListServices(t)
	ctx := context.Background()
	instructor := uuid.New()
	withLessons := openTeacherCourse(t, pool, instructor, "BLM101")
	openTeacherCourse(t, pool, instructor, "BLM102")
	_, err := repository.NewScheduleRepository(pool).CreateScheduleSession(ctx, db.CreateScheduleSessionParams{
		SemesterCourseID: withLessons, DayOfWeek: db.DayOfWeekEnum("monday"), SlotNumber: 1, SessionType: db.ScheduleSessionTypeEnum("theory"),
	})
	require.NoError(t, err)

	resp, err := semesters.GetTeacherCourses(ctx, instructor, nil)
	require.NoError(t, err)

	schedules := map[string][]dto.TeacherScheduleSession{}
	for _, c := range resp.Courses {
		schedules[c.CourseCode] = c.Schedule
	}
	require.Len(t, schedules, 2)
	require.Len(t, schedules["BLM101"], 1)
	assert.Equal(t, "monday", schedules["BLM101"][0].Day)
	assert.NotNil(t, schedules["BLM102"])
	assert.Empty(t, schedules["BLM102"])
}

// The list used to drop the read error and show the course with no lessons.
func TestGetTeacherCourses_ScheduleUnreadable_ReturnsInternal(t *testing.T) {
	_, semesters, pool := newListServices(t)
	ctx := context.Background()
	instructor := uuid.New()
	openTeacherCourse(t, pool, instructor, "BLM101")
	_, err := pool.Exec(ctx, `ALTER TABLE course_catalog.course_schedule_sessions RENAME TO course_schedule_sessions_gone`)
	require.NoError(t, err)

	_, err = semesters.GetTeacherCourses(ctx, instructor, nil)

	appErr, ok := sharedErrors.As(err)
	require.True(t, ok, "got %v", err)
	assert.Equal(t, sharedErrors.ErrInternal.Code, appErr.Code)
}
