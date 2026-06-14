package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestLoginValidation tests login input validation
func TestLoginValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    any
		expectedStatus int
	}{
		{
			name: "Invalid email format",
			requestBody: map[string]string{
				"email":    "not-an-email",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty password",
			requestBody: map[string]string{
				"email":    "test@university.edu.tr",
				"password": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Short password",
			requestBody: map[string]string{
				"email":    "test@university.edu.tr",
				"password": "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing email field",
			requestBody: map[string]string{
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()

			// Simple handler that just validates input
			router.POST("/login", func(c *gin.Context) {
				var req dto.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, dto.ErrorResponse{
						Error:   "VALIDATION_ERROR",
						Message: err.Error(),
					})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// Prepare request
			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			// Execute
			router.ServeHTTP(resp, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, resp.Code)
		})
	}
}

// TestChangePasswordValidation tests password change validation
func TestChangePasswordValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]string
		expectedStatus int
	}{
		{
			name: "Short new password",
			requestBody: map[string]string{
				"old_password": "OldPassword123!",
				"new_password": "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty old password",
			requestBody: map[string]string{
				"old_password": "",
				"new_password": "NewPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing new password",
			requestBody: map[string]string{
				"old_password": "OldPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()

			router.POST("/change-password", func(c *gin.Context) {
				var req dto.ChangePasswordRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, dto.ErrorResponse{
						Error:   "VALIDATION_ERROR",
						Message: err.Error(),
					})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// Prepare request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/change-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			// Execute
			router.ServeHTTP(resp, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, resp.Code)
		})
	}
}

// TestDTOSerialization tests that DTOs serialize correctly to JSON
func TestDTOSerialization(t *testing.T) {
	t.Run("LoginResponse serialization", func(t *testing.T) {
		response := dto.LoginResponse{
			AccessToken: "test-token",
			ExpiresIn:   900,
			User: dto.UserResponse{
				ID:    "test-id",
				Email: "test@example.com",
				Role:  "student",
			},
			ForcePasswordChange: false,
		}

		data, err := json.Marshal(response)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "access_token")
		assert.Contains(t, string(data), "expires_in")
		assert.Contains(t, string(data), "user")
	})

	t.Run("ErrorResponse serialization", func(t *testing.T) {
		response := dto.ErrorResponse{
			Error:   "TEST_ERROR",
			Message: "Test error message",
		}

		data, err := json.Marshal(response)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "error")
		assert.Contains(t, string(data), "message")
	})
}
