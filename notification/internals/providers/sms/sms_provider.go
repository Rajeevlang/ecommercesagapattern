package sms

import (
	"context"
	"log"
)

type MockSMSProvider struct{}

func NewMockSMSProvider() *MockSMSProvider {
	return &MockSMSProvider{}
}

func (p *MockSMSProvider) SendSMS(ctx context.Context, recipient, message string) error {
	log.Printf("[MOCK SMS] To: %s | Message: %s\n", recipient, message)
	return nil
}
