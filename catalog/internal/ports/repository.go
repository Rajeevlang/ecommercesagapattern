package ports

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/domain"
)

type CatalogRepository interface {
	GetProduct(ctx context.Context, id string) (*domain.Product, error)
	ListProducts(ctx context.Context, limit int32, cursor string) ([]domain.Product, string, error)
	ReserveStock(ctx context.Context, orderID string, items []domain.Reservation) error
	ReleaseStock(ctx context.Context, orderID string, items []domain.Reservation) error
}
