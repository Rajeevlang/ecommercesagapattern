package domain

import "errors"

var (
	ErrProfileNotFound = errors.New("user profile not found")
	ErrAddressNotFound = errors.New("address not found")
	ErrInvalidAddress  = errors.New("invalid address data provided")
)
