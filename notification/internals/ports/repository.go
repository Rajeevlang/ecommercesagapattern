package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/domain"
)

type NotificationRepository interface {
	CreateNotification(ctx context.Context, notification *domain.Notification) error
	GetNotification(ctx context.Context, id string) (*domain.Notification, error)
	GetNotificationByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Notification, error)
	UpdateNotification(ctx context.Context, notification *domain.Notification) error
	ListPendingOrRetrying(ctx context.Context) ([]*domain.Notification, error)
}
