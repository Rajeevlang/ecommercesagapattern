package ports

import (
	"context"
)

type EmailProvider interface {
	SendEmail(ctx context.Context, recipient, subject, body string) error
}

type SMSProvider interface {
	SendSMS(ctx context.Context, recipient, message string) error
}
