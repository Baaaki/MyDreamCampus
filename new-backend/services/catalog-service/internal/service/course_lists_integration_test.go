//go:build integration

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/catalog/internal/dto"
	"github.com/baaaki/mydreamcampus/catalog/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
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
