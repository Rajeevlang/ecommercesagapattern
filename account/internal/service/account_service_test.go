package service_test

import (
	"context"
	"testing"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/service"
)

func TestAccountService_ProfileAndAddressLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewInMemoryAccountRepository()
	svc := service.NewAccountService(repo)

	userID := "usr-12345"

	// 1. Update/Create Profile
	err := svc.UpdateProfile(ctx, userID, "Alice Smith", "+1234567890", "https://example.com/avatar.png")
	if err != nil {
		t.Fatalf("unexpected error updating profile: %v", err)
	}

	// 2. Fetch Profile
	profile, defaultAddr, err := svc.GetProfile(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error getting profile: %v", err)
	}
	if profile.Name != "Alice Smith" {
		t.Errorf("expected name 'Alice Smith', got '%s'", profile.Name)
	}
	if defaultAddr != nil {
		t.Errorf("expected default shipping address to be nil initially, got %v", defaultAddr)
	}

	// 3. Create Address
	addrInput := &domain.Address{
		UserID:      userID,
		Street:      "123 Main St",
		City:        "Tech City",
		State:       "CA",
		Country:     "USA",
		ZipCode:     "90001",
		IsDefault:   true,
		AddressType: "SHIPPING",
	}

	createdAddr, err := svc.CreateAddress(ctx, addrInput)
	if err != nil {
		t.Fatalf("unexpected error creating address: %v", err)
	}
	if createdAddr.ID == "" {
		t.Error("expected generated address ID")
	}

	// 4. Verify GetProfile includes default shipping address
	profile, defaultAddr, err = svc.GetProfile(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error getting profile after address creation: %v", err)
	}
	if defaultAddr == nil {
		t.Fatal("expected default shipping address, got nil")
	}
	if defaultAddr.Street != "123 Main St" {
		t.Errorf("expected street '123 Main St', got '%s'", defaultAddr.Street)
	}

	// 5. List Addresses
	addrs, err := svc.ListAddresses(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error listing addresses: %v", err)
	}
	if len(addrs) != 1 {
		t.Errorf("expected 1 address, got %d", len(addrs))
	}

	// 6. Delete Address
	err = svc.DeleteAddress(ctx, userID, createdAddr.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting address: %v", err)
	}

	addrs, _ = svc.ListAddresses(ctx, userID)
	if len(addrs) != 0 {
		t.Errorf("expected 0 addresses after delete, got %d", len(addrs))
	}
}
