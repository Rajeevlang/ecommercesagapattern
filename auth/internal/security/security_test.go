package security_test

import (
	"testing"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/security"
)

func TestBcryptHasher(t *testing.T) {
	hasher := security.NewBcryptHasher(10)
	password := "mySecretPassword123"

	hash, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !hasher.ComparePassword(hash, password) {
		t.Fatalf("expected ComparePassword to return true for correct password")
	}

	if hasher.ComparePassword(hash, "wrongPassword") {
		t.Fatalf("expected ComparePassword to return false for wrong password")
	}
}

func TestJWTManager(t *testing.T) {
	secret := "my-jwt-test-secret"
	mgr := security.NewJWTManager(secret, 1*time.Hour)

	user := &domain.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  "customer",
	}

	token, err := mgr.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate valid token: %v", err)
	}

	if claims.UserID != user.ID || claims.Email != user.Email || claims.Role != user.Role {
		t.Fatalf("claims mismatch: got %+v, expected user %+v", claims, user)
	}

	// Test invalid token
	_, err = mgr.ValidateToken("invalid.token.str")
	if err != domain.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
