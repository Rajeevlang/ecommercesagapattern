package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
)

// OrderRepository defines the database operations contract for the Order service.
type OrderRepository interface {
	// CreateOrder stores a new order and its line items in a database transaction.
	CreateOrder(ctx context.Context, order *domain.Order) error
	
	// UpdateOrderStatus updates the state of an order along with audit notes.
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus, notes string) error
	
	// GetOrderDetails retrieves an order and its items by its UUID.
	GetOrderDetails(ctx context.Context, orderID string) (*domain.Order, error)
}