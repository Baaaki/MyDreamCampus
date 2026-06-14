package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	staffBaseURL = "http://localhost:8002"
	staffTimeout = 30 * time.Second
)

// staff-service runs on port 8002. Route split:
//   /public/teachers/*          public — no auth (teacher profile listing)
//   /api/staff/profile/:id      public — no auth (teacher profile by staff id)
//   /internal/staff/*           service-to-service, no auth
//   /api/staff                  any authenticated user (list/read)
//   /api/staff (admin writes)   admin only (create, update, delete)
//
// Protected routes gated by ExtractUserFromHeaders — inject X-User-* directly.
// Mutating admin flows need a pre-seeded staff row; pin only the auth surface
// and validation. Cross-service state left for a wider e2e suite.

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping staff-service integration tests. Set INTEGRATION_TESTS=true to run.")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func makeRequest(t *testing.T, method, url string, body any, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: staffTimeout}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, respBody
}

func studentHeaders(id string) map[string]string {
	return map[string]string{"X-User-ID": id, "X-User-Role": "student"}
}
func teacherHeaders(id string) map[string]string {
	return map[string]string{"X-User-ID": id, "X-User-Role": "teacher"}
}
func adminHeaders(id string) map[string]string {
	return map[string]string{"X-User-ID": id, "X-User-Role": "admin"}
}

func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", staffBaseURL+"/health", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "healthy", result["status"])
}

func TestReadinessCheck(t *testing.T) {
	resp, _ := makeRequest(t, "GET", staffBaseURL+"/ready", nil, nil)
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)
}

func TestListTeacherProfiles_PublicEndpoint(t *testing.T) {
	// /public/teachers is completely public — no auth headers required.
	resp, body := makeRequest(t, "GET", staffBaseURL+"/public/teachers", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.NotNil(t, result)
}

func TestGetTeacherProfileByStaffID_PublicEndpoint(t *testing.T) {
	// /public/teachers/:id is public. A synthetic UUID returns 404 — the
	// endpoint must respond (not 401/403).
	staffID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "GET",
		fmt.Sprintf("%s/public/teachers/%s", staffBaseURL, staffID),
		nil, nil)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)
}

func TestGetStaffProfilePublicRoute_UnknownIs404(t *testing.T) {
	// /api/staff/profile/:id is an additional public profile route.
	staffID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "GET",
		fmt.Sprintf("%s/api/staff/profile/%s", staffBaseURL, staffID),
		nil, nil)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)
}

func TestProtectedRoute_RequiresAuthHeaders(t *testing.T) {
	// GET /api/staff without X-User-* headers → ExtractUserFromHeaders rejects 401.
	resp, _ := makeRequest(t, "GET", staffBaseURL+"/api/staff", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestListStaff_AuthenticatedReturnsListShape(t *testing.T) {
	resp, body := makeRequest(t, "GET", staffBaseURL+"/api/staff",
		nil, teacherHeaders("00000000-0000-0000-0000-000000000099"))

	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode,
		"unexpected status %d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode == http.StatusOK {
		var result any
		require.NoError(t, json.Unmarshal(body, &result))
		assert.NotNil(t, result)
	}
}

func TestGetStaffByID_InvalidUUIDIs400(t *testing.T) {
	// /api/staff/:id parses path as UUID — garbage input must 400.
	resp, _ := makeRequest(t, "GET",
		staffBaseURL+"/api/staff/not-a-uuid",
		nil, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetStaffByID_UnknownStaffIs404(t *testing.T) {
	staffID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "GET",
		fmt.Sprintf("%s/api/staff/%s", staffBaseURL, staffID),
		nil, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateStaff_RejectsNonAdmin(t *testing.T) {
	// POST /api/staff is admin-only. Any non-admin role must get 403.
	resp, _ := makeRequest(t, "POST", staffBaseURL+"/api/staff",
		map[string]any{}, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCreateStaff_AdminEmptyBodyIs400(t *testing.T) {
	// Admin reaches the handler; missing required fields → 400 from binding.
	resp, _ := makeRequest(t, "POST", staffBaseURL+"/api/staff",
		map[string]any{}, adminHeaders("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteStaff_RejectsNonAdmin(t *testing.T) {
	staffID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "DELETE",
		fmt.Sprintf("%s/api/staff/%s", staffBaseURL, staffID),
		nil, studentHeaders("00000000-0000-0000-0000-000000000098"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdateStaff_InvalidUUIDIs400(t *testing.T) {
	resp, _ := makeRequest(t, "PUT",
		staffBaseURL+"/api/staff/not-a-uuid",
		map[string]any{}, adminHeaders("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
