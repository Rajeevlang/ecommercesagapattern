package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
)

type AccountService interface {
	GetProfile(ctx context.Context, userID string) (*domain.Profile, *domain.Address, error)
	UpdateProfile(ctx context.Context, userID, name, phone, avatarURL string) error
	CreateAddress(ctx context.Context, address *domain.Address) (*domain.Address, error)
	ListAddresses(ctx context.Context, userID string) ([]*domain.Address, error)
	DeleteAddress(ctx context.Context, userID, addressID string) error
	SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error
}
