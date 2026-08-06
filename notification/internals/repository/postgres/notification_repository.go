package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	dbPool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{dbPool: pool}
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (
			idempotency_key, user_id, recipient, channel, template_name, subject, content, status, retry_count, max_retries, error_message, sent_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`
	err := r.dbPool.QueryRow(ctx, query,
		n.IdempotencyKey,
		n.UserID,
		n.Recipient,
		string(n.Channel),
		n.TemplateName,
		n.Subject,
		n.Content,
		string(n.Status),
		n.RetryCount,
		n.MaxRetries,
		n.ErrorMessage,
		n.SentAt,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return domain.ErrDuplicateNotification
		}
		return fmt.Errorf("failed to insert notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	query := `
		SELECT id, idempotency_key, user_id, recipient, channel, template_name, subject, content, status, retry_count, max_retries, error_message, sent_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`
	var n domain.Notification
	var channel, status string
	err := r.dbPool.QueryRow(ctx, query, id).Scan(
		&n.ID,
		&n.IdempotencyKey,
		&n.UserID,
		&n.Recipient,
		&channel,
		&n.TemplateName,
		&n.Subject,
		&n.Content,
		&status,
		&n.RetryCount,
		&n.MaxRetries,
		&n.ErrorMessage,
		&n.SentAt,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, err
	}
	n.Channel = domain.NotificationChannel(channel)
	n.Status = domain.NotificationStatus(status)
	return &n, nil
}

func (r *NotificationRepository) GetNotificationByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Notification, error) {
	query := `
		SELECT id, idempotency_key, user_id, recipient, channel, template_name, subject, content, status, retry_count, max_retries, error_message, sent_at, created_at, updated_at
		FROM notifications
		WHERE idempotency_key = $1
	`
	var n domain.Notification
	var channel, status string
	err := r.dbPool.QueryRow(ctx, query, idempotencyKey).Scan(
		&n.ID,
		&n.IdempotencyKey,
		&n.UserID,
		&n.Recipient,
		&channel,
		&n.TemplateName,
		&n.Subject,
		&n.Content,
		&status,
		&n.RetryCount,
		&n.MaxRetries,
		&n.ErrorMessage,
		&n.SentAt,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, err
	}
	n.Channel = domain.NotificationChannel(channel)
	n.Status = domain.NotificationStatus(status)
	return &n, nil
}

func (r *NotificationRepository) UpdateNotification(ctx context.Context, n *domain.Notification) error {
	query := `
		UPDATE notifications
		SET status = $1, retry_count = $2, error_message = $3, sent_at = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`
	_, err := r.dbPool.Exec(ctx, query,
		string(n.Status),
		n.RetryCount,
		n.ErrorMessage,
		n.SentAt,
		n.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) ListPendingOrRetrying(ctx context.Context) ([]*domain.Notification, error) {
	query := `
		SELECT id, idempotency_key, user_id, recipient, channel, template_name, subject, content, status, retry_count, max_retries, error_message, sent_at, created_at, updated_at
		FROM notifications
		WHERE status IN ('PENDING', 'RETRYING')
		ORDER BY created_at ASC
	`
	rows, err := r.dbPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var list []*domain.Notification
	for rows.Next() {
		var n domain.Notification
		var channel, status string
		err := rows.Scan(
			&n.ID,
			&n.IdempotencyKey,
			&n.UserID,
			&n.Recipient,
			&channel,
			&n.TemplateName,
			&n.Subject,
			&n.Content,
			&status,
			&n.RetryCount,
			&n.MaxRetries,
			&n.ErrorMessage,
			&n.SentAt,
			&n.CreatedAt,
			&n.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		n.Channel = domain.NotificationChannel(channel)
		n.Status = domain.NotificationStatus(status)
		list = append(list, &n)
	}
	return list, nil
}
