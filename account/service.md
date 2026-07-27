# Account Microservice - Design Blueprint & Implementation Guide

This document provides a production-grade, step-by-step implementation guide for building the **Account Microservice** in Go, using Hexagonal Architecture, gRPC, and PostgreSQL.

---

## 🏛️ 1. Architecture Overview & Design Rationale

### 1.1 Service Boundaries
* **Auth Service (`auth_db`)**: Responsible strictly for authentication, credentials, password verification, and issuing JWT tokens.
* **Account Service (`account_db`)**: Responsible for rich user domain profiles, multiple shipping/billing addresses, user avatars, and preferences.

### 1.2 Hexagonal Project Layout
```text
account/
├── Makefile 
├── Dockerfile  
├── docker-compose.yml 
├── go.mod
├── cmd/
│   └── server/
│       └── main.go                  # Application entry point & DI wiring
├── config/
│   └── config.go                    # Environment configurations
├── migrations/                      # PostgreSQL DDL migrations
│   ├── 000001_create_profiles_and_addresses.up.sql
│   └── 000001_create_profiles_and_addresses.down.sql
└── internal/
    ├── domain/                      # Domain entities & business errors 
    │   ├── profile.go
    │   ├── address.go
    │   └── errors.go
    ├── ports/                       # Interfaces (Contracts)
    │   ├── repository.go
    │   └── service.go
    ├── repository/                  # Database Adapters
    │   └── memory/                  # In-memory mock repository (for tests)
    │   └── postgres/                # PostgreSQL repository implementation
    ├── service/                     # Core Business Logic / Use Cases
    │   └── account_service.go
    └── grpc/                        # gRPC Transport Handler
        └── handler.go
```

---

## 📊 2. Database Schema (PostgreSQL `account_db`)

### `migrations/000001_create_profiles_and_addresses.up.sql`
```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Trigger for updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- User Profiles Table
CREATE TABLE IF NOT EXISTS profiles (
    user_id UUID PRIMARY KEY, -- Foreign key matching Auth Service user_id
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) DEFAULT '',
    avatar_url TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_profiles_updated_at
BEFORE UPDATE ON profiles
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- User Addresses Table (Supports Multiple Shipping/Billing Addresses)
CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES profiles(user_id) ON DELETE CASCADE,
    street VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL,
    zip_code VARCHAR(20) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    address_type VARCHAR(20) NOT NULL DEFAULT 'SHIPPING', -- 'SHIPPING' or 'BILLING'
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_addresses_updated_at
BEFORE UPDATE ON addresses
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Indexes for Fast Querying
CREATE INDEX IF NOT EXISTS idx_addresses_user_id ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_addresses_user_default ON addresses(user_id, is_default, address_type);
```

---

## 💻 3. Step-by-Step Code Implementation

### Step 1: Go Module Config (`account/go.mod`)
```go
module github.com/Rajeevlang/ecommercesagapattern/account

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
)
```

---

### Step 2: Configuration (`account/config/config.go`)
```go
package config

import "os"

type Config struct {
	GRPCPort    string
	DatabaseURL string
	Environment string
}

func LoadConfig() *Config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50052"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root:secretpassword@localhost:5432/account_db?sslmode=disable"
	}

	return &Config{
		GRPCPort:    port,
		DatabaseURL: dbURL,
		Environment: "development",
	}
}
```

---

### Step 3: Domain Models & Errors

#### `account/internal/domain/profile.go`
```go
package domain

import "time"

type Profile struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

#### `account/internal/domain/address.go`
```go
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
```

#### `account/internal/domain/errors.go`
```go
package domain

import "errors"

var (
	ErrProfileNotFound = errors.New("user profile not found")
	ErrAddressNotFound = errors.New("address not found")
	ErrInvalidAddress  = errors.New("invalid address data provided")
)
```

---

### Step 4: Ports (Interfaces)

#### `account/internal/ports/repository.go`
```go
package ports

import (
	"context"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
)

type AccountRepository interface {
	CreateProfile(ctx context.Context, profile *domain.Profile) error
	GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, profile *domain.Profile) error
	
	CreateAddress(ctx context.Context, address *domain.Address) error
	GetAddressByID(ctx context.Context, addressID string) (*domain.Address, error)
	ListAddressesByUserID(ctx context.Context, userID string) ([]*domain.Address, error)
	DeleteAddress(ctx context.Context, userID, addressID string) error
	SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error
}
```

#### `account/internal/ports/service.go`
```go
package ports

