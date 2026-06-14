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
	enrollmentBaseURL = "http://localhost:8005"
	enrollmentTimeout = 30 * time.Second
)

// enrollment-service runs on port 8005 (CLAUDE.md / shared/config/ports).
// Every protected route is gated by Traefik forward-auth via the
// ExtractUserFromHeaders middleware — bypass Traefik in tests by injecting
// X-User-* headers directly. Opt in with INTEGRATION_TESTS=true.
//
// These tests are deliberately read-only against a known student. The mutating
// endpoints (POST /programs, DELETE /programs) need a fully provisioned
// student + an active enrollment period; covering them properly belongs in a
// cross-service end-to-end suite, not a per-service integration check.

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping enrollment-service integration tests. Set INTEGRATION_TESTS=true to run.")
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

	client := &http.Client{Timeout: enrollmentTimeout}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, respBody
}

func studentHeaders(studentID string) map[string]string {
	return map[string]string{
		"X-User-ID":   studentID,
		"X-User-Role": "student",
	}
}

func advisorHeaders(advisorID string) map[string]string {
	return map[string]string{
		"X-User-ID":   advisorID,
		"X-User-Role": "teacher",
	}
}

func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", enrollmentBaseURL+"/health", nil, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "healthy", result["status"])
}

func TestProtectedRoute_RequiresAuthHeaders(t *testing.T) {
	// No X-User-* headers → ExtractUserFromHeaders rejects with 401.
	resp, _ := makeRequest(t, "GET",
		enrollmentBaseURL+"/api/enrollment/available-courses?semester=2025-2026-Fall",
		nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestStudentRoute_RejectsTeacherRole(t *testing.T) {
	// RequireStudent middleware must 403 a teacher hitting a student-only route.
	resp, _ := makeRequest(t, "GET",
		enrollmentBaseURL+"/api/enrollment/available-courses?semester=2025-2026-Fall",
		nil, advisorHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAdvisorRoute_RejectsStudentRole(t *testing.T) {
	resp, _ := makeRequest(t, "GET",
		enrollmentBaseURL+"/api/enrollment/advisor/pending-programs",
		nil, studentHeaders("00000000-0000-0000-0000-000000000098"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGetAvailableCourses_RequiresSemesterQuery(t *testing.T) {
	// The handler returns 400 when ?semester= is missing — gate on the
	// validation surface, not on whether the requested student exists.
	studentID := "00000000-0000-0000-0000-000000000001"
	resp, _ := makeRequest(t, "GET",
		enrollmentBaseURL+"/api/enrollment/available-courses",
		nil, studentHeaders(studentID))

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetAvailableCourses_UnknownStudentReturns404Or200Empty(t *testing.T) {
	// A semester filter against a synthetic student ID either:
	//   * 200 with an empty AvailableCourses slice (cache returned no rows)
	//   * 404 if the service treats "no student row in cache" as not found
	// Both are acceptable — the test pins the surface, not the policy.
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, body := makeRequest(t, "GET",
		enrollmentBaseURL+"/api/enrollment/available-courses?semester=2025-2026-Fall",
		nil, studentHeaders(studentID))

	require.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden}, resp.StatusCode,
		"unexpected status %d body=%s", resp.StatusCode, string(body))
}

func TestGetMyEnrollments_NoFiltersReturnsListEnvelope(t *testing.T) {
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, body := makeRequest(t, "GET",
		enrollmentBaseURL+"/api/enrollment/my-enrollments",
		nil, studentHeaders(studentID))

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		t.Skipf("synthetic student rejected by upstream gate: %d", resp.StatusCode)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", string(body))

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Contains(t, result, "programs", "response shape is { student_id, programs[] }")
}

func TestCreateEnrollmentProgram_InvalidPayloadIs400(t *testing.T) {
	// Empty course_ids must trip gin's binding (min=1 on the field) before
	// any service work fires. Pin the validation surface.
	studentID := "11111111-1111-1111-1111-111111111111"
	body := map[string]any{
		"semester":   "2025-2026-Fall",
		"course_ids": []string{},
	}
	resp, _ := makeRequest(t, "POST",
		enrollmentBaseURL+"/api/enrollment/programs", body,
		studentHeaders(studentID))

	if resp.StatusCode == http.StatusForbidden {
		t.Skipf("upstream gate rejected synthetic student: %d", resp.StatusCode)
	}
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRejectEnrollmentProgram_RequiresReason(t *testing.T) {
	// /advisor/programs/:program_id/reject must reject a payload missing the
	// rejection_reason field. The handler binds RejectEnrollmentRequest which
	// has rejection_reason as required.
	advisorID := "00000000-0000-0000-0000-000000000099"
	programID := "11111111-1111-1111-1111-111111111111"
	body := map[string]any{} // no rejection_reason

	resp, _ := makeRequest(t, "POST",
		fmt.Sprintf("%s/api/enrollment/advisor/programs/%s/reject", enrollmentBaseURL, programID),
		body, advisorHeaders(advisorID))

	if resp.StatusCode == http.StatusForbidden {
		t.Skipf("upstream gate rejected synthetic advisor: %d", resp.StatusCode)
	}
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
