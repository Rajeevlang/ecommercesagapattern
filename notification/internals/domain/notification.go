package domain

import (
	"errors"
	"time"
)

var (
	ErrDuplicateNotification = errors.New("duplicate notification event")
	ErrNotificationNotFound  = errors.New("notification not found")
)

type NotificationChannel string
type NotificationStatus string

const (
	ChannelEmail NotificationChannel = "EMAIL"
	ChannelSMS   NotificationChannel = "SMS"
	ChannelPush  NotificationChannel = "PUSH"

	StatusPending  NotificationStatus = "PENDING"
	StatusSent     NotificationStatus = "SENT"
	StatusFailed   NotificationStatus = "FAILED"
	StatusRetrying NotificationStatus = "RETRYING"
)

type Notification struct {
	ID             string
	IdempotencyKey string
	UserID         string
	Recipient      string
	Channel        NotificationChannel
	TemplateName   string
	Subject        string
	Content        string
	Status         NotificationStatus
	RetryCount     int32
	MaxRetries     int32
	ErrorMessage   string
	SentAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
