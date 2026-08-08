package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	catalogErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/errors"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStaffTestClient(t *testing.T, handler http.HandlerFunc) *HTTPStaffClient {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewHTTPStaffClient(client.NewBase(client.Config{
		Target:  "staff-service",
		BaseURL: srv.URL,
		Secret:  "test-secret",
	}))
}

func TestHTTPStaffClient_GetInstructor_ReturnsInstructorInfo(t *testing.T) {
	id := uuid.New()
	var gotPath, gotSecret string
	c := newStaffTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSecret = r.Header.Get("X-Internal-Secret")
		_ = json.NewEncoder(w).Encode(staffMember{
			ID: id.String(), FirstName: "Ada", LastName: "Lovelace",
			Department: "Bilgisayar Muhendisligi", Status: "active",
		})
	})

	info, err := c.GetInstructor(context.Background(), id, "Bilgisayar Muhendisligi")
	require.NoError(t, err)

	assert.Equal(t, id, info.ID)
	assert.Equal(t, "Ada Lovelace", info.FullName)
	assert.Equal(t, "/internal/staff/"+id.String(), gotPath)
	assert.Equal(t, "test-secret", gotSecret)
}

func TestHTTPStaffClient_GetInstructor_NotFoundMapsToSentinel(t *testing.T) {
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetInstructor(context.Background(), uuid.New(), "Bilgisayar Muhendisligi")
	assert.ErrorIs(t, err, catalogErrors.ErrInstructorNotFound)
}

func TestHTTPStaffClient_GetInstructor_InactiveMapsToSentinel(t *testing.T) {
	id := uuid.New()
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(staffMember{ID: id.String(), Department: "Matematik", Status: "inactive"})
	})

	_, err := c.GetInstructor(context.Background(), id, "Matematik")
	assert.ErrorIs(t, err, catalogErrors.ErrInstructorNotActive)
}

func TestHTTPStaffClient_GetInstructor_OtherDepartmentMapsToSentinel(t *testing.T) {
	id := uuid.New()
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(staffMember{ID: id.String(), Department: "Matematik", Status: "active"})
	})

	_, err := c.GetInstructor(context.Background(), id, "Fizik")
	assert.ErrorIs(t, err, catalogErrors.ErrInstructorNotInDepartment)
}

func TestHTTPStaffClient_GetInstructor_ServerErrorPropagates(t *testing.T) {
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetInstructor(context.Background(), uuid.New(), "Fizik")
	require.Error(t, err)
	assert.NotErrorIs(t, err, catalogErrors.ErrInstructorNotFound, "an outage must not look like a missing instructor")
	assert.ErrorIs(t, err, client.ErrUnavailable)
}

func TestHTTPStaffClient_GetInstructorsByDepartment_ReturnsList(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	var gotQuery string
	c := newStaffTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("department")
		_ = json.NewEncoder(w).Encode(staffListResponse{Data: []staffMember{
			{ID: first.String(), FirstName: "Ada", LastName: "L", Department: "Fizik", Status: "active"},
			{ID: "not-a-uuid", FirstName: "Broken", LastName: "Row"},
			{ID: second.String(), FirstName: "Grace", LastName: "H", Department: "Fizik", Status: "active"},
		}})
	})

	list, err := c.GetInstructorsByDepartment(context.Background(), "Fizik")
	require.NoError(t, err)

	assert.Equal(t, "Fizik", gotQuery)
	require.Len(t, list, 2, "unparsable rows are skipped, as in-process does")
	assert.Equal(t, first, list[0].ID)
	assert.Equal(t, "Grace H", list[1].FullName)
}
