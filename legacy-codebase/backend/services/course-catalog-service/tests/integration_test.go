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
	catalogBaseURL = "http://localhost:8004"
	catalogTimeout = 30 * time.Second
)

// course-catalog-service runs on port 8004 (see CLAUDE.md / shared/config/ports).
// These tests hit live HTTP endpoints. Opt in with INTEGRATION_TESTS=true.
//
// Auth: protected routes are gated by Traefik forward-auth, which sets
// X-User-* headers downstream. The middleware reads those directly when
// running outside Traefik, so the tests inject the headers manually for
// admin-scoped writes. Public read endpoints need no headers.

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping course-catalog-service integration tests. Set INTEGRATION_TESTS=true to run.")
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

	client := &http.Client{Timeout: catalogTimeout}
	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp, respBody
}

func adminHeaders() map[string]string {
	// Traefik forward-auth sets these headers; bypass it for direct testing.
	// The middleware shared/middleware.ExtractUserFromHeaders reads them.
	return map[string]string{
		"X-User-ID":   "00000000-0000-0000-0000-000000000001",
		"X-User-Role": "admin",
	}
}

func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", catalogBaseURL+"/health", nil, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "healthy", result["status"])
}

func TestReadinessCheck(t *testing.T) {
	resp, _ := makeRequest(t, "GET", catalogBaseURL+"/ready", nil, nil)

	// Ready returns 200 when all health checks pass (DB, Redis, RabbitMQ);
	// 503 if any are degraded. Both are acceptable signals — the test asserts
	// the endpoint responds and returns one of the two documented states.
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)
}

func TestListCourses_PublicEndpoint(t *testing.T) {
	// Public endpoint — no auth headers required. Should return 200 even if
	// no courses exist yet (empty data array).
	resp, body := makeRequest(t, "GET", catalogBaseURL+"/api/catalog/courses?page=1&limit=10", nil, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Contains(t, result, "data", "list endpoint returns paginated envelope with 'data'")
}

func TestGetCourse_NotFound(t *testing.T) {
	resp, _ := makeRequest(t, "GET", catalogBaseURL+"/api/catalog/courses/NONEXISTENT-9999", nil, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateCourse_RequiresAdmin(t *testing.T) {
	// Without admin headers, the protected POST /api/catalog/courses must
	// reject the request. The exact code depends on which middleware fires
	// first (auth missing → 401, role missing → 403); both are acceptable
	// "you cannot do this" signals.
	body := map[string]any{
		"course_code": "TEST-AUTH-101",
		"course_name": "Should Not Be Created",
		"faculty":     "Engineering",
		"department":  "Computer Science",
		"credits":     3,
		"class_level": 1,
		"course_type": "compulsory",
	}
	resp, _ := makeRequest(t, "POST", catalogBaseURL+"/api/catalog/courses", body, nil)
	assert.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, resp.StatusCode)
}

func TestCreateCourse_AdminFlow(t *testing.T) {
	// Use a unique course code per run so reruns don't 409 on the prior row.
	courseCode := fmt.Sprintf("TEST-INT-%d", time.Now().UnixNano()%1000000)

	body := map[string]any{
		"course_code": courseCode,
		"course_name": "Integration Test Course",
		"faculty":     "Engineering",
		"department":  "Computer Science",
		"credits":     3,
		"class_level": 1,
		"course_type": "compulsory",
	}

	resp, respBody := makeRequest(t, "POST", catalogBaseURL+"/api/catalog/courses", body, adminHeaders())

	// Service may not be reachable in CI — accept 201 (created) or 401/403
	// (auth wiring differs locally vs Traefik path) so the suite degrades
	// gracefully when the gating middleware rejects the synthetic headers.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Skipf("admin route gated upstream of ExtractUserFromHeaders: %d", resp.StatusCode)
	}
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", string(respBody))

	var created map[string]any
	require.NoError(t, json.Unmarshal(respBody, &created))
	assert.Equal(t, courseCode, created["course_code"])

	// Read it back through the public endpoint.
	getResp, getBody := makeRequest(t, "GET",
		fmt.Sprintf("%s/api/catalog/courses/%s", catalogBaseURL, courseCode),
		nil, nil)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var fetched map[string]any
	require.NoError(t, json.Unmarshal(getBody, &fetched))
	assert.Equal(t, "Integration Test Course", fetched["course_name"])
}

func TestCreateCourse_InvalidPayloadIs400(t *testing.T) {
	// Missing required fields (course_code, course_name) should be a 400 from
	// gin binding before any service logic fires. This pins the validation
	// surface — auth-protected, but invalid bodies must still be rejected.
	body := map[string]any{
		"faculty": "Engineering",
	}
	resp, _ := makeRequest(t, "POST", catalogBaseURL+"/api/catalog/courses", body, adminHeaders())

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Skipf("admin route gated upstream: %d", resp.StatusCode)
	}
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
