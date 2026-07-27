package memory

import (
	"context"
	"sync"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
)

type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User // keyed by ID
	emails map[string]string      // email -> ID mapping
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:  make(map[string]*domain.User),
		emails: make(map[string]string),
	}
}

func (r *InMemoryUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.emails[user.Email]; exists {
		return domain.ErrUserAlreadyExists
	}

	r.users[user.ID] = user
	r.emails[user.Email] = user.ID
	return nil
}

func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.emails[email]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
