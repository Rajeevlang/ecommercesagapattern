package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/ports"
	"github.com/google/uuid"
)

type AuthService struct {
	repo         ports.UserRepository
	hasher       ports.PasswordHasher
	tokenManager ports.TokenManager
}

func NewAuthService(
	repo ports.UserRepository,
	hasher ports.PasswordHasher,
	tokenManager ports.TokenManager,
) *AuthService {
	return &AuthService{
		repo:         repo,
		hasher:       hasher,
		tokenManager: tokenManager,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (string, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	if cleanEmail == "" || password == "" {
		return "", domain.ErrInvalidCredentials
	}

	existing, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return "", err
	}
	if existing != nil {
		return "", domain.ErrUserAlreadyExists
	}

	hash, err := s.hasher.HashPassword(password)
	if err != nil {
		return "", err
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        cleanEmail,
		PasswordHash: hash,
		Name:         name,
		Role:         "customer",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return "", err
	}

	return user.ID, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", err
	}

	if !s.hasher.ComparePassword(user.PasswordHash, password) {
		return "", "", domain.ErrInvalidCredentials
	}

	token, err := s.tokenManager.GenerateToken(user)
	if err != nil {
		return "", "", err
	}

	return token, user.ID, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenStr string) (*domain.TokenClaims, error) {
	// Strip "Bearer " prefix if provided
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	tokenStr = strings.TrimSpace(tokenStr)

	return s.tokenManager.ValidateToken(tokenStr)
}

