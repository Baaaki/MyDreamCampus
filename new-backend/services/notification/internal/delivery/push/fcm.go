package push

import (
	"context"
	"go.uber.org/zap"
)

// FCMSender is a dummy/template struct for Firebase Cloud Messaging.
// This is not active yet, but acts as a placeholder for future mobile app integration.
type FCMSender struct {
	log *zap.Logger
	// firebaseClient *messaging.Client (To be added later when Firebase Admin SDK is installed)
}

func New(log *zap.Logger) *FCMSender {
	return &FCMSender{
		log: log,
	}
}

// Send is the skeleton method that will send Push Notifications to mobile devices.
func (f *FCMSender) Send(ctx context.Context, userID, title, body string) error {
	// TODO: When mobile app is ready:
	// 1. Fetch user's FCM tokens from database (user_devices table).
	// 2. Build the firebase messaging payload.
	// 3. f.firebaseClient.SendMulticast(ctx, message)

	f.log.Info("MOCK PUSH NOTIFICATION SENT (Template)",
		zap.String("user_id", userID),
		zap.String("title", title),
		zap.String("body", body),
	)

	return nil
}
