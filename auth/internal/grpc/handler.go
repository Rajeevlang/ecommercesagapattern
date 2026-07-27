package grpc

import (
	"context"
	"errors"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/ports"
	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthGRPCHandler struct {
	authv1.UnimplementedAuthServiceServer
	authService ports.AuthService
}

func NewAuthGRPCHandler(authService ports.AuthService) *AuthGRPCHandler {
	return &AuthGRPCHandler{
		authService: authService,
	}
}

func (h *AuthGRPCHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	userID, err := h.authService.Register(ctx, req.GetEmail(), req.GetPassword(), req.GetName())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		case errors.Is(err, domain.ErrInvalidCredentials):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &authv1.RegisterResponse{
		UserId: userID,
	}, nil
}

func (h *AuthGRPCHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	token, userID, err := h.authService.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrUserNotFound):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, domain.ErrUserInactive):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &authv1.LoginResponse{
		Token:  token,
		UserId: userID,
	}, nil
}

func (h *AuthGRPCHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	claims, err := h.authService.ValidateToken(ctx, req.GetToken())
	if err != nil {
		return &authv1.ValidateTokenResponse{
			IsValid: false,
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		IsValid: true,
		UserId:  claims.UserID,
		Role:    claims.Role,
	}, nil
}

