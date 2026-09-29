package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gradeErrors "github.com/baaaki/mydreamcampus/grades/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStudentTestClient(t *testing.T, handler http.HandlerFunc) *HTTPStudentClient {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewHTTPStudentClient(client.NewBase(client.Config{
		Target:  "student-service",
		BaseURL: srv.URL,
		Secret:  "test-secret",
	}))
}

func TestHTTPStudentClient_GetStudentByID_ReadsTheAdvisor(t *testing.T) {
	id := uuid.New()
	var gotPath, gotSecret string
	c := newStudentTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotSecret = r.URL.Path, r.Header.Get("X-Internal-Secret")
		_, _ = w.Write([]byte(`{"id":"` + id.String() + `","student_number":"2026001","advisor_id":"0190a000-0000-7000-8000-00000000000a"}`))
	})

	s, err := c.GetStudentByID(context.Background(), id)

	require.NoError(t, err)
	require.NotNil(t, s.AdvisorID)
	assert.Equal(t, "0190a000-0000-7000-8000-00000000000a", *s.AdvisorID)
	assert.Equal(t, "/internal/students/"+id.String(), gotPath)
	assert.Equal(t, "test-secret", gotSecret)
}

func TestHTTPStudentClient_GetStudentByID_NotFoundIsGradesOwnError(t *testing.T) {
	c := newStudentTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetStudentByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, gradeErrors.ErrStudentNotFound)
	assert.NotErrorIs(t, err, client.ErrUnavailable)
}

func TestHTTPStudentClient_GetStudentByID_ServerErrorMarksUnavailable(t *testing.T) {
	c := newStudentTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetStudentByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, client.ErrUnavailable)
}
