package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rajeevlang/ecommercesagapattern/catalog/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogRepository struct {
	dbPool *pgxpool.Pool
}

// NewCatalogRepository creates a new PostgreSQL repository adapter for Catalog.
func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{dbPool: pool}
}

// GetProduct retrieves a single product by ID.
func (r *CatalogRepository) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	query := `
		SELECT id, name, description, price_cents, stock, created_at, updated_at
		FROM products
		WHERE id = $1
	`
	var p domain.Product
	err := r.dbPool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.PriceCents,
		&p.Stock,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("product not found: %s", id)
		}
		return nil, err
	}
	return &p, nil
}

// ListProducts returns products with cursor-based pagination.
func (r *CatalogRepository) ListProducts(ctx context.Context, limit int32, cursor string) ([]domain.Product, string, error) {
	if limit <= 0 {
		limit = 10
	}

	var query string
	var rows pgx.Rows
	var err error

	if cursor == "" {
		query = `
			SELECT id, name, description, price_cents, stock, created_at, updated_at
			FROM products
			ORDER BY id ASC
			LIMIT $1
		`
		rows, err = r.dbPool.Query(ctx, query, limit)
	} else {
		query = `
			SELECT id, name, description, price_cents, stock, created_at, updated_at
			FROM products
			WHERE id > $1
			ORDER BY id ASC
			LIMIT $2
		`
		rows, err = r.dbPool.Query(ctx, query, cursor, limit)
	}

	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.PriceCents,
			&p.Stock,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, "", err
		}
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(products) > 0 && len(products) == int(limit) {
		nextCursor = products[len(products)-1].ID
	}

	return products, nextCursor, nil
}

// ReserveStock reserves inventory for a pending order. It locks product rows to prevent race conditions.
func (r *CatalogRepository) ReserveStock(ctx context.Context, orderID string, items []domain.Reservation) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Idempotency Check: check if order already has reservations
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM stock_reservations WHERE order_id = $1)", orderID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		// Already reserved, return nil (noop/idempotency success)
		return nil
	}

	// 2. Lock and reserve stock for each item
	for _, item := range items {
		var currentStock int32
		// SELECT FOR UPDATE to lock product row
		err = tx.QueryRow(ctx, "SELECT stock FROM products WHERE id = $1 FOR UPDATE", item.ProductID).Scan(&currentStock)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("product not found: %s", item.ProductID)
			}
			return err
		}

		if currentStock < item.Quantity {
			return fmt.Errorf("insufficient stock for product %s: available %d, requested %d", item.ProductID, currentStock, item.Quantity)
		}

		// Decrement stock
		_, err = tx.Exec(ctx, "UPDATE products SET stock = stock - $1 WHERE id = $2", item.Quantity, item.ProductID)
		if err != nil {
			return err
		}

		// Insert reservation record
		_, err = tx.Exec(ctx, "INSERT INTO stock_reservations (order_id, product_id, quantity) VALUES ($1, $2, $3)", orderID, item.ProductID, item.Quantity)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// ReleaseStock rolls back a stock reservation, restoring stock levels.
func (r *CatalogRepository) ReleaseStock(ctx context.Context, orderID string, items []domain.Reservation) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Check if reservation exists
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM stock_reservations WHERE order_id = $1)", orderID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		// No reservation found, nothing to rollback. Return nil (noop/idempotency success)
		return nil
	}

	// 2. Query the actual reservation details from database to be absolutely accurate
	rows, err := tx.Query(ctx, "SELECT product_id, quantity FROM stock_reservations WHERE order_id = $1", orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type resRecord struct {
		productID string
		quantity  int32
	}
	var records []resRecord
	for rows.Next() {
		var rec resRecord
		if err := rows.Scan(&rec.productID, &rec.quantity); err != nil {
			return err
		}
		records = append(records, rec)
	}
	rows.Close()

	// 3. Restore stock in products table
	for _, rec := range records {
		_, err = tx.Exec(ctx, "UPDATE products SET stock = stock + $1 WHERE id = $2", rec.quantity, rec.productID)
		if err != nil {
			return err
		}
	}

	// 4. Delete reservations
	_, err = tx.Exec(ctx, "DELETE FROM stock_reservations WHERE order_id = $1", orderID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
