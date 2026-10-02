package utils

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_ProducesValidArgon2idHash(t *testing.T) {
	hash, err := HashPassword(context.Background(), "CorrectHorseBattery1")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(hash, "$argon2id$"), "expected argon2id prefix")
	parts := strings.Split(hash, "$")
	assert.Len(t, parts, 6, "encoded hash must have 6 segments")
	assert.Equal(t, "argon2id", parts[1])
}

func TestHashPassword_DifferentSaltsProduceDifferentHashes(t *testing.T) {
	pw := "RepeatablePassword1"
	a, err := HashPassword(context.Background(), pw)
	require.NoError(t, err)
	b, err := HashPassword(context.Background(), pw)
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "salt randomness must produce distinct hashes")
}

func TestVerifyPassword(t *testing.T) {
	pw := "Tr0ub4dor"
	hash, err := HashPassword(context.Background(), pw)
	require.NoError(t, err)
	verify := func(encoded, candidate string) bool {
		ok, err := VerifyPassword(context.Background(), encoded, candidate)
		require.NoError(t, err)
		return ok
	}

	t.Run("correct password verifies", func(t *testing.T) {
		assert.True(t, verify(hash, pw))
	})
	t.Run("wrong password rejected", func(t *testing.T) {
		assert.False(t, verify(hash, "WrongPass1"))
	})
	t.Run("empty password rejected", func(t *testing.T) {
		assert.False(t, verify(hash, ""))
	})
	t.Run("malformed hash rejected", func(t *testing.T) {
		assert.False(t, verify("not-a-hash", pw))
		assert.False(t, verify("$argon2id$v=19$short$bad$bad", pw))
		assert.False(t, verify("$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA", pw),
			"non-argon2id algorithms must be rejected")
	})
	t.Run("empty hash rejected", func(t *testing.T) {
		assert.False(t, verify("", pw))
	})
}

func TestVerifyDummyPassword_NeverMatches(t *testing.T) {
	// The dummy exists for timing parity, so it must never accept anything.
	for _, pw := range []string{"", "short", "AnyPassword123!", strings.Repeat("a", 200)} {
		require.NoError(t, VerifyDummyPassword(context.Background(), pw))
		ok, err := VerifyPassword(context.Background(), dummyHash(), pw)
		require.NoError(t, err)
		assert.False(t, ok, "the dummy hash must not match %q", pw)
	}
}

// withHashSlots resizes the slot pool and the wait for one test.
func withHashSlots(t *testing.T, n int, wait time.Duration) {
	t.Helper()
	previousWait := passwordHashWait
	SetPasswordHashConcurrency(n)
	passwordHashWait = wait
	t.Cleanup(func() {
		SetPasswordHashConcurrency(DefaultPasswordHashConcurrency)
		passwordHashWait = previousWait
	})
}

// holdSlot takes a slot the way a running hash would and keeps it until the
// test ends.
func holdSlot(t *testing.T) {
	t.Helper()
	release, err := acquireHashSlot(context.Background())
	require.NoError(t, err)
	t.Cleanup(release)
}

func TestHashSlots_NeverMoreInFlightThanTheLimit(t *testing.T) {
	withHashSlots(t, 2, 5*time.Second)

	var inFlight, peak atomic.Int32
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			release, err := acquireHashSlot(context.Background())
			if !assert.NoError(t, err) {
				return
			}
			defer release()
			now := inFlight.Add(1)
			for {
				seen := peak.Load()
				if now <= seen || peak.CompareAndSwap(seen, now) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			inFlight.Add(-1)
		})
	}
	wg.Wait()

	assert.Equal(t, int32(2), peak.Load(), "the queue must fill both slots and never a third")
}

func TestHashSlots_BusyOnceTheWaitRunsOut(t *testing.T) {
	withHashSlots(t, 1, 30*time.Millisecond)
	hash, err := HashPassword(context.Background(), "Sifre-12345")
	require.NoError(t, err)
	holdSlot(t)

	_, err = HashPassword(context.Background(), "Sifre-12345")
	assert.ErrorIs(t, err, ErrPasswordHashBusy)
	_, err = VerifyPassword(context.Background(), hash, "Sifre-12345")
	assert.ErrorIs(t, err, ErrPasswordHashBusy)
	assert.ErrorIs(t, VerifyDummyPassword(context.Background(), "Sifre-12345"), ErrPasswordHashBusy,
		"an unknown address must get the same busy answer as a known one")
}

func TestHashSlots_StopWaitingWhenTheCallerDoes(t *testing.T) {
	withHashSlots(t, 1, time.Minute)
	holdSlot(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := HashPassword(ctx, "Sifre-12345")

	assert.ErrorIs(t, err, ErrPasswordHashBusy)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestHashSlots_FreedSlotServesTheNextCaller(t *testing.T) {
	withHashSlots(t, 1, 5*time.Second)
	release, err := acquireHashSlot(context.Background())
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, err := HashPassword(context.Background(), "Sifre-12345")
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	release()

	assert.NoError(t, <-done)
}

func TestVerifyPassword_MalformedHashTakesNoSlot(t *testing.T) {
	withHashSlots(t, 1, time.Minute)
	holdSlot(t)

	ok, err := VerifyPassword(context.Background(), "not-a-hash", "Sifre-12345")

	assert.NoError(t, err)
	assert.False(t, ok)
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
		_, _ = HashPassword(context.Background(), "BenchPassword1")
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	hash, _ := HashPassword(context.Background(), "BenchPassword1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyPassword(context.Background(), hash, "BenchPassword1")
	}
}
