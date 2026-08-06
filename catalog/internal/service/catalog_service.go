package service

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/ports"
)

type CatalogService struct {
	repo ports.CatalogRepository
}

// NewCatalogService constructs a new CatalogService with the injected repository port.
func NewCatalogService(repo ports.CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

// GetProduct queries a single product.
func (s *CatalogService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	return s.repo.GetProduct(ctx, id)
}

// ListProducts retrieves a paginated set of products.
func (s *CatalogService) ListProducts(ctx context.Context, limit int32, cursor string) ([]domain.Product, string, error) {
	return s.repo.ListProducts(ctx, limit, cursor)
}

// ReserveStock executes the reservation phase of inventory locks.
func (s *CatalogService) ReserveStock(ctx context.Context, orderID string, items []domain.Reservation) error {
	return s.repo.ReserveStock(ctx, orderID, items)
}

// ReleaseStock executes the compensation rollback phase of inventory locks.
func (s *CatalogService) ReleaseStock(ctx context.Context, orderID string, items []domain.Reservation) error {
	return s.repo.ReleaseStock(ctx, orderID, items)
}
