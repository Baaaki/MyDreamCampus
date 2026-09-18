package service

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

// qrNow is a fixed instant well inside a window, so "+rotation" and
// "-rotation" land in the neighbouring windows.
var qrNow = time.Date(2026, 3, 2, 10, 0, 5, 0, time.UTC)

func gen(s *QRService, sessionID, secret string, at time.Time) dto.QRPayload {
	return s.GenerateQRPayload(sessionID, secret, at, DefaultQRRotation)
}

func valid(s *QRService, p dto.QRPayload, secret string, at time.Time) bool {
	return s.ValidateQRSignature(p, secret, at, DefaultQRRotation)
}

func TestQRService_GenerateQRPayload(t *testing.T) {
	s := NewQRService()

	t.Run("payload carries session id and current window", func(t *testing.T) {
		payload := gen(s, "session-123", "secret-abc", qrNow)
		assert.Equal(t, "session-123", payload.SessionID)
		assert.Equal(t, QRWindow(qrNow, DefaultQRRotation), payload.Window)
		assert.NotEmpty(t, payload.Signature)
	})

	t.Run("signature is stable within a window", func(t *testing.T) {
		a := gen(s, "session-x", "secret-x", qrNow)
		b := gen(s, "session-x", "secret-x", qrNow.Add(time.Second))
		assert.Equal(t, a.Signature, b.Signature)
	})

	t.Run("signature rotates with the window", func(t *testing.T) {
		a := gen(s, "session-x", "secret-x", qrNow)
		b := gen(s, "session-x", "secret-x", qrNow.Add(DefaultQRRotation))
		assert.NotEqual(t, a.Signature, b.Signature)
	})

	t.Run("signature differs when secret differs", func(t *testing.T) {
		a := gen(s, "same-session", "secret-1", qrNow)
		b := gen(s, "same-session", "secret-2", qrNow)
		assert.NotEqual(t, a.Signature, b.Signature)
	})

	t.Run("signature differs when session id differs", func(t *testing.T) {
		a := gen(s, "session-a", "same-secret", qrNow)
		b := gen(s, "session-b", "same-secret", qrNow)
		assert.NotEqual(t, a.Signature, b.Signature)
	})

	t.Run("signature is hex encoded sha256 (64 chars)", func(t *testing.T) {
		payload := gen(s, "s", "k", qrNow)
		assert.Len(t, payload.Signature, 64)
		_, err := hex.DecodeString(payload.Signature)
		assert.NoError(t, err)
	})
}

func TestQRService_ValidateQRSignature(t *testing.T) {
	s := NewQRService()

	t.Run("accepts a code from the current window", func(t *testing.T) {
		secret, err := s.GenerateSecret()
		require.NoError(t, err)
		assert.True(t, valid(s, gen(s, "real-session", secret, qrNow), secret, qrNow))
	})

	t.Run("accepts a code scanned just after rotation", func(t *testing.T) {
		payload := gen(s, "real-session", "secret", qrNow)
		assert.True(t, valid(s, payload, "secret", qrNow.Add(DefaultQRRotation)))
	})

	t.Run("rejects a forwarded code two windows later", func(t *testing.T) {
		payload := gen(s, "real-session", "secret", qrNow)
		assert.False(t, valid(s, payload, "secret", qrNow.Add(2*DefaultQRRotation)),
			"a photo of the board must stop working once its window has passed")
	})

	t.Run("rejects a code from the future", func(t *testing.T) {
		payload := gen(s, "real-session", "secret", qrNow.Add(DefaultQRRotation))
		assert.False(t, valid(s, payload, "secret", qrNow))
	})

	t.Run("rejects a moved window with the old signature", func(t *testing.T) {
		payload := gen(s, "real-session", "secret", qrNow.Add(-2*DefaultQRRotation))
		payload.Window = QRWindow(qrNow, DefaultQRRotation)
		assert.False(t, valid(s, payload, "secret", qrNow))
	})

	t.Run("rejects when secret is wrong", func(t *testing.T) {
		payload := gen(s, "real-session", "secret-good", qrNow)
		assert.False(t, valid(s, payload, "secret-bad", qrNow))
	})

	t.Run("rejects when session id was tampered", func(t *testing.T) {
		payload := gen(s, "real-session", "secret", qrNow)
		payload.SessionID = "evil-session"
		assert.False(t, valid(s, payload, "secret", qrNow))
	})

	t.Run("rejects empty signature", func(t *testing.T) {
		payload := dto.QRPayload{SessionID: "x", Window: QRWindow(qrNow, DefaultQRRotation)}
		assert.False(t, valid(s, payload, "secret", qrNow))
	})

	t.Run("rejects malformed signature without panicking", func(t *testing.T) {
		payload := dto.QRPayload{SessionID: "x", Window: QRWindow(qrNow, DefaultQRRotation), Signature: "not-the-right-length"}
		assert.False(t, valid(s, payload, "secret", qrNow))
	})
}

func TestCanManageSession_OwnerOrAdminOnly(t *testing.T) {
	owner := uuid.New()
	pgOwner := pgtype.UUID{Bytes: owner, Valid: true}

	assert.True(t, canManageSession(pgOwner, owner, false), "owner")
	assert.True(t, canManageSession(pgOwner, uuid.New(), true), "admin on someone else's session")
	assert.False(t, canManageSession(pgOwner, uuid.New(), false), "another teacher")
}
