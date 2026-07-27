package memory

import (
	"context"
	"sync"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
)

type InMemoryAccountRepository struct {
	mu        sync.RWMutex
	profiles  map[string]*domain.Profile
	addresses map[string]*domain.Address // keyed by address.ID
}

func NewInMemoryAccountRepository() *InMemoryAccountRepository {
	return &InMemoryAccountRepository{
		profiles:  make(map[string]*domain.Profile),
		addresses: make(map[string]*domain.Address),
	}
}

func (r *InMemoryAccountRepository) CreateProfile(ctx context.Context, profile *domain.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	r.profiles[profile.UserID] = profile
	return nil
}

func (r *InMemoryAccountRepository) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[userID]
	if !ok {
		return nil, domain.ErrProfileNotFound
	}
	return p, nil
}

func (r *InMemoryAccountRepository) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	profile.UpdatedAt = time.Now()
	r.profiles[profile.UserID] = profile
	return nil
}

func (r *InMemoryAccountRepository) CreateAddress(ctx context.Context, address *domain.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if address.CreatedAt.IsZero() {
		address.CreatedAt = now
	}
	address.UpdatedAt = now
	r.addresses[address.ID] = address
	return nil
}

func (r *InMemoryAccountRepository) GetAddressByID(ctx context.Context, addressID string) (*domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.addresses[addressID]
	if !ok {
		return nil, domain.ErrAddressNotFound
	}
	return a, nil
}

func (r *InMemoryAccountRepository) ListAddressesByUserID(ctx context.Context, userID string) ([]*domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*domain.Address
	for _, a := range r.addresses {
		if a.UserID == userID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (r *InMemoryAccountRepository) DeleteAddress(ctx context.Context, userID, addressID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.addresses[addressID]
	if !ok || a.UserID != userID {
		return domain.ErrAddressNotFound
	}
	delete(r.addresses, addressID)
	return nil
}

func (r *InMemoryAccountRepository) SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for _, a := range r.addresses {
		if a.UserID == userID && a.AddressType == addressType {
			if a.ID == addressID {
				a.IsDefault = true
				found = true
			} else {
				a.IsDefault = false
			}
		}
	}
	if !found {
		return domain.ErrAddressNotFound
	}
	return nil
}
