package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
)

type EventPublisher interface {
	PublishOrderCompleted(ctx context.Context, order *domain.Order, email string) error
	PublishOrderFailed(ctx context.Context, order *domain.Order, email string, reason string) error
}
