package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL        = "http://localhost:8003"
	staffBaseURL   = "http://localhost:8002"
	testTimeout    = 30 * time.Second
	importWaitTime = 3 * time.Second
)

var (
	testStudentID string
	testAdvisorID string
	testJobID     string
)

// TestMain gates the entire suite — each of these tests hits live HTTP
// endpoints on localhost (8003 + 8002). Opt in with INTEGRATION_TESTS=true.
func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping student-service integration tests. Set INTEGRATION_TESTS=true to run.")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// Helper function to make HTTP requests
func makeRequest(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err, "Failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: testTimeout}
	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to execute request")

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	defer resp.Body.Close()

	return resp, respBody
}

// Test 1: Health Check
func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", baseURL+"/health", nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "student-service", result["service"])
	assert.Equal(t, "healthy", result["status"])

	t.Logf("✅ Health Check: %s", string(body))
}

// Test 2: Create Advisor (prerequisite for creating students)
func TestCreateAdvisor(t *testing.T) {
	// Generate unique email with timestamp
	timestamp := time.Now().UnixNano()
	reqBody := map[string]any{
		"email":           fmt.Sprintf("test.advisor.%d@university.edu.tr", timestamp),
		"first_name":      "Test",
		"last_name":       "Advisor",
		"role":            "teacher",
		"department":      "Computer Engineering",
		"phone":           "+90 555 999 8888",
		"office_location": "Test Building, Room 101",
	}

	resp, body := makeRequest(t, "POST", staffBaseURL+"/api/staff", reqBody)

	if resp.StatusCode == http.StatusConflict {
		t.Log("⚠️ Advisor already exists, using existing one")
		// Get existing advisor
		listResp, listBody := makeRequest(t, "GET", staffBaseURL+"/api/staff?page=1&limit=1", nil)
		require.Equal(t, http.StatusOK, listResp.StatusCode)

		var listResult map[string]any
		err := json.Unmarshal(listBody, &listResult)
		require.NoError(t, err)

		data := listResult["data"].([]any)
		if len(data) > 0 {
			first := data[0].(map[string]any)
			testAdvisorID = first["id"].(string)
			t.Logf("✅ Using Existing Advisor: ID=%s", testAdvisorID)
			return
		}
	}

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	testAdvisorID = result["id"].(string)
	assert.NotEmpty(t, testAdvisorID)

	t.Logf("✅ Created Advisor: ID=%s", testAdvisorID)
}

// Test 3: Create Student
func TestCreateStudent(t *testing.T) {
	require.NotEmpty(t, testAdvisorID, "Advisor must be created first")

	reqBody := map[string]any{
		"student_number":  "TEST001",
		"first_name":      "Integration",
		"last_name":       "Test",
		"email":           "integration.test@university.edu.tr",
		"faculty":         "Engineering",
		"department":      "Computer Engineering",
		"enrollment_year": 2024,
		"class_level":     1,
		"advisor_id":      testAdvisorID,
	}

	resp, body := makeRequest(t, "POST", baseURL+"/api/v1/students", reqBody)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	testStudentID = result["id"].(string)
	assert.NotEmpty(t, testStudentID)
	assert.Equal(t, "TEST001", result["student_number"])
	assert.Equal(t, "Integration", result["first_name"])
	assert.Equal(t, testAdvisorID, result["advisor_id"])

	t.Logf("✅ Created Student: ID=%s", testStudentID)
}

// Test 4: Get Student by ID
func TestGetStudentByID(t *testing.T) {
	require.NotEmpty(t, testStudentID, "Student must be created first")

	url := fmt.Sprintf("%s/api/v1/students/%s", baseURL, testStudentID)
	resp, body := makeRequest(t, "GET", url, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, testStudentID, result["id"])
	assert.Equal(t, "TEST001", result["student_number"])

	t.Logf("✅ Retrieved Student: %s %s", result["first_name"], result["last_name"])
}

// Test 5: List Students
func TestListStudents(t *testing.T) {
	resp, body := makeRequest(t, "GET", baseURL+"/api/v1/students?page=1&limit=10", nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	data := result["data"].([]any)
	pagination := result["pagination"].(map[string]any)

	assert.NotEmpty(t, data)
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(10), pagination["limit"])

	t.Logf("✅ Listed Students: Total=%v", pagination["total"])
}

// Test 6: Update Student
func TestUpdateStudent(t *testing.T) {
	require.NotEmpty(t, testStudentID, "Student must be created first")

	reqBody := map[string]any{
		"class_level": 2,
		"status":      "active",
	}

	url := fmt.Sprintf("%s/api/v1/students/%s", baseURL, testStudentID)
	resp, body := makeRequest(t, "PUT", url, reqBody)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result["class_level"])
	assert.Equal(t, "active", result["status"])

	t.Logf("✅ Updated Student: Class Level changed to %v", result["class_level"])
}

