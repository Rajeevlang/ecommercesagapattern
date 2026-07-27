package service

import (
	"context"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/ports"
	"github.com/google/uuid"
)

type AccountService struct {
	repo ports.AccountRepository
}

func NewAccountService(repo ports.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) GetProfile(ctx context.Context, userID string) (*domain.Profile, *domain.Address, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	addresses, _ := s.repo.ListAddressesByUserID(ctx, userID)
	var defaultAddress *domain.Address
	for _, addr := range addresses {
		if addr.IsDefault && addr.AddressType == "SHIPPING" {
			defaultAddress = addr
			break
		}
	}

	return profile, defaultAddress, nil
}

func (s *AccountService) UpdateProfile(ctx context.Context, userID, name, phone, avatarURL string) error {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		now := time.Now()
		profile = &domain.Profile{
			UserID:    userID,
			Name:      name,
			Phone:     phone,
			AvatarURL: avatarURL,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return s.repo.CreateProfile(ctx, profile)
	}

	profile.Name = name
	profile.Phone = phone
	profile.AvatarURL = avatarURL
	profile.UpdatedAt = time.Now()
	return s.repo.UpdateProfile(ctx, profile)
}

func (s *AccountService) CreateAddress(ctx context.Context, address *domain.Address) (*domain.Address, error) {
	if address.Street == "" || address.City == "" || address.Country == "" {
		return nil, domain.ErrInvalidAddress
	}

	if address.ID == "" {
		address.ID = uuid.New().String()
	}
	if address.AddressType == "" {
		address.AddressType = "SHIPPING"
	}
	now := time.Now()
	address.CreatedAt = now
	address.UpdatedAt = now

	if err := s.repo.CreateAddress(ctx, address); err != nil {
		return nil, err
	}

	if address.IsDefault {
		_ = s.repo.SetDefaultAddress(ctx, address.UserID, address.ID, address.AddressType)
	}

	return address, nil
}

func (s *AccountService) ListAddresses(ctx context.Context, userID string) ([]*domain.Address, error) {
	return s.repo.ListAddressesByUserID(ctx, userID)
}

func (s *AccountService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	return s.repo.DeleteAddress(ctx, userID, addressID)
}

func (s *AccountService) SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error {
	return s.repo.SetDefaultAddress(ctx, userID, addressID, addressType)
}
