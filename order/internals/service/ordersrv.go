package service

import (
	"context"
	"fmt"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
	"github.com/Rajeevlang/ecommercesagapattern/order/internals/ports"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
	paymentv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/payment/v1"
)

// OrderSrv implements the ports.Service interface and manages Saga Orchestration.
type OrderSrv struct {
	repo          ports.OrderRepository
	catalogClient catalogv1.CatalogServiceClient
	paymentClient paymentv1.PaymentServiceClient
	accountClient accountv1.AccountServiceClient
	publisher     ports.EventPublisher
}

// NewOrderService constructs a new OrderSrv injected with repository, external gRPC clients, and event publisher.
func NewOrderService(
	repo ports.OrderRepository,
	catalogClient catalogv1.CatalogServiceClient,
	paymentClient paymentv1.PaymentServiceClient,
	accountClient accountv1.AccountServiceClient,
	publisher     ports.EventPublisher,
) *OrderSrv {
	return &OrderSrv{
		repo:          repo,
		catalogClient: catalogClient,
		paymentClient: paymentClient,
		accountClient: accountClient,
		publisher:     publisher,
	}
}

// CreateOrder initiates the order in local database and orchestrates the Saga workflow.
func (s *OrderSrv) CreateOrder(ctx context.Context, order *domain.Order) error {
	// 1. Persist the order in initial state ORDER_STATUS_PENDING
	order.Status = domain.StatusPending
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to save pending order: %w", err)
	}

	// 2. Saga Step 1: Reserve Stock in Catalog Microservice
	var reserveItems []*catalogv1.ReserveItem
	for _, item := range order.Items {
		reserveItems = append(reserveItems, &catalogv1.ReserveItem{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	reserveResp, err := s.catalogClient.ReserveStock(ctx, &catalogv1.ReserveStockRequest{
		OrderId: order.ID,
		Items:   reserveItems,
	})
	if err != nil || !reserveResp.GetSuccess() {
		errMsg := "Catalog service offline"
		if reserveResp != nil {
			errMsg = reserveResp.GetMessage()
		}
		
		// Compensating Action: Release stock just in case the reservation was committed before network failure
		_, _ = s.catalogClient.ReleaseStock(ctx, &catalogv1.ReleaseStockRequest{
			OrderId: order.ID,
			Items:   reserveItems,
		})

		// Saga Local Failure: Mark order as failed in local DB
		notes := fmt.Sprintf("Stock reservation failed: %s", errMsg)
		_ = s.repo.UpdateOrderStatus(ctx, order.ID, domain.StatusFailed, notes)
		order.Status = domain.StatusFailed

		// Publish failure event to Kafka
		email := s.getUserEmail(ctx, order.UserID)
		if s.publisher != nil {
			_ = s.publisher.PublishOrderFailed(ctx, order, email, notes)
		}

		return fmt.Errorf("saga step 1 (reserve stock) failed: %s", errMsg)
	}

	// 3. Saga Step 2: Process Payment in Payment Microservice
	paymentResp, err := s.paymentClient.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		OrderId:            order.ID,
		UserId:             order.UserID,
		AmountCents:        order.TotalAmountCents,
		PaymentMethodToken: order.PaymentMethodToken,
	})
	if err != nil || !paymentResp.GetSuccess() {
		// Saga Step 2 Failure: Process compensation rollbacks
		errMsg := "Payment service offline"
		paymentID := ""
		if paymentResp != nil {
			errMsg = paymentResp.GetErrorMessage()
			paymentID = paymentResp.GetPaymentId()
		}

		// Compensating Action 1: Release Stock in Catalog Microservice
		var releaseItems []*catalogv1.ReserveItem
		for _, item := range order.Items {
			releaseItems = append(releaseItems, &catalogv1.ReserveItem{
				ProductId: item.ProductID,
				Quantity:  item.Quantity,
			})
		}
		_, _ = s.catalogClient.ReleaseStock(ctx, &catalogv1.ReleaseStockRequest{
			OrderId: order.ID,
			Items:   releaseItems,
		})

		// Compensating Action 2: Void or refund the payment just in case it went through
		_, _ = s.paymentClient.CancelPayment(ctx, &paymentv1.CancelPaymentRequest{
			OrderId:   order.ID,
			PaymentId: paymentID,
			Reason:    fmt.Sprintf("Payment failure rollback: %s", errMsg),
		})

		// Saga Local Failure: Mark order as failed in local DB
		notes := fmt.Sprintf("Payment failed: %s. Stock released.", errMsg)
		_ = s.repo.UpdateOrderStatus(ctx, order.ID, domain.StatusFailed, notes)
		order.Status = domain.StatusFailed

		// Publish failure event to Kafka
		email := s.getUserEmail(ctx, order.UserID)
		if s.publisher != nil {
			_ = s.publisher.PublishOrderFailed(ctx, order, email, notes)
		}

		return fmt.Errorf("saga step 2 (process payment) failed: %s", errMsg)
	}

	// 4. Saga Success Path: Finalize Order State to COMPLETED
	notes := fmt.Sprintf("Payment authorized with ID: %s", paymentResp.GetPaymentId())
	if err := s.repo.UpdateOrderStatus(ctx, order.ID, domain.StatusCompleted, notes); err != nil {
		return fmt.Errorf("failed to finalize order status: %w", err)
	}
	order.Status = domain.StatusCompleted

	// Publish success event to Kafka
	email := s.getUserEmail(ctx, order.UserID)
	if s.publisher != nil {
		_ = s.publisher.PublishOrderCompleted(ctx, order, email)
	}

	return nil
}

func (s *OrderSrv) getUserEmail(ctx context.Context, userID string) string {
	if s.accountClient == nil {
		return "customer@example.com"
	}
	profile, err := s.accountClient.GetProfile(ctx, &accountv1.GetProfileRequest{UserId: userID})
	if err != nil {
		return "customer@example.com"
	}
	return profile.GetEmail()
}

// UpdateOrderStatus modifies order status locally.
func (s *OrderSrv) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus, notes string) error {
	return s.repo.UpdateOrderStatus(ctx, orderID, status, notes)
}

// GetOrderDetails fetches the order details by ID.
func (s *OrderSrv) GetOrderDetails(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.repo.GetOrderDetails(ctx, orderID)
}
