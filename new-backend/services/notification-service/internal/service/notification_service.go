package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/baaaki/mydreamcampus/notification/config"
	"github.com/baaaki/mydreamcampus/notification/internal/delivery/email"
	"github.com/baaaki/mydreamcampus/notification/internal/delivery/push"
	"github.com/baaaki/mydreamcampus/notification/internal/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo  *repository.Repository
	email *email.SMTPSender
	push  *push.FCMSender
	log   *zap.Logger
	cfg   *config.Config
	tpl   *template.Template
}

func New(repo *repository.Repository, email *email.SMTPSender, push *push.FCMSender, log *zap.Logger, cfg *config.Config) (*Service, error) {
	// Parse all HTML templates from the templates directory
	tpl, err := template.ParseGlob(filepath.Join("internal", "templates", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Service{
		repo:  repo,
		email: email,
		push:  push,
		log:   log,
		cfg:   cfg,
		tpl:   tpl,
	}, nil
}

// SendWelcomeEmail renders and sends the welcome email
func (s *Service) SendWelcomeEmail(ctx context.Context, data map[string]any) error {
	emailAddr, ok := data["email"].(string)
	if !ok || emailAddr == "" {
		return fmt.Errorf("missing or invalid email in payload")
	}
	firstName, _ := data["first_name"].(string)
	role, _ := data["role"].(string)

	var body bytes.Buffer
	err := s.tpl.ExecuteTemplate(&body, "welcome.html", map[string]any{
		"first_name": firstName,
		"role":       role,
		"login_url":  s.cfg.AppURL + "/login",
	})
	if err != nil {
		return fmt.Errorf("failed to render welcome template: %w", err)
	}

	err = s.email.Send(ctx, emailAddr, "MyDreamCampus'a hoş geldin!", body.Bytes())
	if err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	return nil
}

// SendPasswordResetEmail renders and sends the password reset email
func (s *Service) SendPasswordResetEmail(ctx context.Context, data map[string]any) error {
	emailAddr, ok := data["email"].(string)
	if !ok || emailAddr == "" {
		return fmt.Errorf("missing or invalid email in payload")
	}
	resetToken, _ := data["reset_token"].(string)
	expiresAt, _ := data["expires_at"].(string)

	var body bytes.Buffer
	err := s.tpl.ExecuteTemplate(&body, "password_reset.html", map[string]any{
		"expires_at": expiresAt,
		"reset_url":  s.cfg.AppURL + "/reset-password?token=" + resetToken,
	})
	if err != nil {
		return fmt.Errorf("failed to render password reset template: %w", err)
	}

	err = s.email.Send(ctx, emailAddr, "Şifre Sıfırlama Talebi", body.Bytes())
	if err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	return nil
}

// ---------------------------------------------------------
// NOTIFICATION ROUTING TEMPLATES (Not fully active yet)
// ---------------------------------------------------------

// SendImportantNotification sends a persistent, highly-visible notification.
// Routing: Email + Mobile Push
func (s *Service) SendImportantNotification(ctx context.Context, userID, emailAddr, title, message string) error {
	s.log.Info("Sending IMPORTANT notification (Email + Push)", zap.String("title", title))
	
	// 1. Send Mobile Push (Fire & Forget)
	_ = s.push.Send(ctx, userID, title, message)
	
	// 2. Send Email (Persistent)
	// For a real app, you would execute an HTML template here, just like SendWelcomeEmail.
	err := s.email.Send(ctx, emailAddr, title, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send important email: %w", err)
	}

	return nil
}

// SendStandardNotification sends an ephemeral/standard notification.
// Routing: Mobile Push ONLY (Fire & Forget)
func (s *Service) SendStandardNotification(ctx context.Context, userID, title, message string) error {
	s.log.Info("Sending STANDARD notification (Push Only)", zap.String("title", title))
	
	// Only send Push Notification. No email is sent.
	err := s.push.Send(ctx, userID, title, message)
	if err != nil {
		return fmt.Errorf("failed to send standard push: %w", err)
	}

	return nil
}

