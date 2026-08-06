package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/domain"
	"github.com/Rajeevlang/ecommercesagapattern/notification/internals/ports"
)

type NotificationService struct {
	repo          ports.NotificationRepository
	emailProvider ports.EmailProvider
	smsProvider   ports.SMSProvider
	stopChan      chan struct{}
}

func NewNotificationService(
	repo ports.NotificationRepository,
	emailProvider ports.EmailProvider,
	smsProvider ports.SMSProvider,
) *NotificationService {
	return &NotificationService{
		repo:          repo,
		emailProvider: emailProvider,
		smsProvider:   smsProvider,
		stopChan:      make(chan struct{}),
	}
}

func (s *NotificationService) SendEmail(
	ctx context.Context,
	userID, recipient, subject, body, templateName, idempotencyKey string,
) (*domain.Notification, error) {
	// 1. Check idempotency
	existing, err := s.repo.GetNotificationByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		// Already exists
		if existing.Status == domain.StatusSent || existing.Status == domain.StatusPending {
			return existing, nil
		}
		// If failed or retrying, we might trigger immediate retry or just let background worker pick it up
		return existing, nil
	}

	if !errors.Is(err, domain.ErrNotificationNotFound) {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// 2. Create notification record
	n := &domain.Notification{
		IdempotencyKey: idempotencyKey,
		UserID:         userID,
		Recipient:      recipient,
		Channel:        domain.ChannelEmail,
		TemplateName:   templateName,
		Subject:        subject,
		Content:        body,
		Status:         domain.StatusPending,
		RetryCount:     0,
		MaxRetries:     3,
	}

	if err := s.repo.CreateNotification(ctx, n); err != nil {
		if errors.Is(err, domain.ErrDuplicateNotification) {
			// Handled concurrent write
			return s.repo.GetNotificationByIdempotencyKey(ctx, idempotencyKey)
		}
		return nil, fmt.Errorf("failed to log notification: %w", err)
	}

	// 3. Attempt delivery
	deliveryErr := s.emailProvider.SendEmail(ctx, n.Recipient, n.Subject, n.Content)
	if deliveryErr != nil {
		n.Status = domain.StatusRetrying
		n.ErrorMessage = deliveryErr.Error()
		n.RetryCount++
		_ = s.repo.UpdateNotification(ctx, n)
		return n, fmt.Errorf("initial email delivery failed: %w", deliveryErr)
	}

	// 4. Success
	now := time.Now()
	n.Status = domain.StatusSent
	n.SentAt = &now
	n.ErrorMessage = ""
	_ = s.repo.UpdateNotification(ctx, n)

	return n, nil
}

func (s *NotificationService) SendSMS(
	ctx context.Context,
	userID, recipient, message, idempotencyKey string,
) (*domain.Notification, error) {
	// 1. Check idempotency
	existing, err := s.repo.GetNotificationByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		if existing.Status == domain.StatusSent || existing.Status == domain.StatusPending {
			return existing, nil
		}
		return existing, nil
	}

	if !errors.Is(err, domain.ErrNotificationNotFound) {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// 2. Create notification record
	n := &domain.Notification{
		IdempotencyKey: idempotencyKey,
		UserID:         userID,
		Recipient:      recipient,
		Channel:        domain.ChannelSMS,
		TemplateName:   "sms_direct",
		Subject:        "",
		Content:        message,
		Status:         domain.StatusPending,
		RetryCount:     0,
		MaxRetries:     3,
	}

	if err := s.repo.CreateNotification(ctx, n); err != nil {
		if errors.Is(err, domain.ErrDuplicateNotification) {
			return s.repo.GetNotificationByIdempotencyKey(ctx, idempotencyKey)
		}
		return nil, fmt.Errorf("failed to log notification: %w", err)
	}

	// 3. Attempt delivery
	deliveryErr := s.smsProvider.SendSMS(ctx, n.Recipient, n.Content)
	if deliveryErr != nil {
		n.Status = domain.StatusRetrying
		n.ErrorMessage = deliveryErr.Error()
		n.RetryCount++
		_ = s.repo.UpdateNotification(ctx, n)
		return n, fmt.Errorf("initial SMS delivery failed: %w", deliveryErr)
	}

	// 4. Success
	now := time.Now()
	n.Status = domain.StatusSent
	n.SentAt = &now
	n.ErrorMessage = ""
	_ = s.repo.UpdateNotification(ctx, n)

	return n, nil
}

func (s *NotificationService) GetNotificationStatus(ctx context.Context, id string) (*domain.Notification, error) {
	return s.repo.GetNotification(ctx, id)
}

// StartRetryWorker runs a background worker loop that periodically retries PENDING/RETRYING notifications.
func (s *NotificationService) StartRetryWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		log.Println("Starting notification retry background worker...")
		for {
			select {
			case <-ticker.C:
				s.processRetries()
			case <-s.stopChan:
				ticker.Stop()
				log.Println("Notification retry background worker stopped.")
				return
			}
		}
	}()
}

func (s *NotificationService) StopRetryWorker() {
	close(s.stopChan)
}

func (s *NotificationService) processRetries() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	list, err := s.repo.ListPendingOrRetrying(ctx)
	if err != nil {
		log.Printf("Retry worker: failed to list notifications to retry: %v\n", err)
		return
	}

	for _, n := range list {
		// Simple exponential backoff check
		// backoff = 5 seconds * 2^retry_count
		backoff := time.Second * 5 * (1 << uint(n.RetryCount))
		if time.Since(n.UpdatedAt) < backoff {
			continue
		}

		log.Printf("Retry worker: retrying notification %s (attempt %d/%d)...\n", n.ID, n.RetryCount+1, n.MaxRetries)

		var deliveryErr error
		if n.Channel == domain.ChannelEmail {
			deliveryErr = s.emailProvider.SendEmail(ctx, n.Recipient, n.Subject, n.Content)
		} else if n.Channel == domain.ChannelSMS {
			deliveryErr = s.smsProvider.SendSMS(ctx, n.Recipient, n.Content)
		}

		if deliveryErr != nil {
			n.RetryCount++
			n.ErrorMessage = deliveryErr.Error()
			if n.RetryCount >= n.MaxRetries {
				n.Status = domain.StatusFailed
				log.Printf("Retry worker: notification %s failed permanently: %v\n", n.ID, deliveryErr)
			} else {
				n.Status = domain.StatusRetrying
			}
			_ = s.repo.UpdateNotification(ctx, n)
		} else {
			now := time.Now()
			n.Status = domain.StatusSent
			n.SentAt = &now
			n.ErrorMessage = ""
			_ = s.repo.UpdateNotification(ctx, n)
			log.Printf("Retry worker: notification %s sent successfully\n", n.ID)
		}
	}
}
