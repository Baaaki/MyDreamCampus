package utils

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// ErrPasswordHashBusy means no Argon2id slot came free within the wait, or the
// caller's context ended first. It says nothing about the password itself.
var ErrPasswordHashBusy = errors.New("password hashing is at capacity")

// DefaultPasswordHashConcurrency is the slot count until
// SetPasswordHashConcurrency changes it.
const DefaultPasswordHashConcurrency = 4

// passwordHashWait is how long a caller queues for a slot. A login that cannot
// start within it is better answered "try again" than parked behind a burst
// whose end it cannot see. A variable so tests can shorten it.
var passwordHashWait = 5 * time.Second

// hashSlots bounds the Argon2id computations in flight. Every hash and every
// verify holds DefaultParams.Memory (64 MiB) for its whole run, so this — not
// the request rate — keeps a login burst inside the auth container's memory:
// N slots cost N x 64 MiB however many logins arrive at once.
var hashSlots atomic.Pointer[chan struct{}]

func init() { SetPasswordHashConcurrency(DefaultPasswordHashConcurrency) }

// SetPasswordHashConcurrency resizes the slot pool. Meant for startup; a hash
// already running gives its slot back to the pool it took it from.
func SetPasswordHashConcurrency(n int) {
	slots := make(chan struct{}, max(n, 1))
	hashSlots.Store(&slots)
}

// acquireHashSlot waits for a slot and returns the function that frees it.
func acquireHashSlot(ctx context.Context) (func(), error) {
	slots := *hashSlots.Load()
	timer := time.NewTimer(passwordHashWait)
	defer timer.Stop()
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, nil
	case <-timer.C:
		return nil, ErrPasswordHashBusy
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %w", ErrPasswordHashBusy, ctx.Err())
	}
}

// dummyHash is a real Argon2id hash of an unguessable random string. Use it
// via VerifyDummyPassword to keep login response time uniform when the user
// does not exist, defeating email enumeration via timing side-channel.
//
// Computed on first use rather than in init(): init ran a 64 MB Argon2id
// hash in every binary that imports utils — all ten services — and in every
// test process, although only auth ever needs it. Auth calls
// WarmDummyPassword at startup so no login pays for the first computation.
var dummyHash = sync.OnceValue(func() string {
	// 32 random bytes is unguessable; the value never matches any real
	// password and the resulting hash exercises the full Argon2id work
	// factor that VerifyPassword would otherwise skip on a malformed hash.
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		panic("failed to seed dummy password hash: " + err.Error())
	}
	h, err := hashPassword(base64.RawStdEncoding.EncodeToString(random))
	if err != nil {
		panic("failed to compute dummy password hash: " + err.Error())
	}
	return h
})

// WarmDummyPassword computes the dummy hash ahead of the first login, whose
// response would otherwise be one Argon2id round slower — a timing signal.
func WarmDummyPassword() {
	dummyHash()
}

// VerifyDummyPassword runs the full Argon2id verification against a
// precomputed throwaway hash; it never matches. Use on the "user not found"
// branch of a login flow so the response time is indistinguishable from the
// password-mismatch branch. It queues for a slot like a real verify: if it
// skipped the queue, a burst would answer unknown addresses faster — and a
// busy answer must not leak which branch was taken either.
func VerifyDummyPassword(ctx context.Context, password string) error {
	_, err := VerifyPassword(ctx, dummyHash(), password)
	return err
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

// HashPassword generates an Argon2id hash of the password once a slot is free.
// Returns format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
func HashPassword(ctx context.Context, password string) (string, error) {
	release, err := acquireHashSlot(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return hashPassword(password)
}

// hashPassword does the work without taking a slot. Only the dummy hash calls
// it directly: that runs once, at startup, before any request competes.
func hashPassword(password string) (string, error) {
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

// VerifyPassword reports whether password matches the Argon2id hash, comparing
// in constant time. A malformed hash is a plain false; the error is only ever
// ErrPasswordHashBusy, which is no answer about the password at all.
func VerifyPassword(ctx context.Context, encodedHash, password string) (bool, error) {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, nil
	}

	release, err := acquireHashSlot(ctx)
	if err != nil {
		return false, err
	}
	defer release()

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
	return subtle.ConstantTimeCompare(hash, computedHash) == 1, nil
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
	params.SaltLength = ClampToUint32(len(salt))

	// Decode hash
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	params.KeyLength = ClampToUint32(len(hash))

	return params, salt, hash, nil
}
