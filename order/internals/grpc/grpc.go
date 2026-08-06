package grpc

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
	"github.com/Rajeevlang/ecommercesagapattern/order/internals/ports"
	orderv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/order/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GrpcHandler implements the orderv1.OrderServiceServer interface.
type GrpcHandler struct {
	orderv1.UnimplementedOrderServiceServer
	svc ports.Service
}

// NewGrpcHandler creates a new instance of GrpcHandler.
func NewGrpcHandler(svc ports.Service) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

// CreateOrder handles gRPC requests to place a new order and kick off the Saga.
func (h *GrpcHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	var items []domain.OrderItem
	var totalAmount int64

	for _, item := range req.GetItems() {
		items = append(items, domain.OrderItem{
			ProductID:  item.GetProductId(),
			Quantity:   item.GetQuantity(),
			PriceCents: item.GetPriceCents(),
		})
		totalAmount += item.GetPriceCents() * int64(item.GetQuantity())
	}

	order := &domain.Order{
		UserID:             req.GetUserId(),
		TotalAmountCents:   totalAmount,
		PaymentMethodToken: req.GetPaymentMethodToken(),
		Items:              items,
	}

	err := h.svc.CreateOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	return &orderv1.CreateOrderResponse{
		OrderId: order.ID,
		Status:  mapDomainStatusToProto(order.Status),
	}, nil
}

// GetOrder retrieves the details of a single order and maps it back to Protobuf response.
func (h *GrpcHandler) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	order, err := h.svc.GetOrderDetails(ctx, req.GetOrderId())
	if err != nil {
		return nil, err
	}

	var items []*orderv1.OrderItem
	for _, item := range order.Items {
		items = append(items, &orderv1.OrderItem{
			ProductId:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
		})
	}

	return &orderv1.GetOrderResponse{
		OrderId:          order.ID,
		UserId:           order.UserID,
		Items:            items,
		TotalAmountCents: order.TotalAmountCents,
		Status:           mapDomainStatusToProto(order.Status),
		CreatedAt:        timestamppb.New(order.CreatedAt),
		UpdatedAt:        timestamppb.New(order.UpdatedAt),
	}, nil
}

// UpdateOrderStatus modifies order status locally during Saga orchestration.
func (h *GrpcHandler) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	status := mapProtoStatusToDomain(req.GetStatus())
	err := h.svc.UpdateOrderStatus(ctx, req.GetOrderId(), status, req.GetNotes())
	if err != nil {
		return &orderv1.UpdateOrderStatusResponse{Success: false}, err
	}
	return &orderv1.UpdateOrderStatusResponse{Success: true}, nil
}

// Helper: mapDomainStatusToProto converts internal domain status to protobuf OrderStatus.
func mapDomainStatusToProto(status domain.OrderStatus) orderv1.OrderStatus {
	switch status {
	case domain.StatusPending:
		return orderv1.OrderStatus_ORDER_STATUS_PENDING
	case domain.StatusCompleted:
		return orderv1.OrderStatus_ORDER_STATUS_COMPLETED
	case domain.StatusFailed:
		return orderv1.OrderStatus_ORDER_STATUS_FAILED
	case domain.StatusCancelled:
		return orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

// Helper: mapProtoStatusToDomain converts protobuf OrderStatus to internal domain status.
func mapProtoStatusToDomain(status orderv1.OrderStatus) domain.OrderStatus {
	switch status {
	case orderv1.OrderStatus_ORDER_STATUS_PENDING:
		return domain.StatusPending
	case orderv1.OrderStatus_ORDER_STATUS_COMPLETED:
		return domain.StatusCompleted
	case orderv1.OrderStatus_ORDER_STATUS_FAILED:
		return domain.StatusFailed
	case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.StatusCancelled
	default:
		return domain.OrderStatus("ORDER_STATUS_UNSPECIFIED")
	}
}