import (
	"context"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
)

type AccountService interface {
	GetProfile(ctx context.Context, userID string) (*domain.Profile, *domain.Address, error)
	UpdateProfile(ctx context.Context, userID, name, phone, avatarURL string) error
	CreateAddress(ctx context.Context, address *domain.Address) (*domain.Address, error)
	ListAddresses(ctx context.Context, userID string) ([]*domain.Address, error)
	DeleteAddress(ctx context.Context, userID, addressID string) error
	SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error
}
```

---

### Step 5: Service Implementation (`account/internal/service/account_service.go`)
```go
package service

import (
	"context"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/ports"
	"github.com/google/uuid"
)

type AccountService struct {
	repo ports.AccountRepository
}

func NewAccountService(repo ports.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) GetProfile(ctx context.Context, userID string) (*domain.Profile, *domain.Address, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	addresses, _ := s.repo.ListAddressesByUserID(ctx, userID)
	var defaultAddress *domain.Address
	for _, addr := range addresses {
		if addr.IsDefault && addr.AddressType == "SHIPPING" {
			defaultAddress = addr
			break
		}
	}

	return profile, defaultAddress, nil
}

func (s *AccountService) UpdateProfile(ctx context.Context, userID, name, phone, avatarURL string) error {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		// Auto-provision profile if missing
		profile = &domain.Profile{
			UserID:    userID,
			Name:      name,
			Phone:     phone,
			AvatarURL: avatarURL,
		}
		return s.repo.CreateProfile(ctx, profile)
	}

	profile.Name = name
	profile.Phone = phone
	profile.AvatarURL = avatarURL
	return s.repo.UpdateProfile(ctx, profile)
}

func (s *AccountService) CreateAddress(ctx context.Context, address *domain.Address) (*domain.Address, error) {
	if address.Street == "" || address.City == "" || address.Country == "" {
		return nil, domain.ErrInvalidAddress
	}

	address.ID = uuid.New().String()
	if address.AddressType == "" {
		address.AddressType = "SHIPPING"
	}

	if err := s.repo.CreateAddress(ctx, address); err != nil {
		return nil, err
	}

	if address.IsDefault {
		_ = s.repo.SetDefaultAddress(ctx, address.UserID, address.ID, address.AddressType)
	}

	return address, nil
}

func (s *AccountService) ListAddresses(ctx context.Context, userID string) ([]*domain.Address, error) {
	return s.repo.ListAddressesByUserID(ctx, userID)
}

func (s *AccountService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	return s.repo.DeleteAddress(ctx, userID, addressID)
}

func (s *AccountService) SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error {
	return s.repo.SetDefaultAddress(ctx, userID, addressID, addressType)
}
```

---

### Step 6: Memory Repository Implementation (`account/internal/repository/memory/account_repository.go`)
```go
package memory

import (
	"context"
	"sync"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
)

type InMemoryAccountRepository struct {
	mu        sync.RWMutex
	profiles  map[string]*domain.Profile
	addresses map[string]*domain.Address // keyed by address.ID
}

func NewInMemoryAccountRepository() *InMemoryAccountRepository {
	return &InMemoryAccountRepository{
		profiles:  make(map[string]*domain.Profile),
		addresses: make(map[string]*domain.Address),
	}
}

func (r *InMemoryAccountRepository) CreateProfile(ctx context.Context, profile *domain.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.UserID] = profile
	return nil
}

func (r *InMemoryAccountRepository) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[userID]
	if !ok {
		return nil, domain.ErrProfileNotFound
	}
	return p, nil
}

func (r *InMemoryAccountRepository) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.UserID] = profile
	return nil
}

func (r *InMemoryAccountRepository) CreateAddress(ctx context.Context, address *domain.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addresses[address.ID] = address
	return nil
}

func (r *InMemoryAccountRepository) GetAddressByID(ctx context.Context, addressID string) (*domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.addresses[addressID]
	if !ok {
		return nil, domain.ErrAddressNotFound
	}
	return a, nil
}

