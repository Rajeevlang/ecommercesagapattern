package domain

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrDatabase           = errors.New("database operation failed")
)

