package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_ProducesValidArgon2idHash(t *testing.T) {
	hash, err := HashPassword("CorrectHorseBattery1")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(hash, "$argon2id$"), "expected argon2id prefix")
	parts := strings.Split(hash, "$")
	assert.Len(t, parts, 6, "encoded hash must have 6 segments")
	assert.Equal(t, "argon2id", parts[1])
}

func TestHashPassword_DifferentSaltsProduceDifferentHashes(t *testing.T) {
	pw := "RepeatablePassword1"
	a, err := HashPassword(pw)
	require.NoError(t, err)
	b, err := HashPassword(pw)
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "salt randomness must produce distinct hashes")
}

func TestVerifyPassword(t *testing.T) {
	pw := "Tr0ub4dor"
	hash, err := HashPassword(pw)
	require.NoError(t, err)

	t.Run("correct password verifies", func(t *testing.T) {
		assert.True(t, VerifyPassword(hash, pw))
	})
	t.Run("wrong password rejected", func(t *testing.T) {
		assert.False(t, VerifyPassword(hash, "WrongPass1"))
	})
	t.Run("empty password rejected", func(t *testing.T) {
		assert.False(t, VerifyPassword(hash, ""))
	})
	t.Run("malformed hash rejected", func(t *testing.T) {
		assert.False(t, VerifyPassword("not-a-hash", pw))
		assert.False(t, VerifyPassword("$argon2id$v=19$short$bad$bad", pw))
		assert.False(t, VerifyPassword("$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA", pw),
			"non-argon2id algorithms must be rejected")
	})
	t.Run("empty hash rejected", func(t *testing.T) {
		assert.False(t, VerifyPassword("", pw))
	})
}

func TestVerifyDummyPassword_AlwaysFalse(t *testing.T) {
	// must always return false; goal is timing parity, not validity
	for _, pw := range []string{"", "short", "AnyPassword123!", strings.Repeat("a", 200)} {
		assert.False(t, VerifyDummyPassword(pw),
			"dummy verify must return false for %q", pw)
	}
}

func TestValidatePasswordPolicy(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 8-char", "Aa345678", false},
		{"valid long", "MyS3cretPassphrase", false},
		{"too short", "Aa1bcd", true},
		{"missing uppercase", "abcdefgh1", true},
		{"missing lowercase", "ABCDEFGH1", true},
		{"missing digit", "Abcdefgh", true},
		{"only digits", "12345678", true},
		{"empty", "", true},
		{"unicode letters", "Çğşİö123", true}, // unicode upper/lower not counted by ASCII rule
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordPolicy(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrWeakPassword)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecodeHash_RejectsCorruptInputs(t *testing.T) {
	_, _, _, err := decodeHash("$argon2id$v=99$m=65536,t=3,p=4$c2FsdA$aGFzaA")
	assert.ErrorIs(t, err, ErrIncompatibleVersion)

	_, _, _, err = decodeHash("$argon2id$v=19$m=bad,t=3,p=4$c2FsdA$aGFzaA")
	assert.Error(t, err)

	_, _, _, err = decodeHash("$argon2id$v=19$m=65536,t=3,p=4$!!!$aGFzaA")
	assert.Error(t, err, "invalid base64 salt must error")
}

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword("BenchPassword1")
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	hash, _ := HashPassword("BenchPassword1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(hash, "BenchPassword1")
	}
}
