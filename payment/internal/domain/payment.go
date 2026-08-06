package domain

import (
	"errors"
	"time"
)

var (
	ErrDuplicatePayment = errors.New("payment for this order already exists")
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrInvalidAmount    = errors.New("invalid payment amount")
)

type PaymentStatus string

const (
	StatusAuthorized PaymentStatus = "AUTHORIZED"
	StatusDeclined   PaymentStatus = "DECLINED"
	StatusRefunded   PaymentStatus = "REFUNDED"
)

type Payment struct {
	ID                 string
	OrderID            string
	UserID             string
	AmountCents        int64
	PaymentMethodToken string
	Status             PaymentStatus
	ErrorMessage       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
