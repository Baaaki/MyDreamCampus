package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret-key-minimum-32-characters-long")

func TestGenerateAccessTokenWithSecret(t *testing.T) {
	token, jti, err := GenerateAccessTokenWithSecret(
		"user-1", "student", "CS", 1, testSecret, 15,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, jti)
	assert.Equal(t, 2, strings.Count(token, "."), "JWT must have header.payload.signature")

	claims, err := ValidateTokenWithSecret(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "student", claims.Role)
	assert.Equal(t, "CS", claims.Department)
	assert.Equal(t, 1, claims.TokenVersion)
	assert.Equal(t, string(AccessToken), claims.TokenType)
	assert.Equal(t, jti, claims.JTI)
}

func TestGenerateRefreshTokenWithSecret(t *testing.T) {
	token, jti, err := GenerateRefreshTokenWithSecret("user-2", 5, testSecret, 24)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, jti)

	claims, err := ValidateTokenWithSecret(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "user-2", claims.UserID)
	assert.Equal(t, 5, claims.TokenVersion)
	assert.Equal(t, string(RefreshToken), claims.TokenType)
}

func TestValidateTokenWithSecret_RejectsBadSignature(t *testing.T) {
	token, _, err := GenerateAccessTokenWithSecret("u", "r", "", 1, testSecret, 15)
	require.NoError(t, err)

	_, err = ValidateTokenWithSecret(token, []byte("different-secret-key-minimum-32chars"))
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateTokenWithSecret_RejectsExpired(t *testing.T) {
	clock.Set(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	defer clock.Reset()

	token, _, err := GenerateAccessTokenWithSecret("u", "r", "", 1, testSecret, 15)
	require.NoError(t, err)

	clock.Set(time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)) // 1 hour later
	_, err = ValidateTokenWithSecret(token, testSecret)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateTokenWithSecret_RejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"not-a-jwt",
		"a.b.c",
		"eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoiYWJjIn0.",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			_, err := ValidateTokenWithSecret(c, testSecret)
			assert.Error(t, err)
		})
	}
}

func TestValidateTokenWithSecret_RejectsAlgNone(t *testing.T) {
	// craft a token with "alg: none" — must be rejected
	claims := &Claims{
		UserID:    "attacker",
		TokenType: string(AccessToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = ValidateTokenWithSecret(signed, testSecret)
	assert.ErrorIs(t, err, ErrInvalidToken, "alg:none must be rejected")
}

func TestValidateTokenIgnoreExpiry_AcceptsExpired(t *testing.T) {
	clock.Set(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	defer clock.Reset()

	token, _, err := GenerateAccessTokenWithSecret("u-3", "admin", "", 2, testSecret, 15)
	require.NoError(t, err)

	clock.Set(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)) // way past expiry
	claims, err := ValidateTokenIgnoreExpiryWithSecret(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "u-3", claims.UserID)
	assert.Equal(t, "admin", claims.Role)
}

func TestValidateTokenIgnoreExpiry_StillRejectsBadSignature(t *testing.T) {
	token, _, err := GenerateAccessTokenWithSecret("u", "r", "", 1, testSecret, 15)
	require.NoError(t, err)

	_, err = ValidateTokenIgnoreExpiryWithSecret(token, []byte("wrong-secret-of-sufficient-length-aaaaaa"))
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestGenerateTokens_UniqueJTI(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		_, jti, err := GenerateAccessTokenWithSecret("u", "r", "", 1, testSecret, 15)
		require.NoError(t, err)
		assert.False(t, seen[jti], "jti collision at iteration %d", i)
		seen[jti] = true
	}
}
