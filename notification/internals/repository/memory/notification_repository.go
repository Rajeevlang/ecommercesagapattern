package memory

import (
	"context"
	"sync"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/domain"
	"github.com/google/uuid"
)

type InMemoryNotificationRepository struct {
	mu            sync.RWMutex
	notifications map[string]*domain.Notification
}

func NewInMemoryNotificationRepository() *InMemoryNotificationRepository {
	return &InMemoryNotificationRepository{
		notifications: make(map[string]*domain.Notification),
	}
}

func (r *InMemoryNotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.notifications {
		if existing.IdempotencyKey == n.IdempotencyKey {
			return domain.ErrDuplicateNotification
		}
	}

	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now

	// Clone to save
	saved := *n
	r.notifications[n.ID] = &saved
	return nil
}

func (r *InMemoryNotificationRepository) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n, exists := r.notifications[id]
	if !exists {
		return nil, domain.ErrNotificationNotFound
	}
	// return clone
	ret := *n
	return &ret, nil
}

func (r *InMemoryNotificationRepository) GetNotificationByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, n := range r.notifications {
		if n.IdempotencyKey == idempotencyKey {
			ret := *n
			return &ret, nil
		}
	}
	return nil, domain.ErrNotificationNotFound
}

func (r *InMemoryNotificationRepository) UpdateNotification(ctx context.Context, n *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.notifications[n.ID]
	if !exists {
		return domain.ErrNotificationNotFound
	}

	existing.Status = n.Status
	existing.RetryCount = n.RetryCount
	existing.ErrorMessage = n.ErrorMessage
	existing.SentAt = n.SentAt
	existing.UpdatedAt = time.Now()

	*n = *existing
	return nil
}

func (r *InMemoryNotificationRepository) ListPendingOrRetrying(ctx context.Context) ([]*domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Notification
	for _, n := range r.notifications {
		if n.Status == domain.StatusPending || n.Status == domain.StatusRetrying {
			ret := *n
			result = append(result, &ret)
		}
	}
	return result, nil
}
