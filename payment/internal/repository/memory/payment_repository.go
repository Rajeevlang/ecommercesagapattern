package memory

import (
	"context"
	"sync"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/domain"
	"github.com/google/uuid"
)

type InMemoryPaymentRepository struct {
	mu       sync.RWMutex
	payments map[string]*domain.Payment
}

func NewInMemoryPaymentRepository() *InMemoryPaymentRepository {
	return &InMemoryPaymentRepository{
		payments: make(map[string]*domain.Payment),
	}
}

func (r *InMemoryPaymentRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.payments {
		if existing.OrderID == p.OrderID {
			return domain.ErrDuplicatePayment
		}
	}

	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	saved := *p
	r.payments[p.ID] = &saved
	return nil
}

func (r *InMemoryPaymentRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.payments {
		if p.OrderID == orderID {
			ret := *p
			return &ret, nil
		}
	}
	return nil, domain.ErrPaymentNotFound
}

func (r *InMemoryPaymentRepository) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.payments[id]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}
	ret := *p
	return &ret, nil
}

func (r *InMemoryPaymentRepository) UpdatePaymentStatus(ctx context.Context, id string, status domain.PaymentStatus, errMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.payments[id]
	if !exists {
		return domain.ErrPaymentNotFound
	}

	p.Status = status
	p.ErrorMessage = errMessage
	p.UpdatedAt = time.Now()
	return nil
}
