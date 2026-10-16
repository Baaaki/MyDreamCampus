package dto

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportJobResponse_OmitsOptionalFields(t *testing.T) {
	r := ImportJobResponse{
		JobID: "j-1", FileName: "x.csv", Status: "pending",
		TotalRecords: 100, ProcessedRecords: 0, CreatedBy: "u-1",
		CreatedAt: time.Now(),
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	assert.NotContains(t, str, "errors", "empty errors should be omitted")
	assert.NotContains(t, str, "started_at")
	assert.NotContains(t, str, "completed_at")
}

func TestImportJobError_RoundTrip(t *testing.T) {
	e := ImportJobError{
		Row: 42, StudentNumber: "20240001",
		ErrorCode: "DUPLICATE_EMAIL", Message: "email already exists",
	}
	data, err := json.Marshal(e)
	require.NoError(t, err)
	for _, want := range []string{"row", "student_number", "error_code", "message", "20240001"} {
		assert.Contains(t, string(data), want)
	}

	var got ImportJobError
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, e, got)
}

func TestCSVStudentRecord_FieldsAreTaggedForCSV(t *testing.T) {
	// We just verify the type compiles + field presence; CSV parsing
	// happens via gocsv at runtime.
	r := CSVStudentRecord{
		StudentNumber: "20240001", FirstName: "Ada", LastName: "Lovelace",
		Email: "a@b.tr", Faculty: "Eng", Department: "CS",
		EnrollmentYear: 2024, ClassLevel: 1,
	}
	assert.NotEmpty(t, r.StudentNumber)
	assert.Equal(t, int16(1), r.ClassLevel)
}

// The list used to embed PaginationQuery, whose student-status oneof turned
// ?status=pending into a 400. Unknown params must now be ignored.
func TestImportJobFilterQuery_StatusParam_BindsWithDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/imports?status=pending", nil)

	var q ImportJobFilterQuery
	require.NoError(t, binding.Query.Bind(req, &q))
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 20, q.Limit)
}

func TestImportJobFilterQuery_LimitAboveMax_Rejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/imports?limit=101", nil)

	var q ImportJobFilterQuery
	require.Error(t, binding.Query.Bind(req, &q))
}

func TestStaffAdvisor_RoundTrip(t *testing.T) {
	id := uuid.New()
	a := StaffAdvisor{
		ID: id, FirstName: "X", LastName: "Y",
		Email: "x@y.tr", Department: "CS",
	}
	data, err := json.Marshal(a)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), id.String()))

	var got StaffAdvisor
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, a, got)
}
