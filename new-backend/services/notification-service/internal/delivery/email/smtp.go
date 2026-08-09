package email

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/baaaki/mydreamcampus/notification/config"
	"github.com/jordan-wright/email"
)

type SMTPSender struct {
	cfg config.SMTPConfig
}

func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, to, subject string, htmlBody []byte) error {
	e := email.NewEmail()
	e.From = s.cfg.From
	e.To = []string{to}
	e.Subject = subject
	e.HTML = htmlBody

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	
	// Use plain auth if username is provided
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	return e.Send(addr, auth)
}
