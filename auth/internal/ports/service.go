package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
)

type AuthService interface {
	Register(ctx context.Context, email, password, name string) (string, error)
	Login(ctx context.Context, email, password string) (string, string, error)
	ValidateToken(ctx context.Context, tokenStr string) (*domain.TokenClaims, error)
}
