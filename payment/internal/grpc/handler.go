package grpc

import (
	"context"
	"errors"

	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/ports"
	paymentv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	svc ports.PaymentService
}

func NewGrpcHandler(svc ports.PaymentService) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

func (h *GrpcHandler) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id cannot be empty")
	}
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id cannot be empty")
	}

	p, err := h.svc.ProcessPayment(ctx, req.GetOrderId(), req.GetUserId(), req.GetAmountCents(), req.GetPaymentMethodToken())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAmount) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
	}

	success := p.Status == domain.StatusAuthorized

	return &paymentv1.ProcessPaymentResponse{
		PaymentId:         p.ID,
		Success:           success,
		TransactionStatus: string(p.Status),
		ErrorMessage:      p.ErrorMessage,
	}, nil
}

func (h *GrpcHandler) CancelPayment(ctx context.Context, req *paymentv1.CancelPaymentRequest) (*paymentv1.CancelPaymentResponse, error) {
	if req.GetOrderId() == "" && req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "must specify order_id or payment_id")
	}

	err := h.svc.CancelPayment(ctx, req.GetOrderId(), req.GetPaymentId(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel payment: %v", err)
	}

	return &paymentv1.CancelPaymentResponse{
		Success: true,
	}, nil
}
