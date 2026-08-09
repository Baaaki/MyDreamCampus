package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginResponse_JSONShape(t *testing.T) {
	resp := LoginResponse{
		AccessToken:         "at",
		RefreshToken:        "rt",
		ExpiresIn:           900,
		ForcePasswordChange: true,
		User: UserResponse{
			ID:    "u-1",
			Email: "x@y.tr",
			Role:  "student",
		},
		Message: "Şifrenizi değiştirin",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	str := string(data)

	for _, key := range []string{
		"access_token", "refresh_token", "expires_in",
		"user", "force_password_change", "message",
		"u-1", "x@y.tr", "student",
	} {
		assert.Contains(t, str, key, "missing field/value: %s", key)
	}

	// Round-trip
	var got LoginResponse
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, resp, got)
}

func TestLoginResponse_OmitsEmptyMessage(t *testing.T) {
	resp := LoginResponse{AccessToken: "at"}
	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"message"`)
}

func TestUserResponse_OmitsNilDepartment(t *testing.T) {
	u := UserResponse{ID: "1", Email: "a@b.tr", Role: "admin"}
	data, err := json.Marshal(u)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "department")

	dept := "CS"
	u.Department = &dept
	data, err = json.Marshal(u)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"department":"CS"`)
}

func TestSessionResponse_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	dev := "Chrome/Linux"
	ip := "10.0.0.1"
	s := SessionResponse{
		ID:         "s-1",
		DeviceInfo: &dev,
		IPAddress:  &ip,
		CreatedAt:  now,
		LastUsedAt: now,
		IsCurrent:  true,
	}
	data, err := json.Marshal(s)
	require.NoError(t, err)

	var got SessionResponse
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, s, got)
}

func TestSessionsResponse_EmptyList(t *testing.T) {
	r := SessionsResponse{}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"sessions":null`,
		"nil slice serializes to null (current behavior)")
}

func TestErrorResponse_OmitsEmptyMessage(t *testing.T) {
	e := ErrorResponse{Error: "X"}
	data, err := json.Marshal(e)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"message"`)
}

func TestRefreshTokenRequest_Optional(t *testing.T) {
	// Empty body should still parse — this is the "web cookie client" case.
	var req RefreshTokenRequest
	require.NoError(t, json.Unmarshal([]byte(`{}`), &req))
	assert.Empty(t, req.RefreshToken)

	require.NoError(t, json.Unmarshal([]byte(`{"refresh_token":"abc"}`), &req))
	assert.Equal(t, "abc", req.RefreshToken)
}

func TestChangePasswordResponse_Shape(t *testing.T) {
	r := ChangePasswordResponse{
		Message:      "ok",
		AccessToken:  "at",
		RefreshToken: "rt",
		ExpiresIn:    900,
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	for _, key := range []string{"message", "access_token", "refresh_token", "expires_in"} {
		assert.True(t, strings.Contains(string(data), key))
	}
}
