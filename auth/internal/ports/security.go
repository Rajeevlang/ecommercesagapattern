package ports

import (
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hashedPassword, password string) bool
}

type TokenManager interface {
	GenerateToken(user *domain.User) (string, error)
	ValidateToken(tokenStr string) (*domain.TokenClaims, error)
}
