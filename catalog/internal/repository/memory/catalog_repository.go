package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/domain"
)

type InMemoryCatalogRepository struct {
	mu           sync.RWMutex
	products     map[string]*domain.Product
	reservations map[string][]domain.Reservation // order_id -> list of reservations
}

// NewInMemoryCatalogRepository creates a new in-memory repository seeded with default products.
func NewInMemoryCatalogRepository() *InMemoryCatalogRepository {
	repo := &InMemoryCatalogRepository{
		products:     make(map[string]*domain.Product),
		reservations: make(map[string][]domain.Reservation),
	}
	repo.seed()
	return repo
}

func (r *InMemoryCatalogRepository) seed() {
	p1 := &domain.Product{
		ID:          "a4f4efb5-2efc-4e89-8d7a-1a8511cb89ff",
		Name:        "Wireless Mechanical Keyboard",
		Description: "A 75% mechanical keyboard with hot-swappable tactile switches.",
		PriceCents:  8999,
		Stock:       100,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	p2 := &domain.Product{
		ID:          "b5e5f0c6-3ffd-4f9a-9e8b-2b9622dc90aa",
		Name:        "Ergonomic Wireless Mouse",
		Description: "Multi-device wireless mouse with precise scroll wheel.",
		PriceCents:  5999,
		Stock:       150,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	p3 := &domain.Product{
		ID:          "c6f6f1d7-400e-50ab-af9c-3ca733ed01bb",
		Name:        "4K Ultra-Wide Monitor",
		Description: "34-inch curved monitor with HDR support and 144Hz refresh rate.",
		PriceCents:  49999,
		Stock:       30,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	r.products[p1.ID] = p1
	r.products[p2.ID] = p2
	r.products[p3.ID] = p3
}

// GetProduct retrieves a single product from memory.
func (r *InMemoryCatalogRepository) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.products[id]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", id)
	}

	pCopy := *p
	return &pCopy, nil
}

// ListProducts returns a sorted slice of products from memory.
func (r *InMemoryCatalogRepository) ListProducts(ctx context.Context, limit int32, cursor string) ([]domain.Product, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	var list []domain.Product
	var keys []string

	for k := range r.products {
		if cursor == "" || k > cursor {
			keys = append(keys, k)
		}
	}

	// Sort keys alphabetically
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	count := int32(0)
	nextCursor := ""
	for _, k := range keys {
		if count >= limit {
			nextCursor = k
			break
		}
		list = append(list, *r.products[k])
		count++
	}

	return list, nextCursor, nil
}

// ReserveStock locks inventory in-memory.
func (r *InMemoryCatalogRepository) ReserveStock(ctx context.Context, orderID string, items []domain.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.reservations[orderID]; ok {
		return nil
	}

	// Verify stock levels first
	for _, item := range items {
		p, ok := r.products[item.ProductID]
		if !ok {
			return fmt.Errorf("product not found: %s", item.ProductID)
		}
		if p.Stock < item.Quantity {
			return fmt.Errorf("insufficient stock for product %s: available %d, requested %d", item.ProductID, p.Stock, item.Quantity)
		}
	}

	// Apply deductions
	for _, item := range items {
		r.products[item.ProductID].Stock -= item.Quantity
	}
	r.reservations[orderID] = items

	return nil
}

// ReleaseStock unlocks inventory in-memory.
func (r *InMemoryCatalogRepository) ReleaseStock(ctx context.Context, orderID string, items []domain.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, ok := r.reservations[orderID]
	if !ok {
		return nil
	}

	for _, item := range res {
		if p, ok := r.products[item.ProductID]; ok {
			p.Stock += item.Quantity
		}
	}

	delete(r.reservations, orderID)
	return nil
}
