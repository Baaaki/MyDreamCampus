package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestHTTPStaffClient_GetAdvisorInfo_ReturnsAdvisor(t *testing.T) {
	id := uuid.New()
	var gotSecret string
	c := newStaffTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Internal-Secret")
		_ = json.NewEncoder(w).Encode(staffMember{
			ID: id.String(), FirstName: "Ada", LastName: "Lovelace",
			Role: "teacher", Status: "active",
		})
	})

	advisor, err := c.GetAdvisorInfo(context.Background(), id)
	require.NoError(t, err)

	assert.Equal(t, "Ada Lovelace", advisor.Name)
	assert.Equal(t, "test-secret", gotSecret)
}

func TestHTTPStaffClient_GetAdvisorInfo_NotFoundMatchesInProcessWording(t *testing.T) {
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetAdvisorInfo(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Equal(t, "advisor not found", err.Error())
}

func TestHTTPStaffClient_GetAdvisorInfo_NonTeacherRejected(t *testing.T) {
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(staffMember{ID: uuid.NewString(), Role: "admin", Status: "active"})
	})

	_, err := c.GetAdvisorInfo(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Equal(t, "staff is not a teacher", err.Error())
}

func TestHTTPStaffClient_GetAdvisorInfo_InactiveRejected(t *testing.T) {
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(staffMember{ID: uuid.NewString(), Role: "teacher", Status: "inactive"})
	})

	_, err := c.GetAdvisorInfo(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Equal(t, "advisor is not active", err.Error())
}

func TestHTTPStaffClient_GetAdvisorInfo_ServerErrorIsNotNotFound(t *testing.T) {
	c := newStaffTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetAdvisorInfo(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrUnavailable)
	assert.NotEqual(t, "advisor not found", err.Error())
}

func TestHTTPStaffClient_GetInstructorsByDepartment_ReturnsIDs(t *testing.T) {
	first := uuid.New()
	c := newStaffTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Fizik", r.URL.Query().Get("department"))
		_ = json.NewEncoder(w).Encode(staffListResponse{Data: []staffMember{
			{ID: first.String(), Role: "teacher", Status: "active"},
			{ID: "not-a-uuid"},
		}})
	})

	ids, err := c.GetInstructorsByDepartment(context.Background(), "Fizik")
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{first}, ids)
}
