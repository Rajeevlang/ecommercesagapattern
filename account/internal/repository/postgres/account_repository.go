package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAccountRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAccountRepository(pool *pgxpool.Pool) *PostgresAccountRepository {
	return &PostgresAccountRepository{pool: pool}
}

func (r *PostgresAccountRepository) CreateProfile(ctx context.Context, profile *domain.Profile) error {
	query := `
		INSERT INTO profiles (user_id, email, name, phone, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			name = EXCLUDED.name,
			phone = EXCLUDED.phone,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, profile.UserID, profile.Email, profile.Name, profile.Phone, profile.AvatarURL)
	if err != nil {
		return fmt.Errorf("failed to insert profile: %w", err)
	}
	return nil
}

func (r *PostgresAccountRepository) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	query := `
		SELECT user_id, email, name, phone, avatar_url, created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`
	var p domain.Profile
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&p.UserID,
		&p.Email,
		&p.Name,
		&p.Phone,
		&p.AvatarURL,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &p, nil
}

func (r *PostgresAccountRepository) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	query := `
		UPDATE profiles
		SET name = $2, phone = $3, avatar_url = $4, updated_at = NOW()
		WHERE user_id = $1
	`
	cmd, err := r.pool.Exec(ctx, query, profile.UserID, profile.Name, profile.Phone, profile.AvatarURL)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrProfileNotFound
	}
	return nil
}

func (r *PostgresAccountRepository) CreateAddress(ctx context.Context, address *domain.Address) error {
	query := `
		INSERT INTO addresses (id, user_id, street, city, state, country, zip_code, is_default, address_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query,
		address.ID,
		address.UserID,
		address.Street,
		address.City,
		address.State,
		address.Country,
		address.ZipCode,
		address.IsDefault,
		address.AddressType,
	)
	if err != nil {
		return fmt.Errorf("failed to insert address: %w", err)
	}
	return nil
}

func (r *PostgresAccountRepository) GetAddressByID(ctx context.Context, addressID string) (*domain.Address, error) {
	query := `
		SELECT id, user_id, street, city, state, country, zip_code, is_default, address_type, created_at, updated_at
		FROM addresses
		WHERE id = $1
	`
	var a domain.Address
	err := r.pool.QueryRow(ctx, query, addressID).Scan(
		&a.ID,
		&a.UserID,
		&a.Street,
		&a.City,
		&a.State,
		&a.Country,
		&a.ZipCode,
		&a.IsDefault,
		&a.AddressType,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get address: %w", err)
	}
	return &a, nil
}

func (r *PostgresAccountRepository) ListAddressesByUserID(ctx context.Context, userID string) ([]*domain.Address, error) {
	query := `
		SELECT id, user_id, street, city, state, country, zip_code, is_default, address_type, created_at, updated_at
		FROM addresses
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query addresses: %w", err)
	}
	defer rows.Close()

	var list []*domain.Address
	for rows.Next() {
		var a domain.Address
		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.Street,
			&a.City,
			&a.State,
			&a.Country,
			&a.ZipCode,
			&a.IsDefault,
			&a.AddressType,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan address row: %w", err)
		}
		list = append(list, &a)
	}
	return list, nil
}

func (r *PostgresAccountRepository) DeleteAddress(ctx context.Context, userID, addressID string) error {
	query := `
		DELETE FROM addresses
		WHERE id = $1 AND user_id = $2
	`
	cmd, err := r.pool.Exec(ctx, query, addressID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *PostgresAccountRepository) SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set all addresses of this type for user to false
	_, err = tx.Exec(ctx, `
		UPDATE addresses
		SET is_default = FALSE
		WHERE user_id = $1 AND address_type = $2
	`, userID, addressType)
	if err != nil {
		return fmt.Errorf("failed to reset default flags: %w", err)
	}

	// Set target address to default
	cmd, err := tx.Exec(ctx, `
		UPDATE addresses
		SET is_default = TRUE
		WHERE id = $1 AND user_id = $2 AND address_type = $3
	`, addressID, userID, addressType)
	if err != nil {
		return fmt.Errorf("failed to set default address: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrAddressNotFound
	}

	return tx.Commit(ctx)
}
