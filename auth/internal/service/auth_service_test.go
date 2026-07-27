package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/security"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/service"
)

func setupTestAuthService() *service.AuthService {
	repo := memory.NewInMemoryUserRepository()
	hasher := security.NewBcryptHasher(10)
	tokenMgr := security.NewJWTManager("test-secret-key", 24*time.Hour)
	return service.NewAuthService(repo, hasher, tokenMgr)
}


func TestRegister_Success(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	userID, err := svc.Register(ctx, "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if userID == "" {
		t.Fatalf("expected non-empty user ID")
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "dup@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = svc.Register(ctx, "dup@example.com", "anotherpass", "Another User")
	if err != domain.ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestRegister_EmptyCredentials(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "", "password123", "Test User")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for empty email, got %v", err)
	}

	_, err = svc.Register(ctx, "test@example.com", "", "Test User")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for empty password, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	userID, err := svc.Register(ctx, "login@example.com", "password123", "Login User")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	token, returnedID, err := svc.Login(ctx, "login@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error on login, got %v", err)
	}
	if returnedID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, returnedID)
	}
	if token == "" {
		t.Fatalf("expected non-empty JWT token")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "wrongpass@example.com", "correctpassword", "Wrong Pass User")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	_, _, err = svc.Login(ctx, "wrongpass@example.com", "incorrectpassword")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "nonexistent@example.com", "password123")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for non-existent user, got %v", err)
	}
}

func TestValidateToken_Success(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	userID, err := svc.Register(ctx, "val@example.com", "password123", "Val User")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	token, _, err := svc.Login(ctx, "val@example.com", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	claims, err := svc.ValidateToken(ctx, "Bearer "+token)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != "val@example.com" {
		t.Fatalf("expected email val@example.com, got %s", claims.Email)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	svc := setupTestAuthService()
	ctx := context.Background()

	_, err := svc.ValidateToken(ctx, "invalid.token.string")
	if err != domain.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
