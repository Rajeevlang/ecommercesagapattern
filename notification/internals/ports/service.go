package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/domain"
)

type NotificationService interface {
	SendEmail(ctx context.Context, userID, recipient, subject, body, templateName, idempotencyKey string) (*domain.Notification, error)
	SendSMS(ctx context.Context, userID, recipient, message, idempotencyKey string) (*domain.Notification, error)
	GetNotificationStatus(ctx context.Context, notificationID string) (*domain.Notification, error)
}
