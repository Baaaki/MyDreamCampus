package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// NOTE: JWT Generation should ONLY be used by Auth Service
// Other services should validate tokens using middleware, not generate them
// This is a utility function to avoid code duplication, not a design decision

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents JWT claims structure
type Claims struct {
	UserID       string `json:"user_id"`
	Role         string `json:"role"`
	Department   string `json:"department,omitempty"`
	TokenVersion int    `json:"token_version"`
	TokenType    string `json:"token_type"`
	JTI          string `json:"jti,omitempty"` // JWT ID for refresh tokens
	jwt.RegisteredClaims
}

// GetJWTSecret returns JWT secret from environment variable
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set")
	}
	return []byte(secret)
}

// GenerateAccessToken creates a 15-minute access token
// Should ONLY be called by Auth Service
func GenerateAccessToken(userID, role, department string, tokenVersion int) (string, error) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	claims := &Claims{
		UserID:       userID,
		Role:         role,
		Department:   department,
		TokenVersion: tokenVersion,
		TokenType:    string(AccessToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}

// GenerateRefreshToken creates a 24-hour refresh token with unique JTI
// Should ONLY be called by Auth Service
// Returns: (tokenString, jti, error)
func GenerateRefreshToken(userID string, tokenVersion int) (string, string, error) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
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
	tokenString, err := token.SignedString(GetJWTSecret())
	return tokenString, jti, err
}

// ValidateToken validates and parses a JWT token
// Used by all services in their middleware
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return GetJWTSecret(), nil
	})

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

// ValidateTokenIgnoreExpiry validates token signature but ignores expiration
// Used for logout operations where expired tokens should still be accepted
func ValidateTokenIgnoreExpiry(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return GetJWTSecret(), nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
