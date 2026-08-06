package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/domain"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *domain.Payment) error
	GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	GetPayment(ctx context.Context, id string) (*domain.Payment, error)
	UpdatePaymentStatus(ctx context.Context, id string, status domain.PaymentStatus, errMessage string) error
}
