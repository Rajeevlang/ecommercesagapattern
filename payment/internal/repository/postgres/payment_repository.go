package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rajeevlang/ecommercesagapattern/payment/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	dbPool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{dbPool: pool}
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	query := `
		INSERT INTO payments (
			order_id, user_id, amount_cents, payment_method_token, status, error_message
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := r.dbPool.QueryRow(ctx, query,
		p.OrderID,
		p.UserID,
		p.AmountCents,
		p.PaymentMethodToken,
		string(p.Status),
		p.ErrorMessage,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation for order_id
			return domain.ErrDuplicatePayment
		}
		return fmt.Errorf("failed to insert payment record: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount_cents, payment_method_token, status, error_message, created_at, updated_at
		FROM payments
		WHERE order_id = $1
	`
	var p domain.Payment
	var status string
	err := r.dbPool.QueryRow(ctx, query, orderID).Scan(
		&p.ID,
		&p.OrderID,
		&p.UserID,
		&p.AmountCents,
		&p.PaymentMethodToken,
		&status,
		&p.ErrorMessage,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	p.Status = domain.PaymentStatus(status)
	return &p, nil
}

func (r *PaymentRepository) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount_cents, payment_method_token, status, error_message, created_at, updated_at
		FROM payments
		WHERE id = $1
	`
	var p domain.Payment
	var status string
	err := r.dbPool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.OrderID,
		&p.UserID,
		&p.AmountCents,
		&p.PaymentMethodToken,
		&status,
		&p.ErrorMessage,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	p.Status = domain.PaymentStatus(status)
	return &p, nil
}

func (r *PaymentRepository) UpdatePaymentStatus(ctx context.Context, id string, status domain.PaymentStatus, errMessage string) error {
	query := `
		UPDATE payments
		SET status = $1, error_message = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := r.dbPool.Exec(ctx, query, string(status), errMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	return nil
}
