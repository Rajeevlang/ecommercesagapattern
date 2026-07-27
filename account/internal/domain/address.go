package domain

import "time"

type Address struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Street      string    `json:"street"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Country     string    `json:"country"`
	ZipCode     string    `json:"zip_code"`
	IsDefault   bool      `json:"is_default"`
	AddressType string    `json:"address_type"` // "SHIPPING" or "BILLING"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
