package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/ports"
)

type PaymentService struct {
	repo ports.PaymentRepository
}

func NewPaymentService(repo ports.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ProcessPayment(
	ctx context.Context,
	orderID, userID string,
	amountCents int64,
	token string,
) (*domain.Payment, error) {
	if amountCents <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	// 1. Check idempotency: If payment for this order already exists, return it
	existing, err := s.repo.GetPaymentByOrderID(ctx, orderID)
	if err == nil {
		log.Printf("ProcessPayment: returning existing payment record for order %s (status: %s)\n", orderID, existing.Status)
		return existing, nil
	}
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		return nil, fmt.Errorf("failed to query payment idempotency: %w", err)
	}

	// 2. Simulate payment processing gateway
	status := domain.StatusAuthorized
	errMsg := ""

	// Simple simulation rules:
	// If token contains "fail", "decline", or order ID contains "decline", fail the transaction
	normalizedToken := strings.ToLower(token)
	if strings.Contains(normalizedToken, "fail") || strings.Contains(normalizedToken, "decline") || strings.Contains(strings.ToLower(orderID), "decline") {
		status = domain.StatusDeclined
		errMsg = "Card declined by issuing bank (insufficient funds)"
		log.Printf("ProcessPayment: Simulated payment decline for order %s\n", orderID)
	} else {
		log.Printf("ProcessPayment: Simulated payment authorization success for order %s\n", orderID)
	}

	// 3. Persist the transaction log
	p := &domain.Payment{
		OrderID:            orderID,
		UserID:             userID,
		AmountCents:        amountCents,
		PaymentMethodToken: token,
		Status:             status,
		ErrorMessage:       errMsg,
	}

	if err := s.repo.CreatePayment(ctx, p); err != nil {
		if errors.Is(err, domain.ErrDuplicatePayment) {
			// Concurrent insert safety
			return s.repo.GetPaymentByOrderID(ctx, orderID)
		}
		return nil, fmt.Errorf("failed to save payment record: %w", err)
	}

	return p, nil
}

func (s *PaymentService) CancelPayment(ctx context.Context, orderID, paymentID, reason string) error {
	log.Printf("CancelPayment: compensaton triggered for order %s, payment %s (reason: %s)\n", orderID, paymentID, reason)

	var p *domain.Payment
	var err error

	// 1. Try to find the payment record
	if orderID != "" {
		p, err = s.repo.GetPaymentByOrderID(ctx, orderID)
	} else if paymentID != "" {
		p, err = s.repo.GetPayment(ctx, paymentID)
	} else {
		return errors.New("must provide order_id or payment_id to cancel payment")
	}

	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			// Idempotency: If payment was never processed, return nil (success/noop)
			log.Printf("CancelPayment: payment record not found. Compensating action is idempotent (noop).\n")
			return nil
		}
		return fmt.Errorf("failed to fetch payment for cancellation: %w", err)
	}

	// 2. Handle state transitions
	if p.Status == domain.StatusDeclined {
		log.Printf("CancelPayment: payment was already declined, no refund needed.\n")
		return nil
	}
	if p.Status == domain.StatusRefunded {
		log.Printf("CancelPayment: payment was already refunded/cancelled.\n")
		return nil
	}

	// 3. Process refund (Status transitions from AUTHORIZED to REFUNDED)
	log.Printf("CancelPayment: refunding payment %s for order %s...\n", p.ID, p.OrderID)
	err = s.repo.UpdatePaymentStatus(ctx, p.ID, domain.StatusRefunded, fmt.Sprintf("Refunded: %s", reason))
	if err != nil {
		return fmt.Errorf("failed to execute payment refund: %w", err)
	}

	return nil
}
