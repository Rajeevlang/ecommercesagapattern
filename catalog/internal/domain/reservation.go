package domain

import "time"

type Reservation struct {
	OrderID   string
	ProductID string
	Quantity  int32
	CreatedAt time.Time
}
