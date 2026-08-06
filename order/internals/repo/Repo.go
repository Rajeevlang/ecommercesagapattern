package repo

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/order/internals/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepox implements the ports.OrderRepository interface using PostgreSQL.
type OrderRepox struct {
	dbPool *pgxpool.Pool
}

// NewRepoHandler creates a new instance of OrderRepox.
func NewRepoHandler(pool *pgxpool.Pool) *OrderRepox {
	return &OrderRepox{dbPool: pool}
}

// CreateOrder stores a new order and all its items inside a database transaction.
func (h *OrderRepox) CreateOrder(ctx context.Context, order *domain.Order) error {
	tx, err := h.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Insert order metadata
	var orderStmt string
	if order.ID == "" {
		orderStmt = `
			INSERT INTO orders (user_id, total_amount_cents, status, notes)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`
		err = tx.QueryRow(ctx, orderStmt, order.UserID, order.TotalAmountCents, string(order.Status), order.Notes).
			Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	} else {
		orderStmt = `
			INSERT INTO orders (id, user_id, total_amount_cents, status, notes)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING created_at, updated_at
		`
		err = tx.QueryRow(ctx, orderStmt, order.ID, order.UserID, order.TotalAmountCents, string(order.Status), order.Notes).
			Scan(&order.CreatedAt, &order.UpdatedAt)
	}
	if err != nil {
		return err
	}

	// 2. Insert order items
	itemStmt := `
		INSERT INTO order_items (order_id, product_id, quantity, price_cents)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	for i := range order.Items {
		item := &order.Items[i]
		err = tx.QueryRow(ctx, itemStmt, order.ID, item.ProductID, item.Quantity, item.PriceCents).
			Scan(&item.ID)
		if err != nil {
			return err
		}
		item.OrderID = order.ID
	}

	return tx.Commit(ctx)
}

// UpdateOrderStatus updates the status and notes of a specific order.
func (h *OrderRepox) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus, notes string) error {
	stmt := `UPDATE orders SET status = $1, notes = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	_, err := h.dbPool.Exec(ctx, stmt, string(status), notes, orderID)
	return err
}

// GetOrderDetails retrieves an order by its ID, including its associated line items.
func (h *OrderRepox) GetOrderDetails(ctx context.Context, orderID string) (*domain.Order, error) {
	order := &domain.Order{}

	// Query order metadata
	orderStmt := `
		SELECT id, user_id, total_amount_cents, status, notes, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	var statusStr string
	err := h.dbPool.QueryRow(ctx, orderStmt, orderID).Scan(
		&order.ID,
		&order.UserID,
		&order.TotalAmountCents,
		&statusStr,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	order.Status = domain.OrderStatus(statusStr)

	// Query order items
	itemsStmt := `
		SELECT id, order_id, product_id, quantity, price_cents
		FROM order_items
		WHERE order_id = $1
	`
	rows, err := h.dbPool.Query(ctx, itemsStmt, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		err = rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceCents)
		if err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return order, nil
}
