package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
)

type AccountRepository interface {
	CreateProfile(ctx context.Context, profile *domain.Profile) error
	GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, profile *domain.Profile) error

	CreateAddress(ctx context.Context, address *domain.Address) error
	GetAddressByID(ctx context.Context, addressID string) (*domain.Address, error)
	ListAddressesByUserID(ctx context.Context, userID string) ([]*domain.Address, error)
	DeleteAddress(ctx context.Context, userID, addressID string) error
	SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error
}