// Test 7: Bulk Import (PostgreSQL COPY)
func TestBulkImport(t *testing.T) {
	// Create CSV file with unique student numbers using timestamp
	timestamp := time.Now().UnixNano() / 1000000 // milliseconds
	csvContent := fmt.Sprintf(`student_number,first_name,last_name,email,faculty,department,enrollment_year,class_level
TEST%d1,Bulk1,Import1,bulk1.%d@university.edu.tr,Engineering,Computer Engineering,2024,1
TEST%d2,Bulk2,Import2,bulk2.%d@university.edu.tr,Engineering,Computer Engineering,2024,1
TEST%d3,Bulk3,Import3,bulk3.%d@university.edu.tr,Engineering,Electrical Engineering,2024,1
`, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp)

	// Create temporary CSV file
	tmpFile, err := os.CreateTemp("", "test_students_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Prepare multipart form
	file, err := os.Open(tmpFile.Name())
	require.NoError(t, err)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(tmpFile.Name()))
	require.NoError(t, err)

	_, err = io.Copy(part, file)
	require.NoError(t, err)
	writer.Close()

	// Make request
	req, err := http.NewRequest("POST", baseURL+"/api/v1/students/bulk-import", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: testTimeout}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var result map[string]any
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	testJobID = result["job_id"].(string)
	assert.NotEmpty(t, testJobID)
	assert.Equal(t, "pending", result["status"])

	t.Logf("✅ Bulk Import Started: Job ID=%s", testJobID)
}

// Test 8: Get Import Job Status
func TestGetImportJobStatus(t *testing.T) {
	require.NotEmpty(t, testJobID, "Bulk import must be started first")

	// Wait for import to complete
	t.Logf("⏳ Waiting %v for import to complete...", importWaitTime)
	time.Sleep(importWaitTime)

	url := fmt.Sprintf("%s/api/v1/students/bulk-import/%s", baseURL, testJobID)
	resp, body := makeRequest(t, "GET", url, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, testJobID, result["job_id"])
	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, float64(3), result["total_records"])
	assert.Equal(t, float64(3), result["successful_records"])
	assert.Equal(t, float64(0), result["failed_records"])
	assert.Equal(t, float64(100), result["progress_percentage"])

	t.Logf("✅ Import Completed: %v/%v successful (%.0f%% progress)",
		result["successful_records"],
		result["total_records"],
		result["progress_percentage"])
}

// Test 9: List Import Jobs
func TestListImportJobs(t *testing.T) {
	resp, body := makeRequest(t, "GET", baseURL+"/api/v1/students/bulk-import?page=1&limit=10", nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	data := result["data"].([]any)
	pagination := result["pagination"].(map[string]any)

	assert.NotEmpty(t, data)
	assert.GreaterOrEqual(t, len(data), 1)
	assert.Equal(t, float64(1), pagination["page"])

	t.Logf("✅ Listed Import Jobs: Total=%v", pagination["total"])
}

// Test 10: Delete Student
func TestDeleteStudent(t *testing.T) {
	require.NotEmpty(t, testStudentID, "Student must be created first")

	url := fmt.Sprintf("%s/api/v1/students/%s", baseURL, testStudentID)
	resp, body := makeRequest(t, "DELETE", url, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "Student deleted successfully", result["message"])

	t.Logf("✅ Deleted Student: ID=%s (soft delete - deleted_at set)", testStudentID)

	// Verify deletion - soft deleted students should not be retrievable
	resp, _ = makeRequest(t, "GET", url, nil)
	// After soft delete, student should not be found (filtered by deleted_at IS NULL)
	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Deleted student should not return 200 OK")

	t.Logf("✅ Verified Soft Delete: Student not accessible (status: %d)", resp.StatusCode)
}

// Test order matters! Run tests sequentially.
// Requires student-service (8003) and staff-service (8002) running locally.
// Gate with INTEGRATION_TESTS=true to opt in.
func TestStudentServiceIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=true to run.")
	}
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("1_HealthCheck", TestHealthCheck)
	t.Run("2_CreateAdvisor", TestCreateAdvisor)
	t.Run("3_CreateStudent", TestCreateStudent)
	t.Run("4_GetStudentByID", TestGetStudentByID)
	t.Run("5_ListStudents", TestListStudents)
	t.Run("6_UpdateStudent", TestUpdateStudent)
	t.Run("7_BulkImport", TestBulkImport)
	t.Run("8_GetImportJobStatus", TestGetImportJobStatus)
	t.Run("9_ListImportJobs", TestListImportJobs)
	t.Run("10_DeleteStudent", TestDeleteStudent)
}
