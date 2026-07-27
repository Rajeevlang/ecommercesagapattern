# Account Microservice - Design Blueprint & Implementation Guide

> Note: The primary copy of this design document is maintained at [account/service.md](file:///home/rajeev/files/webdev/gprojects/ecommercesagapattern/account/service.md).

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

Refer to [account/service.md](file:///home/rajeev/files/webdev/gprojects/ecommercesagapattern/account/service.md) for full Go code implementation of each package layer.
