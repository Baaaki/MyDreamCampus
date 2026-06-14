package service

import (
	"encoding/hex"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQRService_GenerateSecret(t *testing.T) {
	s := NewQRService()

	t.Run("returns 64-char hex string (32 random bytes)", func(t *testing.T) {
		secret, err := s.GenerateSecret()
		require.NoError(t, err)
		assert.Len(t, secret, 64)

		raw, err := hex.DecodeString(secret)
		require.NoError(t, err)
		assert.Len(t, raw, 32)
	})

	t.Run("returns different value on each call", func(t *testing.T) {
		seen := make(map[string]struct{}, 32)
		for i := 0; i < 32; i++ {
			secret, err := s.GenerateSecret()
			require.NoError(t, err)
			_, dup := seen[secret]
			assert.False(t, dup, "GenerateSecret produced duplicate at iter %d", i)
			seen[secret] = struct{}{}
		}
	})
}

func TestQRService_GenerateQRPayload(t *testing.T) {
	s := NewQRService()

	t.Run("payload carries session id verbatim", func(t *testing.T) {
		payload := s.GenerateQRPayload("session-123", "secret-abc")
		assert.Equal(t, "session-123", payload.SessionID)
		assert.NotEmpty(t, payload.Signature)
	})

	t.Run("signature is deterministic for same inputs", func(t *testing.T) {
		a := s.GenerateQRPayload("session-x", "secret-x")
		b := s.GenerateQRPayload("session-x", "secret-x")
		assert.Equal(t, a.Signature, b.Signature)
	})

	t.Run("signature differs when secret differs", func(t *testing.T) {
		a := s.GenerateQRPayload("same-session", "secret-1")
		b := s.GenerateQRPayload("same-session", "secret-2")
		assert.NotEqual(t, a.Signature, b.Signature)
	})

	t.Run("signature differs when session id differs", func(t *testing.T) {
		a := s.GenerateQRPayload("session-a", "same-secret")
		b := s.GenerateQRPayload("session-b", "same-secret")
		assert.NotEqual(t, a.Signature, b.Signature)
	})

	t.Run("signature is hex encoded sha256 (64 chars)", func(t *testing.T) {
		payload := s.GenerateQRPayload("s", "k")
		assert.Len(t, payload.Signature, 64)
		_, err := hex.DecodeString(payload.Signature)
		assert.NoError(t, err)
	})
}

func TestQRService_ValidateQRSignature(t *testing.T) {
	s := NewQRService()

	t.Run("accepts a valid round-tripped payload", func(t *testing.T) {
		secret, err := s.GenerateSecret()
		require.NoError(t, err)
		payload := s.GenerateQRPayload("real-session", secret)

		assert.True(t, s.ValidateQRSignature(payload, secret))
	})

	t.Run("rejects when secret is wrong", func(t *testing.T) {
		payload := s.GenerateQRPayload("real-session", "secret-good")
		assert.False(t, s.ValidateQRSignature(payload, "secret-bad"))
	})

	t.Run("rejects when session id was tampered", func(t *testing.T) {
		payload := s.GenerateQRPayload("real-session", "secret")
		tampered := dto.QRPayload{
			SessionID: "evil-session",
			Signature: payload.Signature,
		}
		assert.False(t, s.ValidateQRSignature(tampered, "secret"))
	})

	t.Run("rejects empty signature", func(t *testing.T) {
		payload := dto.QRPayload{SessionID: "x", Signature: ""}
		assert.False(t, s.ValidateQRSignature(payload, "secret"))
	})

	t.Run("rejects malformed signature without panicking", func(t *testing.T) {
		payload := dto.QRPayload{SessionID: "x", Signature: "not-the-right-length"}
		assert.False(t, s.ValidateQRSignature(payload, "secret"))
	})
}
