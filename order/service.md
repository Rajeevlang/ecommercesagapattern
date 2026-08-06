# Catalog Microservice - Design Blueprint & Implementation Guide

This document details the design, project layout, PostgreSQL database schema, and implementation strategy for the **Catalog Microservice**.

---

## 🏛️ 1. Architecture Overview & Hexagonal Project Layout

To maintain architectural consistency across our monorepo (matching the `account` and `order` services), the `catalog` service is structured using **Hexagonal Architecture (Ports and Adapters)**:

```text
catalog/
├── Makefile
├── go.mod
├── cmd/
│   └── server/
│       └── main.go                  # Application entry point & DI wiring
├── config/
│   └── config.go                    # Environment & database configurations
├── migrations/                      # PostgreSQL DDL migrations
│   ├── 000001_create_products_and_reservations.up.sql
│   └── 000001_create_products_and_reservations.down.sql
└── internal/
    ├── domain/                      # Domain entities & business logic models
    │   ├── product.go
    │   └── reservation.go
    ├── ports/                       # Service & Repository interfaces (Contracts)
    │   ├── repository.go
    │   └── service.go
    ├── repository/                  # Database Adapters (implementations)
    │   └── postgres/                # PostgreSQL pgxpool queries
    ├── service/                     # Core business logic / Use cases
    │   └── catalog_service.go
    └── grpc/                        # gRPC Transport handlers
        └── handler.go
```

---

## 📊 2. Database Schema (PostgreSQL `catalog_db`)

The database requires two primary tables:
1. `products`: To store catalog listings, pricing, and available stock levels.
2. `stock_reservations`: To track locked inventory allocated to pending orders. This is **critical** for achieving idempotency and managing Saga compensating rollbacks.

### `migrations/000001_create_products_and_reservations.up.sql`
```sql
-- 1. Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Trigger function to auto-update 'updated_at' column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 3. Products Table
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0), -- Represented in integer cents
    stock INT NOT NULL DEFAULT 0 CHECK (stock >= 0),     -- Quantity available for sale
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Trigger for auto updating updated_at in products
DROP TRIGGER IF EXISTS set_products_updated_at ON products;
CREATE TRIGGER set_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 4. Stock Reservations Table (For Saga Lock Management)
CREATE TABLE IF NOT EXISTS stock_reservations (
    order_id UUID NOT NULL,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (order_id, product_id)
);

-- 5. Indexes for fast query operations
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_order_id ON stock_reservations(order_id);
```

### `migrations/000001_create_products_and_reservations.down.sql`
```sql
DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS products;
```

---

## ⚙️ 3. Saga Transaction Implementations (Concurrency & Idempotency)

For the Saga pattern to be safe and resilient under high concurrency, we must address two risks: **race conditions** (double allocation of stock) and **network retries** (duplicate reservation requests).

### A. Stock Reservation Flow (`ReserveStock`)
When the Saga Orchestrator initiates a reservation:
1. **Check Idempotency:** Query `stock_reservations` to see if a reservation for `order_id` already exists. If yes, return `success = true` (noop).
2. **Lock & Verify (Transaction):** 
   * Begin a PostgreSQL transaction.
   * Lock the product row using `SELECT stock FROM products WHERE id = $1 FOR UPDATE` to block concurrent updates.
   * Verify if `stock >= quantity`. If not, abort the transaction and return an error ("out of stock").
   * Decrement the `stock` column in `products`.
   * Insert a new reservation record into `stock_reservations`.
   * Commit the transaction.

```sql
-- Conceptual SQL sequence inside pgx transaction:
-- 1. Check existing
SELECT 1 FROM stock_reservations WHERE order_id = $1; 

-- 2. Lock row
SELECT stock FROM products WHERE id = $2 FOR UPDATE;

-- 3. Update stock & insert reservation
UPDATE products SET stock = stock - $3 WHERE id = $2;
INSERT INTO stock_reservations (order_id, product_id, quantity) VALUES ($1, $2, $3);
```

### B. Stock Release Flow (`ReleaseStock` - Compensating Action)
If a subsequent Saga step (e.g. Payment) fails, the Orchestrator rolls back the transaction:
1. **Check Reservation:** Query `stock_reservations` for `order_id`. If no reservation exists, return `success = true` (noop/idempotent safety).
2. **Restore Stock (Transaction):**
   * Begin a transaction.
   * For each reserved product, increment the available `stock` in `products` by the reserved `quantity`.
   * Delete the corresponding reservation records from `stock_reservations`.
   * Commit the transaction.

---

## 🚀 4. Step-by-Step Implementation Roadmap
When we begin coding the **Catalog Service**, we will follow this phased approach:
1. **Scaffold Directory Structure:** Create the internal folders (`internal/domain`, `ports`, `repository/postgres`, `service`, `grpc`).
2. **Apply Database Migrations:** Run the Postgres SQL schema to initialize tables and indexes.
3. **Write Domain Models:** Define `Product` and `Reservation` model in `internal/domain`.
4. **Implement PostgreSQL Adapter:** Write SQL query transactions in `internal/repository/postgres` using `pgxpool`.
5. **Implement Catalog Service Logic:** Write the business service implementing `ReserveStock` and `ReleaseStock`.
6. **Implement gRPC Handler:** Expose these methods over gRPC using generated proto stubs.
