package dto

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateBinding runs the gin validator over a JSON body to test
// `binding:"..."` tags without instantiating handlers.
func validateBinding[T any](t *testing.T, body any) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var got T
	r.POST("/", func(c *gin.Context) {
		if err := c.ShouldBindJSON(&got); err != nil {
			c.AbortWithStatusJSON(400, gin.H{"err": err.Error()})
			return
		}
		c.Status(200)
	})

	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == 200 {
		return nil
	}
	return assert.AnError // return non-nil to signal validation rejection
}

func TestCreateStaffRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]any
		wantErr bool
	}{
		{"valid teacher", map[string]any{
			"email":      "t@uni.tr",
			"first_name": "Ada",
			"last_name":  "Lovelace",
			"role":       "teacher",
		}, false},
		{"missing email", map[string]any{
			"first_name": "A", "last_name": "B", "role": "teacher",
		}, true},
		{"invalid email", map[string]any{
			"email": "not-email", "first_name": "A", "last_name": "B", "role": "teacher",
		}, true},
		{"missing first_name", map[string]any{
			"email": "t@uni.tr", "last_name": "B", "role": "teacher",
		}, true},
		{"missing role", map[string]any{
			"email": "t@uni.tr", "first_name": "A", "last_name": "B",
		}, true},
		{"role admin not allowed", map[string]any{
			"email": "a@uni.tr", "first_name": "A", "last_name": "B", "role": "admin",
		}, true},
		{"role student not allowed", map[string]any{
			"email": "s@uni.tr", "first_name": "A", "last_name": "B", "role": "student",
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBinding[CreateStaffRequest](t, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStaffResponse_OmitsEmptyOptionals(t *testing.T) {
	r := StaffResponse{
		ID: "1", Email: "x@y.tr", FirstName: "A", LastName: "B",
		Role: "teacher", Status: "active",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	assert.NotContains(t, str, `"department"`)
	assert.NotContains(t, str, `"phone"`)
	assert.NotContains(t, str, `"office_location"`)
}

func TestStaffListResponse_RoundTrip(t *testing.T) {
	r := StaffListResponse{
		Data: []StaffResponse{{ID: "1", Email: "x@y.tr", Role: "teacher"}},
		Pagination: PaginationResponse{
			Page: 1, Limit: 20, Total: 1, TotalPages: 1,
		},
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)

	var got StaffListResponse
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, r.Pagination, got.Pagination)
	assert.Len(t, got.Data, 1)
}

func TestUpdateStaffRequest_OptionalPointers(t *testing.T) {
	// All-nil update should still parse — only sets fields that are present
	raw := []byte(`{}`)
	var got UpdateStaffRequest
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Nil(t, got.Department)

	dept := "Math"
	raw = []byte(`{"department":"Math"}`)
	got = UpdateStaffRequest{}
	require.NoError(t, json.Unmarshal(raw, &got))
	require.NotNil(t, got.Department)
	assert.Equal(t, dept, *got.Department)
}