func (r *InMemoryAccountRepository) ListAddressesByUserID(ctx context.Context, userID string) ([]*domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*domain.Address
	for _, a := range r.addresses {
		if a.UserID == userID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (r *InMemoryAccountRepository) DeleteAddress(ctx context.Context, userID, addressID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.addresses, addressID)
	return nil
}

func (r *InMemoryAccountRepository) SetDefaultAddress(ctx context.Context, userID, addressID, addressType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.addresses {
		if a.UserID == userID && a.AddressType == addressType {
			a.IsDefault = (a.ID == addressID)
		}
	}
	return nil
}
```

---

### Step 7: gRPC Transport Layer (`account/internal/grpc/handler.go`)
```go
package grpc

import (
	"context"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/ports"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountGRPCHandler struct {
	accountv1.UnimplementedAccountServiceServer
	service ports.AccountService
}

func NewAccountGRPCHandler(service ports.AccountService) *AccountGRPCHandler {
	return &AccountGRPCHandler{service: service}
}

func (h *AccountGRPCHandler) GetProfile(ctx context.Context, req *accountv1.GetProfileRequest) (*accountv1.GetProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	profile, defaultAddr, err := h.service.GetProfile(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	res := &accountv1.GetProfileResponse{
		UserId:    profile.UserID,
		Email:     profile.Email,
		Name:      profile.Name,
		Phone:     profile.Phone,
		AvatarUrl: profile.AvatarURL,
	}

	if defaultAddr != nil {
		res.DefaultShippingAddress = &accountv1.Address{
			Id:          defaultAddr.ID,
			UserId:      defaultAddr.UserID,
			Street:      defaultAddr.Street,
			City:        defaultAddr.City,
			State:       defaultAddr.State,
			Country:     defaultAddr.Country,
			ZipCode:     defaultAddr.ZipCode,
			IsDefault:   defaultAddr.IsDefault,
			AddressType: defaultAddr.AddressType,
		}
	}

	return res, nil
}

func (h *AccountGRPCHandler) UpdateProfile(ctx context.Context, req *accountv1.UpdateProfileRequest) (*accountv1.UpdateProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := h.service.UpdateProfile(ctx, req.GetUserId(), req.GetName(), req.GetPhone(), req.GetAvatarUrl())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &accountv1.UpdateProfileResponse{Success: true, UserId: req.GetUserId()}, nil
}

func (h *AccountGRPCHandler) CreateAddress(ctx context.Context, req *accountv1.CreateAddressRequest) (*accountv1.CreateAddressResponse, error) {
	addr := &domain.Address{
		UserID:      req.GetUserId(),
		Street:      req.GetStreet(),
		City:        req.GetCity(),
		State:       req.GetState(),
		Country:     req.GetCountry(),
		ZipCode:     req.GetZipCode(),
		IsDefault:   req.GetIsDefault(),
		AddressType: req.GetAddressType(),
	}

	created, err := h.service.CreateAddress(ctx, addr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &accountv1.CreateAddressResponse{
		Address: &accountv1.Address{
			Id:          created.ID,
			UserId:      created.UserID,
			Street:      created.Street,
			City:        created.City,
			State:       created.State,
			Country:     created.Country,
			ZipCode:     created.ZipCode,
			IsDefault:   created.IsDefault,
			AddressType: created.AddressType,
		},
	}, nil
}

func (h *AccountGRPCHandler) ListAddresses(ctx context.Context, req *accountv1.ListAddressesRequest) (*accountv1.ListAddressesResponse, error) {
	addresses, err := h.service.ListAddresses(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbAddrs []*accountv1.Address
	for _, a := range addresses {
		pbAddrs = append(pbAddrs, &accountv1.Address{
			Id:          a.ID,
			UserId:      a.UserID,
			Street:      a.Street,
			City:        a.City,
			State:       a.State,
			Country:     a.Country,
			ZipCode:     a.ZipCode,
			IsDefault:   a.IsDefault,
			AddressType: a.AddressType,
		})
	}

	return &accountv1.ListAddressesResponse{Addresses: pbAddrs}, nil
}
```

---

### Step 8: Main Server Entrypoint (`account/cmd/server/main.go`)
```go
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rajeevlang/ecommercesagapattern/account/config"
	accountgrpc "github.com/Rajeevlang/ecommercesagapattern/account/internal/grpc"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/repository/memory"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/service"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Account Microservice on port %s...", cfg.GRPCPort)

	repo := memory.NewInMemoryAccountRepository()
	accountSvc := service.NewAccountService(repo)
	grpcHandler := accountgrpc.NewAccountGRPCHandler(accountSvc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	accountv1.RegisterAccountServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Account gRPC Server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Account gRPC Server...")
	grpcServer.GracefulStop()
	log.Println("Account Server stopped.")
}
```
