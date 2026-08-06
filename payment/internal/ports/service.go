package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/domain"
)

type PaymentService interface {
	ProcessPayment(ctx context.Context, orderID, userID string, amountCents int64, token string) (*domain.Payment, error)
	CancelPayment(ctx context.Context, orderID, paymentID, reason string) error
}
