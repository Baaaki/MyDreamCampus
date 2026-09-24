package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// NOTE: token generation belongs to the auth module; everything else
// validates tokens via platform middleware and must not mint them.

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	// ErrWrongTokenType is a validly signed token of the other kind — a
	// refresh token presented where an access token is required.
	ErrWrongTokenType = errors.New("wrong token type")
	jwtSecret         []byte
)

// InitJWTSecret sets the global JWT secret.
func InitJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents JWT claims structure
type Claims struct {
	UserID              string `json:"user_id"`
	Role                string `json:"role"`
	Department          string `json:"department,omitempty"`
	TokenVersion        int    `json:"token_version"`
	TokenType           string `json:"token_type"`
	JTI                 string `json:"jti,omitempty"` // JWT ID for token blacklisting/revocation
	ForcePasswordChange bool   `json:"force_password_change,omitempty"`
	SuperAdmin          bool   `json:"super_admin,omitempty"`
	jwt.RegisteredClaims
}

// GetJWTSecret returns JWT secret from environment variable
// DEPRECATED: Use GenerateAccessTokenWithSecret or ValidateTokenWithSecret instead
// This function is kept for backward compatibility only
func GetJWTSecret() []byte {
	if len(jwtSecret) > 0 {
		return jwtSecret
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set and InitJWTSecret was not called")
	}
	return []byte(secret)
}

// GenerateAccessToken creates a 15-minute access token
// Should ONLY be called by the auth module
// Returns: (tokenString, jti, error)
// DEPRECATED: Use GenerateAccessTokenWithSecret instead for better testability
func GenerateAccessToken(userID, role, department string, tokenVersion int) (string, string, error) {
	return GenerateAccessTokenWithSecret(userID, role, department, tokenVersion, GetJWTSecret(), 15)
}

// GenerateAccessTokenWithSecret creates an access token with custom secret and expiry
// Should ONLY be called by the auth module
// Returns: (tokenString, jti, error)
// expiryMinutes: token expiry time in minutes
func GenerateAccessTokenWithSecret(userID, role, department string, tokenVersion int, secret []byte, expiryMinutes int) (string, string, error) {
	// Real time, not clock.Now: validation checks exp against the real
	// clock, so a simulated issue time would log everyone out (or keep
	// tokens alive for years) the moment the time machine moved.
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiryMinutes) * time.Minute)
	jti := uuid.New().String() // Unique JWT ID for token blacklisting/revocation

	claims := &Claims{
		UserID:       userID,
		Role:         role,
		Department:   department,
		TokenVersion: tokenVersion,
		TokenType:    string(AccessToken),
		JTI:          jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	return tokenString, jti, err
}

// GenerateRefreshToken creates a 24-hour refresh token with unique JTI
// Should ONLY be called by the auth module
// Returns: (tokenString, jti, error)
// DEPRECATED: Use GenerateRefreshTokenWithSecret instead for better testability
func GenerateRefreshToken(userID string, tokenVersion int) (string, string, error) {
	return GenerateRefreshTokenWithSecret(userID, tokenVersion, GetJWTSecret(), 24)
}

// GenerateRefreshTokenWithSecret creates a refresh token with custom secret and expiry
// Should ONLY be called by the auth module
// Returns: (tokenString, jti, error)
// expiryHours: token expiry time in hours
func GenerateRefreshTokenWithSecret(userID string, tokenVersion int, secret []byte, expiryHours int) (string, string, error) {
	now := time.Now() // real time: see GenerateAccessTokenWithSecret
	expiresAt := now.Add(time.Duration(expiryHours) * time.Hour)
	jti := uuid.New().String() // Unique JWT ID for session tracking

	claims := &Claims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		TokenType:    string(RefreshToken),
		JTI:          jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	return tokenString, jti, err
}

// ValidateToken validates and parses a JWT token
// Used by all services in their middleware
// DEPRECATED: Use ValidateTokenWithSecret instead for better testability
func ValidateToken(tokenString string) (*Claims, error) {
	return ValidateTokenWithSecret(tokenString, GetJWTSecret())
}

// ValidateTokenWithSecret validates and parses a JWT token with custom secret
// Used by all services in their middleware
func ValidateTokenWithSecret(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAccessToken is what request authentication must use: on top of
// ValidateToken it requires token_type=access. Both kinds share the signing
// key, so without this a 24-hour refresh token — which carries no role —
// passes as a 15-minute access token on every route that checks no role.
func ValidateAccessToken(tokenString string) (*Claims, error) {
	return ValidateAccessTokenWithSecret(tokenString, GetJWTSecret())
}

// ValidateAccessTokenWithSecret is ValidateAccessToken with an explicit key.
func ValidateAccessTokenWithSecret(tokenString string, secret []byte) (*Claims, error) {
	claims, err := ValidateTokenWithSecret(tokenString, secret)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != string(AccessToken) {
		return nil, ErrWrongTokenType
	}
	return claims, nil
}

// ValidateTokenIgnoreExpiry validates token signature but ignores expiration
// Used for logout operations where expired tokens should still be accepted
// DEPRECATED: Use ValidateTokenIgnoreExpiryWithSecret instead for better testability
func ValidateTokenIgnoreExpiry(tokenString string) (*Claims, error) {
	return ValidateTokenIgnoreExpiryWithSecret(tokenString, GetJWTSecret())
}

// ValidateTokenIgnoreExpiryWithSecret validates token signature but ignores expiration with custom secret
// Used for logout operations where expired tokens should still be accepted
func ValidateTokenIgnoreExpiryWithSecret(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithoutClaimsValidation(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
