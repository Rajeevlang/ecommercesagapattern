package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
)

// Service defines the business logic and Saga Orchestration contract for the Order service.
type Service interface {
	// CreateOrder initiates the order creation and triggers the Saga workflow.
	CreateOrder(ctx context.Context, order *domain.Order) error
	
	// UpdateOrderStatus modifies order status locally (called during Saga transitions).
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus, notes string) error
	
	// GetOrderDetails fetches the order detail aggregation for GraphQL/gRPC queries.
	GetOrderDetails(ctx context.Context, orderID string) (*domain.Order, error)
}
