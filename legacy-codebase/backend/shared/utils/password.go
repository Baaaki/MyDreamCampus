package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("invalid hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

// Argon2Params represents Argon2id hashing parameters
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams are OWASP recommended parameters for Argon2id
var DefaultParams = &Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// ErrWeakPassword is returned when a password fails policy checks.
var ErrWeakPassword = errors.New("password does not meet policy: min 8 chars, at least one uppercase, one lowercase, one digit")

// dummyHash is a real Argon2id hash of an unguessable random string,
// computed once at startup. Use it via VerifyDummyPassword to keep
// login response time uniform when the user does not exist, defeating
// email enumeration via timing side-channel.
var dummyHash string

func init() {
	// 32 random bytes is unguessable; the value never matches any real
	// password and the resulting hash exercises the full Argon2id work
	// factor that VerifyPassword would otherwise skip on a malformed hash.
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		panic("failed to seed dummy password hash: " + err.Error())
	}
	h, err := HashPassword(base64.RawStdEncoding.EncodeToString(random))
	if err != nil {
		panic("failed to compute dummy password hash: " + err.Error())
	}
	dummyHash = h
}

// VerifyDummyPassword runs the full Argon2id verification against a
// precomputed throwaway hash. Always returns false. Use on the
// "user not found" branch of a login flow so the response time is
// indistinguishable from the password-mismatch branch.
func VerifyDummyPassword(password string) bool {
	return VerifyPassword(dummyHash, password)
}

// ValidatePasswordPolicy enforces the password policy used across the system.
// Frontend mirrors this rule; keep them in sync if you change it.
func ValidatePasswordPolicy(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}
	return nil
}

// HashPassword generates an Argon2id hash of the password
// Returns format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, DefaultParams.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		DefaultParams.Iterations,
		DefaultParams.Memory,
		DefaultParams.Parallelism,
		DefaultParams.KeyLength,
	)

	// Encode to base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		DefaultParams.Memory,
		DefaultParams.Iterations,
		DefaultParams.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// VerifyPassword compares a password with an Argon2id hash
// Uses constant-time comparison to prevent timing attacks
func VerifyPassword(encodedHash, password string) bool {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false
	}

	// Compute hash of the provided password
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Constant-time comparison (prevents timing attacks)
	return subtle.ConstantTimeCompare(hash, computedHash) == 1
}

// decodeHash parses an encoded Argon2id hash
func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	// Check algorithm
	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	// Parse version
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	// Parse parameters
	params := &Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	params.SaltLength = uint32(len(salt))

	// Decode hash
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}
