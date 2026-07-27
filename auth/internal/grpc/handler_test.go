package grpc_test

import (
	"context"
	"testing"
	"time"

	authgrpc "github.com/Rajeevlang/ecommercesagapattern/auth/internal/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/security"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/service"
	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupGRPCHandler() *authgrpc.AuthGRPCHandler {
	repo := memory.NewInMemoryUserRepository()
	hasher := security.NewBcryptHasher(10)
	tokenMgr := security.NewJWTManager("test-grpc-secret", 1*time.Hour)
	authSvc := service.NewAuthService(repo, hasher, tokenMgr)
	return authgrpc.NewAuthGRPCHandler(authSvc)
}


func TestGRPCHandler_RegisterAndLogin(t *testing.T) {
	handler := setupGRPCHandler()
	ctx := context.Background()

	// 1. Test Register
	regRes, err := handler.Register(ctx, &authv1.RegisterRequest{
		Email:    "grpcuser@example.com",
		Password: "password123",
		Name:     "gRPC User",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if regRes.GetUserId() == "" {
		t.Fatalf("expected non-empty user_id in Register response")
	}

	// 2. Test Register Duplicate
	_, err = handler.Register(ctx, &authv1.RegisterRequest{
		Email:    "grpcuser@example.com",
		Password: "password123",
		Name:     "gRPC User",
	})
	if err == nil {
		t.Fatalf("expected error on duplicate register, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.AlreadyExists {
		t.Fatalf("expected codes.AlreadyExists, got %v", st.Code())
	}

	// 3. Test Login Success
	loginRes, err := handler.Login(ctx, &authv1.LoginRequest{
		Email:    "grpcuser@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginRes.GetToken() == "" || loginRes.GetUserId() != regRes.GetUserId() {
		t.Fatalf("invalid Login response: %v", loginRes)
	}

	// 4. Test Login Unauthenticated (wrong pass)
	_, err = handler.Login(ctx, &authv1.LoginRequest{
		Email:    "grpcuser@example.com",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Fatalf("expected error on invalid password login")
	}
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected codes.Unauthenticated, got %v", st.Code())
	}

	// 5. Test ValidateToken
	valRes, err := handler.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		Token: loginRes.GetToken(),
	})
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if !valRes.GetIsValid() || valRes.GetUserId() != regRes.GetUserId() {
		t.Fatalf("ValidateToken returned invalid response: %v", valRes)
	}
}
