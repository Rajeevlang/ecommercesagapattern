package domain

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "ORDER_STATUS_PENDING"
	StatusCompleted OrderStatus = "ORDER_STATUS_COMPLETED"
	StatusFailed    OrderStatus = "ORDER_STATUS_FAILED"
	StatusCancelled OrderStatus = "ORDER_STATUS_CANCELLED"
)

type Order struct {
	ID                 string
	UserID             string
	TotalAmountCents   int64
	Status             OrderStatus
	Notes              string
	PaymentMethodToken string
	Items              []OrderItem
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type OrderItem struct {
	ID         string
	OrderID    string
	ProductID  string
	Quantity   int32
	PriceCents int64
}
