package grpc

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/ports"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
)

type GrpcHandler struct {
	catalogv1.UnimplementedCatalogServiceServer
	svc ports.CatalogService
}

// NewGrpcHandler constructs a new GrpcHandler adapter for the Catalog gRPC service.
func NewGrpcHandler(svc ports.CatalogService) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

// GetProduct handles the gRPC query to retrieve details of a single product.
func (h *GrpcHandler) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	p, err := h.svc.GetProduct(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &catalogv1.GetProductResponse{
		Product: mapDomainProductToProto(p),
	}, nil
}

// ListProducts handles paginated retrieval of products.
func (h *GrpcHandler) ListProducts(ctx context.Context, req *catalogv1.ListProductsRequest) (*catalogv1.ListProductsResponse, error) {
	products, nextCursor, err := h.svc.ListProducts(ctx, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}

	var protoProducts []*catalogv1.Product
	for _, p := range products {
		protoProducts = append(protoProducts, mapDomainProductToProto(&p))
	}

	return &catalogv1.ListProductsResponse{
		Products:      protoProducts,
		NextPageToken: nextCursor,
	}, nil
}

// ReserveStock manages the Saga Forward Action of inventory locks.
func (h *GrpcHandler) ReserveStock(ctx context.Context, req *catalogv1.ReserveStockRequest) (*catalogv1.ReserveStockResponse, error) {
	var reservations []domain.Reservation
	for _, item := range req.GetItems() {
		reservations = append(reservations, domain.Reservation{
			OrderID:   req.GetOrderId(),
			ProductID: item.GetProductId(),
			Quantity:  item.GetQuantity(),
		})
	}

	err := h.svc.ReserveStock(ctx, req.GetOrderId(), reservations)
	if err != nil {
		return &catalogv1.ReserveStockResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &catalogv1.ReserveStockResponse{
		Success: true,
		Message: "Stock reserved successfully",
	}, nil
}

// ReleaseStock manages the Saga Compensating Action of inventory rollback.
func (h *GrpcHandler) ReleaseStock(ctx context.Context, req *catalogv1.ReleaseStockRequest) (*catalogv1.ReleaseStockResponse, error) {
	var reservations []domain.Reservation
	for _, item := range req.GetItems() {
		reservations = append(reservations, domain.Reservation{
			OrderID:   req.GetOrderId(),
			ProductID: item.GetProductId(),
			Quantity:  item.GetQuantity(),
		})
	}

	err := h.svc.ReleaseStock(ctx, req.GetOrderId(), reservations)
	if err != nil {
		return &catalogv1.ReleaseStockResponse{
			Success: false,
		}, nil
	}

	return &catalogv1.ReleaseStockResponse{
		Success: true,
	}, nil
}

func mapDomainProductToProto(p *domain.Product) *catalogv1.Product {
	if p == nil {
		return nil
	}
	return &catalogv1.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Stock:       p.Stock,
	}
}
