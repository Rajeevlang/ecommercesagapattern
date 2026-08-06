# Notification Microservice - Design Blueprint & Implementation Guide

This document provides a production-grade blueprint and implementation guide for building the **Notification Microservice** in Go, using Hexagonal Architecture, gRPC, Event-Driven Messaging (Pub/Sub / NATS / Kafka), HTML/Text Email Templating, and PostgreSQL.

---

## 🏛️ 1. System Overview & Purpose

The **Notification Microservice** (`notification`) is responsible for managing and executing all outbound communications (Emails, SMS, Push Notifications) across the e-commerce platform.

### Key Architectural Principle: Post-Saga / Asynchronous Execution
In our Saga-orchestrated e-commerce architecture:
* Notifications are **side-effects** that cannot easily be undone with compensating transactions (you cannot "un-send" an email).
* Therefore, notification processing is kept **outside the core transactional Saga boundary**. 
* If an email delivery fails, it must **never** roll back an order or payment. Instead, notification failures are handled via background retry queues, worker pools, and Dead Letter Queues (DLQ).

---

## 🔔 2. Notification Inventory (What Notifications Are Needed?)

Our e-commerce system requires notifications across 4 core domains:

| Category | Event / Trigger | Delivery Channel | Recipient | Payload Details |
| :--- | :--- | :--- | :--- | :--- |
| **Authentication & Security** | `UserRegistered` | Email | Customer | Welcome email, email verification link. |
| | `PasswordResetRequested` | Email | Customer | Secure password reset token & link. |
| | `SecurityAlert` | Email / SMS | Customer | Login from new device / location. |
| **Order Lifecycle (Saga)** | `OrderCompleted` | Email + SMS | Customer | Order summary, items purchased, total amount, shipping address. |
| | `OrderFailed` | Email | Customer | Failure reason (e.g. payment decline, out-of-stock), link to retry checkout. |
| | `OrderCancelled` | Email | Customer | Order cancellation confirmation & refund notice. |
| **Fulfillment & Logistics** | `ShipmentDispatched` | Email + SMS | Customer | Tracking number, carrier info, estimated delivery date. |
| | `OrderDelivered` | SMS / Push | Customer | Delivery confirmation notice. |
| **Marketing & Engagement** | `BackInStock` | Email / Push | Customer | Product back-in-stock notification. |

---

## 🛠️ 3. Technology Stack & Architectural Choices

| Component | Selected Technology | Rationale |
| :--- | :--- | :--- |
| **Language & Core Framework** | **Go 1.25** | High concurrency (goroutines) for processing bulk notifications asynchronously. |
| **Transport Protocols** | **gRPC** (Direct Sync Calls) + **Event Bus** (NATS JetStream / Kafka / RabbitMQ) | gRPC for direct admin/system triggers (`SendEmail`); Event Bus for consuming domain events (`OrderCompleted`, etc.). |
| **Database (`notification_db`)** | **PostgreSQL** | Stores notification audit logs, delivery attempt history, template definitions, and user channel preferences. |
| **Templating Engine** | Go native `html/template` & `text/template` | Lightweight, secure against injection, and supports pre-compiled templates with dynamic data binding. |
| **Email Channel Provider Adapter** | **SMTP / SendGrid / AWS SES Adapter** (with Mock Provider for Dev) | Adapter interface allows switching from local logging/SMTP to SendGrid/SES without touching domain logic. |
| **SMS Channel Provider Adapter** | **Twilio / AWS SNS Adapter** (with Mock Provider for Dev) | Adapter interface for sending SMS notifications globally. |
| **Resilience & Background Jobs** | **Goroutine Worker Pool** + Exponential Backoff | Prevents third-party provider rate limits from crashing the service; guarantees retry resilience. |
| **Idempotency Control** | PostgreSQL unique constraint on `idempotency_key` | Prevents sending duplicate emails for the same event trigger (e.g., event replay). |

---

## 📂 4. Service Architecture & Hexagonal Directory Layout

```text
notification/
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── cmd/
│   └── server/
│       └── main.go                  # Entry point (gRPC Server + Event Consumer + Worker Pool)
├── config/
│   └── config.go                    # Environment configurations
├── templates/                       # Email & SMS HTML/Text Templates
│   ├── email_welcome.html
│   ├── email_order_confirmation.html
│   ├── email_order_failed.html
│   └── sms_shipment_dispatched.txt
├── migrations/                      # PostgreSQL DDL migrations
│   ├── 000001_create_notifications_table.up.sql
│   └── 000001_create_notifications_table.down.sql
└── internal/
    ├── domain/                      # Domain entities & business errors
    │   ├── notification.go
    │   ├── template.go
    │   └── errors.go
    ├── ports/                       # Interfaces (Contracts)
    │   ├── repository.go
    │   ├── provider.go              # Email/SMS provider interfaces
    │   └── service.go
    ├── repository/                  # Database Adapters
    │   ├── memory/                  # In-memory mock repository (for tests)
    │   └── postgres/                # PostgreSQL repository implementation
    ├── providers/                   # External Provider Implementations
    │   ├── email/                   # Mock, SMTP, SendGrid adapters
    │   └── sms/                     # Mock, Twilio adapters
    ├── service/                     # Core Business Logic / Use Cases
    │   └── notification_service.go
    ├── event/                       # Event Bus Consumer / Subscriber
    │   └── consumer.go
    └── grpc/                        # gRPC Transport Handler
        └── handler.go
```

