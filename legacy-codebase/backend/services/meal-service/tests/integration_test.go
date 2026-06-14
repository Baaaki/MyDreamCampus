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
	mealBaseURL = "http://localhost:8008"
	mealTimeout = 30 * time.Second
)

// meal-service runs on port 8008. Route split:
//   /api/meals/menu/monthly     public GET (no auth)
//   /api/meals/cafeterias       any authenticated user
//   /api/meals/admin/*          admin only
//   /api/meals/reservations/*   student only
//
// Protected routes gated by ExtractUserFromHeaders — inject X-User-* directly.
// Mutating flows (create reservation, create menu) need pre-seeded cafeteria /
// menu data; leave those for a wider e2e suite. Pin auth surface and
// validation here.

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping meal-service integration tests. Set INTEGRATION_TESTS=true to run.")
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

	client := &http.Client{Timeout: mealTimeout}
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
func adminHeaders(id string) map[string]string {
	return map[string]string{"X-User-ID": id, "X-User-Role": "admin"}
}
func teacherHeaders(id string) map[string]string {
	return map[string]string{"X-User-ID": id, "X-User-Role": "teacher"}
}

func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", mealBaseURL+"/health", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "healthy", result["status"])
}

func TestReadinessCheck(t *testing.T) {
	resp, _ := makeRequest(t, "GET", mealBaseURL+"/ready", nil, nil)
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)
}

func TestGetMonthlyMenu_PublicEndpoint(t *testing.T) {
	// /api/meals/menu/monthly is public — no auth headers needed.
	// Returns 200 with data (possibly empty) or 400 if a required query
	// param (e.g. month) is missing — both are acceptable contract shapes.
	resp, body := makeRequest(t, "GET", mealBaseURL+"/api/meals/menu/monthly", nil, nil)
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, resp.StatusCode,
		"body=%s", string(body))
}

func TestGetCafeterias_RequiresAuth(t *testing.T) {
	resp, _ := makeRequest(t, "GET", mealBaseURL+"/api/meals/cafeterias", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetCafeterias_AuthenticatedReturnsListShape(t *testing.T) {
	resp, body := makeRequest(t, "GET", mealBaseURL+"/api/meals/cafeterias",
		nil, teacherHeaders("00000000-0000-0000-0000-000000000099"))

	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode,
		"unexpected status %d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode == http.StatusOK {
		var result any
		require.NoError(t, json.Unmarshal(body, &result))
		assert.NotNil(t, result)
	}
}

func TestAdminRoute_RejectsStudentRole(t *testing.T) {
	// POST /admin/cafeterias is admin-only; a student must get 403.
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/admin/cafeterias",
		map[string]any{}, studentHeaders("00000000-0000-0000-0000-000000000098"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAdminRoute_RejectsTeacherRole(t *testing.T) {
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/admin/cafeterias",
		map[string]any{}, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCreateCafeteria_AdminEmptyBodyIs400(t *testing.T) {
	// Admin can reach the handler; missing required fields → 400 from binding.
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/admin/cafeterias",
		map[string]any{}, adminHeaders("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStudentRoute_RejectsTeacherRole(t *testing.T) {
	// POST /reservations is student-only; teacher must get 403.
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/reservations",
		map[string]any{}, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestStudentRoute_RequiresAuthHeaders(t *testing.T) {
	resp, _ := makeRequest(t, "GET", mealBaseURL+"/api/meals/reservations/my", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetMyReservations_StudentReturnsListShape(t *testing.T) {
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, body := makeRequest(t, "GET", mealBaseURL+"/api/meals/reservations/my",
		nil, studentHeaders(studentID))

	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode,
		"unexpected status %d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode == http.StatusOK {
		var result any
		require.NoError(t, json.Unmarshal(body, &result))
		assert.NotNil(t, result)
	}
}

func TestCreateReservation_RequiresPayload(t *testing.T) {
	// POST /reservations binds CreateReservationRequest with required fields.
	// Empty body → 400 before service logic runs.
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/reservations",
		map[string]any{}, studentHeaders(studentID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateBatchReservation_RequiresPayload(t *testing.T) {
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/reservations/batch",
		map[string]any{}, studentHeaders(studentID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCancelReservation_InvalidUUIDIs400(t *testing.T) {
	// DELETE /reservations/:reservation_id parses the path param as UUID.
	// Garbage input must reject before service layer.
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "DELETE",
		mealBaseURL+"/api/meals/reservations/not-a-uuid",
		nil, studentHeaders(studentID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUseReservation_RequiresPayload(t *testing.T) {
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/reservations/use",
		map[string]any{}, studentHeaders(studentID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateMonthlyMenu_RequiresPayload(t *testing.T) {
	adminID := "00000000-0000-0000-0000-000000000001"
	resp, _ := makeRequest(t, "POST", mealBaseURL+"/api/meals/admin/menu/monthly",
		map[string]any{}, adminHeaders(adminID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
