package consumer

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

func (c *Consumer) dispatch(ctx context.Context, eventID, eventType string, event map[string]any) error {
	data, ok := event["data"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing or invalid 'data' payload")
	}

	switch eventType {
	case "user.registered":
		c.logDelivery(ctx, eventID, eventType, "email", fmt.Sprint(data["email"]), "welcome", "pending", nil)
		err := c.svc.SendWelcomeEmail(ctx, data)
		if err != nil {
			c.logDelivery(ctx, eventID, eventType, "email", fmt.Sprint(data["email"]), "welcome", "failed", err)
			return err
		}
		c.logDelivery(ctx, eventID, eventType, "email", fmt.Sprint(data["email"]), "welcome", "sent", nil)

	case "user.password_reset_requested":
		c.logDelivery(ctx, eventID, eventType, "email", fmt.Sprint(data["email"]), "password_reset", "pending", nil)
		err := c.svc.SendPasswordResetEmail(ctx, data)
		if err != nil {
			c.logDelivery(ctx, eventID, eventType, "email", fmt.Sprint(data["email"]), "password_reset", "failed", err)
			return err
		}
		c.logDelivery(ctx, eventID, eventType, "email", fmt.Sprint(data["email"]), "password_reset", "sent", nil)

	// -------------------------------------------------------------
	// FUTURE NOTIFICATION EVENTS (Skeleton implementation)
	// -------------------------------------------------------------
	case "grades.entered":
		// Standard notification: Send Push only (Fire & Forget)
		userID := fmt.Sprint(data["user_id"])
		title := "Vize Notunuz Girildi!"
		message := "Bilgisayar Bilimine Giriş dersi için vize notunuz sisteme girilmiştir."
		_ = c.svc.SendStandardNotification(ctx, userID, title, message)

	case "student.enrolled", "student.graduated", "grade.appeal_accepted":
		// Important notification: Send Email + Push
		userID := fmt.Sprint(data["user_id"])
		emailAddr := fmt.Sprint(data["email"])
		
		var title, message string
		if eventType == "student.enrolled" {
			title = "Okula Kaydınız Yapıldı"
			message = "Tebrikler, üniversitemize kaydınız başarıyla tamamlanmıştır."
		} else if eventType == "student.graduated" {
			title = "Mezuniyetiniz Tamamlandı!"
			message = "Tebrikler, tüm zorunlulukları yerine getirdiniz ve mezun oldunuz!"
		} else {
			title = "Not İtirazınız Kabul Edildi"
			message = "Geçen haftaki itirazınız değerlendirildi ve kabul edildi."
		}

		_ = c.svc.SendImportantNotification(ctx, userID, emailAddr, title, message)

	default:
		c.log.Info("unhandled event type", zap.String("event_type", eventType))
	}

	return nil
}
