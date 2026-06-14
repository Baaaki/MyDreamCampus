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
	gradesBaseURL = "http://localhost:8007"
	gradesTimeout = 30 * time.Second
)

// grades-service runs on port 8007. Routes split by role:
//   /course/*    teacher or admin
//   /student/my  any student
//   /transcript  any authenticated user (own or other's, depending on policy)
//   /admin/*     admin only
// All gated by Traefik forward-auth via ExtractUserFromHeaders.
//
// The full grade-submission flow (POST /course/:id/scores → outbox → finalize
// consumer → grade.finalized event) needs cross-service state. Pin the auth
// surface, validation, and read-only paths here; the cross-service flow
// belongs in a wider e2e suite.

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping grades-service integration tests. Set INTEGRATION_TESTS=true to run.")
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

	client := &http.Client{Timeout: gradesTimeout}
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
	resp, body := makeRequest(t, "GET", gradesBaseURL+"/health", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "healthy", result["status"])
}

func TestProtectedRoute_RequiresAuthHeaders(t *testing.T) {
	resp, _ := makeRequest(t, "GET", gradesBaseURL+"/api/grades/student/my", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestTeacherRoute_RejectsStudentRole(t *testing.T) {
	courseID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "GET",
		fmt.Sprintf("%s/api/grades/course/%s/status", gradesBaseURL, courseID),
		nil, studentHeaders("00000000-0000-0000-0000-000000000098"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAdminRoute_RejectsTeacherRole(t *testing.T) {
	resp, _ := makeRequest(t, "POST", gradesBaseURL+"/api/grades/admin/appeal",
		map[string]any{}, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGetMyGrades_StudentReturnsListShape(t *testing.T) {
	studentID := "11111111-1111-1111-1111-111111111111"
	resp, body := makeRequest(t, "GET", gradesBaseURL+"/api/grades/student/my",
		nil, studentHeaders(studentID))

	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode,
		"unexpected status %d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode == http.StatusOK {
		var result any
		require.NoError(t, json.Unmarshal(body, &result))
		assert.NotNil(t, result, "OK response must have a body")
	}
}

func TestGetCourseStatus_InvalidUUIDIs400(t *testing.T) {
	resp, _ := makeRequest(t, "GET",
		gradesBaseURL+"/api/grades/course/not-a-uuid/status",
		nil, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSubmitScore_RequiresPayload(t *testing.T) {
	courseID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "POST",
		fmt.Sprintf("%s/api/grades/course/%s/scores", gradesBaseURL, courseID),
		map[string]any{}, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBulkSubmitScores_RequiresArray(t *testing.T) {
	// /scores/bulk binds to a slice; an object missing the array field must
	// 400 from gin's binding before service logic runs.
	courseID := "11111111-1111-1111-1111-111111111111"
	resp, _ := makeRequest(t, "POST",
		fmt.Sprintf("%s/api/grades/course/%s/scores/bulk", gradesBaseURL, courseID),
		map[string]any{}, teacherHeaders("00000000-0000-0000-0000-000000000099"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetTranscript_InvalidStudentIDIs400(t *testing.T) {
	// /transcript/:student_id parses the path as UUID. Garbage input → 400.
	resp, _ := makeRequest(t, "GET",
		gradesBaseURL+"/api/grades/transcript/not-a-uuid",
		nil, studentHeaders("11111111-1111-1111-1111-111111111111"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProcessAppeal_AdminWithEmptyBodyIs400(t *testing.T) {
	resp, _ := makeRequest(t, "POST", gradesBaseURL+"/api/grades/admin/appeal",
		map[string]any{}, adminHeaders("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
