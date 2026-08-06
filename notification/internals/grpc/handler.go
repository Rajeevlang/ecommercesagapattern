package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/ports"
	notificationv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/notification/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	notificationv1.UnimplementedNotificationServiceServer
	svc ports.NotificationService
}

func NewGrpcHandler(svc ports.NotificationService) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

func (h *GrpcHandler) SendEmail(ctx context.Context, req *notificationv1.SendEmailRequest) (*notificationv1.SendEmailResponse, error) {
	if req.GetRecipientEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient email cannot be empty")
	}

	// Since the simple proto doesn't have an idempotency key, we generate one based on recipient, subject, and timestamp
	idempotencyKey := fmt.Sprintf("grpc-email-%s-%x-%d", req.GetRecipientEmail(), req.GetSubject(), time.Now().UnixNano())

	_, err := h.svc.SendEmail(ctx, "system", req.GetRecipientEmail(), req.GetSubject(), req.GetBody(), "direct_email", idempotencyKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send email: %v", err)
	}

	return &notificationv1.SendEmailResponse{
		Success: true,
	}, nil
}
