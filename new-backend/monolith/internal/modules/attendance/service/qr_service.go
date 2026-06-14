package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
)

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

// GenerateQRPayload generates a static QR payload for the session
// The QR code stays the same for the entire session duration
func (s *QRService) GenerateQRPayload(sessionID, secret string) dto.QRPayload {
	signature := s.calculateHMAC(secret, sessionID)

	return dto.QRPayload{
		SessionID: sessionID,
		Signature: signature,
	}
}

// ValidateQRSignature validates QR signature against session secret
func (s *QRService) ValidateQRSignature(payload dto.QRPayload, secret string) bool {
	expected := s.calculateHMAC(secret, payload.SessionID)
	return hmac.Equal([]byte(expected), []byte(payload.Signature))
}

func (s *QRService) calculateHMAC(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
