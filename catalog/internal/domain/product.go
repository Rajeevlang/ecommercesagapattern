package domain

import "time"

type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Stock       int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
