package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tokenTestSecret = "auth-token-test-secret-at-least-32-bytes"

func newTokenTestService() *AuthService {
	cfg := &config.Config{}
	cfg.JWT.Secret = tokenTestSecret
	cfg.JWT.AccessTokenExpiry = 15
	cfg.JWT.RefreshTokenExpiry = 24
	return &AuthService{config: cfg}
}

func tokenTestUser() db.User {
	return db.User{
		ID:           pgUUID(uuid.New()),
		Email:        "ogrenci@uni.edu.tr",
		Role:         "student",
		TokenVersion: utils.Int32Ptr(3),
	}
}

func TestGenerateAccessToken_PassesSharedAccessValidation(t *testing.T) {
	s := newTokenTestService()

	token, err := s.generateAccessToken(tokenTestUser())
	require.NoError(t, err)

	claims, err := utils.ValidateAccessTokenWithSecret(token, []byte(tokenTestSecret))
	require.NoError(t, err)
	assert.Equal(t, string(utils.AccessToken), claims.TokenType)
	assert.Equal(t, "student", claims.Role)
}

func TestGenerateRefreshToken_RejectedAsAccessToken(t *testing.T) {
	s := newTokenTestService()

	token, _, err := s.generateRefreshToken(tokenTestUser())
	require.NoError(t, err)

	_, err = utils.ValidateAccessTokenWithSecret(token, []byte(tokenTestSecret))
	assert.ErrorIs(t, err, utils.ErrWrongTokenType)
}

func TestParseRefreshToken_AccessToken_Rejected(t *testing.T) {
	s := newTokenTestService()
	access, err := s.generateAccessToken(tokenTestUser())
	require.NoError(t, err)

	_, err = s.parseRefreshToken(access)
	assert.ErrorIs(t, err, utils.ErrWrongTokenType)

	_, err = s.parseRefreshTokenWithoutValidation(access)
	assert.ErrorIs(t, err, utils.ErrWrongTokenType)
}

func TestParseRefreshToken_RefreshToken_Accepted(t *testing.T) {
	s := newTokenTestService()
	user := tokenTestUser()
	refresh, jti, err := s.generateRefreshToken(user)
	require.NoError(t, err)

	claims, err := s.parseRefreshToken(refresh)
	require.NoError(t, err)
	assert.Equal(t, jti, claims["jti"])
	assert.Equal(t, utils.PgtypeToUUID(user.ID).String(), claims["user_id"])
}

func TestTokenParsers_HS512Signed_Rejected(t *testing.T) {
	s := newTokenTestService()
	s.redisClient = newFakeAuthCache()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"user_id":    uuid.NewString(),
		"jti":        uuid.NewString(),
		"token_type": string(utils.RefreshToken),
		"exp":        time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(tokenTestSecret))
	require.NoError(t, err)

	_, err = s.parseRefreshToken(token)
	assert.Error(t, err)

	_, err = s.parseRefreshTokenWithoutValidation(token)
	assert.Error(t, err)

	assert.Error(t, s.blacklistAccessToken(context.Background(), token))
}