---

## 📊 5. Database Schema (`notification_db`)

### `migrations/000001_create_notifications_table.up.sql`
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Trigger for updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Notifications Audit & Delivery Log Table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(255) UNIQUE NOT NULL, -- Prevents duplicate notifications for same event
    user_id VARCHAR(255) NOT NULL,
    recipient VARCHAR(255) NOT NULL,              -- Email address or Phone number
    channel VARCHAR(50) NOT NULL,                 -- 'EMAIL', 'SMS', 'PUSH'
    template_name VARCHAR(100) NOT NULL,          -- e.g. 'order_confirmation'
    subject VARCHAR(255) DEFAULT '',
    content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',-- 'PENDING', 'SENT', 'FAILED', 'RETRYING'
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    error_message TEXT DEFAULT '',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_notifications_updated_at
BEFORE UPDATE ON notifications
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Indexes for fast querying & background worker queue polling
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status) WHERE status IN ('PENDING', 'RETRYING');
CREATE INDEX IF NOT EXISTS idx_notifications_idempotency ON notifications(idempotency_key);
```

---

## 📜 6. gRPC Interface Specification (`notification.proto`)

```protobuf
syntax = "proto3";

package ecommerce.notification.v1;

option go_package = "github.com/Rajeevlang/ecommercesagapattern/shared/pb/notification/v1;notificationv1";

import "google/protobuf/timestamp.proto";

service NotificationService {
  // Direct gRPC trigger for sending custom or targeted emails
  rpc SendEmail(SendEmailRequest) returns (SendEmailResponse);
  
  // Direct gRPC trigger for sending SMS
  rpc SendSMS(SendSMSRequest) returns (SendSMSResponse);

  // Queries delivery logs & status
  rpc GetNotificationStatus(GetNotificationStatusRequest) returns (GetNotificationStatusResponse);
}

message SendEmailRequest {
  string recipient_email = 1;
  string subject = 2;
  string body = 3;
  string template_name = 4;
  map<string, string> template_data = 5;
  string idempotency_key = 6;
  string user_id = 7;
}

message SendEmailResponse {
  bool success = 1;
  string notification_id = 2;
}

message SendSMSRequest {
  string recipient_phone = 1;
  string message = 2;
  string idempotency_key = 3;
  string user_id = 4;
}

message SendSMSResponse {
  bool success = 1;
  string notification_id = 2;
}

message GetNotificationStatusRequest {
  string notification_id = 1;
}

message GetNotificationStatusResponse {
  string notification_id = 1;
  string status = 2; // "PENDING", "SENT", "FAILED"
  int32 retry_count = 3;
  string error_message = 4;
  google.protobuf.Timestamp sent_at = 5;
}
```

---

## ⚡ 7. Asynchronous Event Bus Topics & Payloads

The notification service subscribes to messages broadcast over the message broker:

| Event Topic | Trigger Source | Payload Schema | Action Taken |
| :--- | :--- | :--- | :--- |
| `events.order.completed` | Order Saga Orchestrator | `{ order_id, user_id, user_email, total_amount, items: [...] }` | Render `email_order_confirmation.html` and send email. Send SMS summary. |
| `events.order.failed` | Order Saga Orchestrator | `{ order_id, user_id, user_email, reason }` | Render `email_order_failed.html` with error details & retry link. |
| `events.user.registered` | Auth / Account Service | `{ user_id, email, name }` | Render `email_welcome.html` and send welcome email. |
| `events.shipment.dispatched` | Fulfillment Service | `{ order_id, user_id, tracking_number, carrier, phone }` | Render `sms_shipment_dispatched.txt` and send SMS. |

---

## 🔄 8. Retry Logic & Idempotency Strategy

1. **Idempotency Checks**:
   * Every event incoming from the message broker contains an `event_id` or `idempotency_key` (e.g. `order-completed-ORD-99182`).
   * Before attempting delivery, the Notification Service performs an atomic `INSERT INTO notifications (idempotency_key, ...)` with `ON CONFLICT DO NOTHING`.
   * If the insert yields 0 affected rows, the event is flagged as a duplicate and safely skipped.

2. **Background Worker Retry Pool**:
   * Failed network calls to third-party providers (e.g. SendGrid rate limit, SMTP connection drop) mark the notification status as `RETRYING` and increment `retry_count`.
   * A background worker polls for records where `status = 'RETRYING'` with exponential backoff ($2^n \times 5\text{ seconds}$).
   * If `retry_count >= max_retries` (e.g., 3 attempts), the notification is moved to `FAILED` and logged for operational inspection.
