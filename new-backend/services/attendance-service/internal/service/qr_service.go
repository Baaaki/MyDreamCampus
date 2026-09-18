package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/dto"
)

// DefaultQRRotation is how long one attendance QR stays current. A code is
// accepted for its own window and the one after it, so a photo of the board
// is useless after at most two windows — short enough that it cannot be
// forwarded to someone outside the room and used later.
const DefaultQRRotation = 15 * time.Second

type QRService struct{}

func NewQRService() *QRService {
	return &QRService{}
}

// GenerateSecret generates a random secret for QR signing
func (s *QRService) GenerateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// QRWindow is the rotation bucket `now` falls into.
func QRWindow(now time.Time, rotation time.Duration) int64 {
	return now.Unix() / int64(rotation/time.Second)
}

// GenerateQRPayload signs the session id together with the current window.
// The signature changes every rotation, so the instructor's screen has to
// keep fetching a fresh code.
func (s *QRService) GenerateQRPayload(sessionID, secret string, now time.Time, rotation time.Duration) dto.QRPayload {
	window := QRWindow(now, rotation)
	return dto.QRPayload{
		SessionID: sessionID,
		Window:    window,
		Signature: s.calculateHMAC(secret, qrMessage(sessionID, window)),
	}
}

// ValidateQRSignature accepts the current window or the previous one, so a
// scan made just after a rotation is not rejected.
func (s *QRService) ValidateQRSignature(payload dto.QRPayload, secret string, now time.Time, rotation time.Duration) bool {
	current := QRWindow(now, rotation)
	if payload.Window != current && payload.Window != current-1 {
		return false
	}
	expected := s.calculateHMAC(secret, qrMessage(payload.SessionID, payload.Window))
	return hmac.Equal([]byte(expected), []byte(payload.Signature))
}

func qrMessage(sessionID string, window int64) string {
	return sessionID + ":" + strconv.FormatInt(window, 10)
}

func (s *QRService) calculateHMAC(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
