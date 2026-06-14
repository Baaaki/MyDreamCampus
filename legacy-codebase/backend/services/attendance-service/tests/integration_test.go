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
	attendanceBaseURL = "http://localhost:8006"
	attendanceTimeout = 30 * time.Second
)

// attendance-service runs on port 8006 (CLAUDE.md / shared/config/ports).
// Routes are split: /scan + /my are student-only, /sessions/* + /courses/*
// are teacher-only, /admin/* is admin-only. All gated by Traefik forward-auth
// (ExtractUserFromHeaders) — bypass with X-User-* headers in tests.
//
// The mutating teacher flow (POST /sessions, scan, close, finalize) needs a
// pre-seeded course + active session, which is cross-service state — leave
// that to a wider e2e suite. Here we pin the auth surface, validation, and
// the read-only endpoints.

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping attendance-service integration tests. Set INTEGRATION_TESTS=true to run.")
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

	client := &http.Client{Timeout: attendanceTimeout}
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

func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", attendanceBaseURL+"/health", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "healthy", result["status"])
}

func TestProtectedRoute_RequiresAuthHeaders(t *testing.T) {
	resp, _ := makeRequest(t, "GET", attendanceBaseURL+"/api/attendance/my", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestStudentRoute_RejectsTeacherRole(t *testing.T) {
	resp, _ := makeRequest(t, "GET", attendanceBaseURL+"/api/attendance/my",
		nil, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestTeacherRoute_RejectsStudentRole(t *testing.T) {
	resp, _ := makeRequest(t, "POST", attendanceBaseURL+"/api/attendance/sessions",
		map[string]any{}, studentHeaders("00000000-0000-0000-0000-000000000098"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGetMyAttendance_ReturnsListShape(t *testing.T) {
	// /my requires no query params; with a synthetic student ID the response
	// is either an empty list (200) or 404 if the cache rejects unknown IDs.
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, body := makeRequest(t, "GET", attendanceBaseURL+"/api/attendance/my",
		nil, studentHeaders(studentID))

	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode,
		"unexpected status %d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode == http.StatusOK {
		var result map[string]any
		require.NoError(t, json.Unmarshal(body, &result))
		// Shape pin: response is an envelope, not a bare array.
		assert.NotNil(t, result)
	}
}

func TestScanQR_RequiresQRCodeBody(t *testing.T) {
	// POST /scan binds a JSON body containing the QR token. Missing field
	// must 400 from gin binding before any service work runs.
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "POST", attendanceBaseURL+"/api/attendance/scan",
		map[string]any{}, studentHeaders(studentID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateSession_RequiresPayload(t *testing.T) {
	// POST /sessions requires CreateSessionRequest with required fields like
	// course_id + session_type. Empty body → 400.
	teacherID := "00000000-0000-0000-0000-000000000099"
	resp, _ := makeRequest(t, "POST", attendanceBaseURL+"/api/attendance/sessions",
		map[string]any{}, teacherHeaders(teacherID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSessionDetails_InvalidUUIDIs400(t *testing.T) {
	// /sessions/:session_id parses session_id as UUID — non-UUID path param
	// must reject before service layer.
	teacherID := "00000000-0000-0000-0000-000000000099"
	resp, _ := makeRequest(t, "GET",
		attendanceBaseURL+"/api/attendance/sessions/not-a-uuid",
		nil, teacherHeaders(teacherID))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSessionDetails_UnknownSessionIs404(t *testing.T) {
	teacherID := "00000000-0000-0000-0000-000000000099"
	sessionID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "GET",
		fmt.Sprintf("%s/api/attendance/sessions/%s", attendanceBaseURL, sessionID),
		nil, teacherHeaders(teacherID))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
