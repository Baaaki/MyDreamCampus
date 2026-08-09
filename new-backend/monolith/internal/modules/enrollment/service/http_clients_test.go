package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	enrollmentErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBase(t *testing.T, handler http.HandlerFunc) *client.Base {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.NewBase(client.Config{
		Target:  "test-service",
		BaseURL: srv.URL,
		Secret:  "test-secret",
	})
}

func TestHTTPStudentClient_GetStudentByID_ReturnsStudent(t *testing.T) {
	id := uuid.New()
	var gotPath, gotSecret string
	c := NewHTTPStudentClient(newTestBase(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotSecret = r.URL.Path, r.Header.Get("X-Internal-Secret")
		_ = json.NewEncoder(w).Encode(contracts.StudentResponse{
			ID: id.String(), StudentNumber: "20250001", FirstName: "Ada", ClassLevel: 2,
		})
	}))

	student, err := c.GetStudentByID(context.Background(), id)
	require.NoError(t, err)

	assert.Equal(t, "20250001", student.StudentNumber)
	assert.Equal(t, "/internal/students/"+id.String(), gotPath)
	assert.Equal(t, "test-secret", gotSecret)
}

func TestHTTPStudentClient_GetStudentByID_NotFoundMapsToSentinel(t *testing.T) {
	c := NewHTTPStudentClient(newTestBase(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	_, err := c.GetStudentByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, enrollmentErrors.ErrStudentNotFound)
}

func TestHTTPStudentClient_GetStudentByID_ServerErrorIsNotNotFound(t *testing.T) {
	c := NewHTTPStudentClient(newTestBase(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := c.GetStudentByID(context.Background(), uuid.New())
	require.Error(t, err)
	assert.NotErrorIs(t, err, enrollmentErrors.ErrStudentNotFound)
	assert.ErrorIs(t, err, client.ErrUnavailable)
}

func TestHTTPStudentClient_GetStudentsByAdvisorID_ReturnsStudents(t *testing.T) {
	advisor := uuid.New()
	var gotAdvisor string
	c := NewHTTPStudentClient(newTestBase(t, func(w http.ResponseWriter, r *http.Request) {
		gotAdvisor = r.URL.Query().Get("advisor_id")
		_ = json.NewEncoder(w).Encode(adviseeList{
			Students: []contracts.StudentResponse{{ID: uuid.NewString(), FirstName: "Ada"}},
		})
	}))

	students, err := c.GetStudentsByAdvisorID(context.Background(), advisor)
	require.NoError(t, err)

	assert.Equal(t, advisor.String(), gotAdvisor)
	require.Len(t, students, 1)
	assert.Equal(t, "Ada", students[0].FirstName)
}

func TestHTTPCourseCatalogClient_GetAvailableCourses_SendsFilters(t *testing.T) {
	var gotQuery url.Values
	c := NewHTTPCourseCatalogClient(newTestBase(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(semesterCourseList{
			Data: []contracts.SemesterCourseListItem{{CourseCode: "BLM101"}},
		})
	}))

	courses, err := c.GetAvailableCourses(context.Background(), "Bilgisayar", 1, "2025-2026-Fall")
	require.NoError(t, err)

	require.Len(t, courses, 1)
	assert.Equal(t, "BLM101", courses[0].CourseCode)
	assert.Equal(t, "2025-2026-Fall", gotQuery.Get("semester"))
	assert.Equal(t, "Bilgisayar", gotQuery.Get("department"))
	assert.Equal(t, "1", gotQuery.Get("class_level"))
}

func TestHTTPCourseCatalogClient_GetCoursesByIDs_FetchesEach(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	var paths []string
	c := NewHTTPCourseCatalogClient(newTestBase(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_ = json.NewEncoder(w).Encode(contracts.SemesterCourseResponse{CourseCode: "BLM" + r.URL.Query().Get("semester")})
	}))

	courses, err := c.GetCoursesByIDs(context.Background(), "2025-2026-Fall", []uuid.UUID{first, second})
	require.NoError(t, err)

	assert.Len(t, courses, 2)
	assert.Equal(t, []string{
		"/internal/semester-courses/" + first.String(),
		"/internal/semester-courses/" + second.String(),
	}, paths)
}

func TestHTTPCourseCatalogClient_GetCoursesByIDs_NotFoundWrapsLikeInProcess(t *testing.T) {
	c := NewHTTPCourseCatalogClient(newTestBase(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	_, err := c.GetCoursesByIDs(context.Background(), "2025-2026-Fall", []uuid.UUID{uuid.New()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "course not found")
	assert.ErrorIs(t, err, client.ErrNotFound)
}
