package email

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
)

type MockEmailProvider struct{}

func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{}
}

func (p *MockEmailProvider) SendEmail(ctx context.Context, recipient, subject, body string) error {
	log.Printf("[MOCK EMAIL] To: %s | Subject: %s\nBody:\n%s\n", recipient, subject, body)
	return nil
}

type SMTPEmailProvider struct {
	host string
	port string
	from string
	auth smtp.Auth
}

func NewSMTPEmailProvider(host, port, from, username, password string) *SMTPEmailProvider {
	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	return &SMTPEmailProvider{
		host: host,
		port: port,
		from: from,
		auth: auth,
	}
}

func (p *SMTPEmailProvider) SendEmail(ctx context.Context, recipient, subject, body string) error {
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", recipient, subject, body))
	addr := fmt.Sprintf("%s:%s", p.host, p.port)
	err := smtp.SendMail(addr, p.auth, p.from, []string{recipient}, msg)
	if err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}
	return nil
}
