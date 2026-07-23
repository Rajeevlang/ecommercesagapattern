# Project Decomposition & Sequential Task Plan

## Overview
The e‑commerce platform is structured as a set of independent **micro‑services** written in Go, orchestrated via a **GraphQL API Gateway**, an **event bus**, and supporting infrastructure (Redis, observability stack, CI/CD, etc.).  Below we split the system into logical sub‑projects and list the core tasks to complete each sub‑project **in order**.

---

## Sub‑projects (sub‑directories)
```
/ecommercesagapattern/
├─ gateway/                # GraphQL API Gateway (gqlgen/chi)
├─ auth/                   # Auth Service (JWT/OAuth2)
├─ account/                # Account Service (Postgres)
├─ catalog/                # Catalog Service (Elasticsearch)
├─ order/                  # Order Service (Postgres)
├─ inventory/              # Inventory Service (Postgres)
├─ payment/                # Payment Service (Postgres + Stripe)
├─ notification/           # Notification Service (email/SMS/push)
├─ cache/                  # Redis client library & utilities
├─ bus/                    # Event Bus abstraction (Kafka/NATS)
├─ observability/          # Prometheus, Jaeger, ELK helpers
└─ infra/                  # Dockerfiles, Helm charts, CI/CD scripts
```

---

## Sequential Task List
### 1️⃣ Project Bootstrap
1. Initialise a **GitHub repository** and push an initial commit.
2. Create a **root `go.mod`** that defines the module name (e.g., `github.com/your-org/ecommercesagapattern`).
3. Add a **Makefile** with common targets (`fmt`, `lint`, `test`, `build`, `docker-build`).
4. Setup **GitHub Actions** workflow for CI (run `go test`, static analysis, build Docker images).

### 2️⃣ Core Infrastructure
1. Define **protobuf definitions** for all gRPC services (shared `proto/` folder).
2. Generate Go stubs with `buf`/`protoc`.
3. Choose an **event‑bus implementation** (Kafka preferred) and create a thin wrapper library under `bus/`.
4. Add **Redis client** code under `cache/` (connections, health‑check helpers).

### 3️⃣ Authentication Service
1. Scaffold the `auth/` module.
2. Implement **JWT issuance & validation** and an OAuth2 flow (Google/Facebook as examples).
3. Write unit tests and integration tests.
4. Expose **gRPC** endpoint for token validation (to be consumed by the gateway).

### 4️⃣ GraphQL API Gateway
1. Initialise the `gateway/` module using **gqlgen**.
2. Wire the gateway to call **auth**, **account**, **catalog**, **order**, etc., via gRPC.
3. Implement **request‑level auth** (extract token, call auth service). 
4. Add **error handling** and **tracing** (OpenTelemetry).
5. Deploy a **local dev Docker compose** to verify end‑to‑end flow.

### 5️⃣ Core Domain Services (one‑by‑one)
#### 5.1 Account Service
- DB schema (users, profiles). 
- CRUD gRPC API. 
- Unit & integration tests.

#### 5.2 Catalog Service
- Index product data into **Elasticsearch**. 
- Search gRPC API (filters, pagination). 
- Sync job to keep Postgres ↔ Elasticsearch in sync.

#### 5.3 Order Service
- Order aggregate model, transactional DB logic. 
- Publish **`OrderCreated`** event to the bus. 
- Idempotency handling for retries.

#### 5.4 Inventory Service
- Reserve / release stock. 
- Subscribe to `OrderCreated` & `PaymentAuthorized` events. 
- Publish **`StockReserved`** event.

#### 5.5 Payment Service
- Integrate **Stripe** SDK (payment intents). 
- Subscribe to `OrderCreated`, emit **`PaymentAuthorized`** or **`PaymentFailed`**.

#### 5.6 Shipping Service
- Manage shipment lifecycle, listen to `StockReserved`. 
- Emit **`OrderShipped`** event.

#### 5.7 Notification Service
- Listen to **all events** on the bus. 
- Dispatch emails/SMS/push via providers (SendGrid, Twilio, Firebase).

### 6️⃣ Observability Stack
1. Add **OpenTelemetry** instrumentation to every service (metrics, traces). 
2. Export metrics to **Prometheus** and traces to **Jaeger**. 
3. Configure **Grafana dashboards** for key KPIs (order rate, latency, error rates). 
4. Set up **ELK** pipeline for log aggregation.

### 7️⃣ Deployment & CI/CD
1. Write **Dockerfiles** for each service (multi‑stage build, minimal runtime image). 
2. Create **Helm charts** under `infra/helm/` for Kubernetes deployment (incl. ConfigMaps, Secrets, PVCs). 
3. Extend GitHub Actions to push images to a container registry and run `helm upgrade --install` on a staging cluster. 
4. Add **canary** or **blue‑green** deployment strategy.

### 8️⃣ End‑to‑End Testing & Documentation
1. Write **integration tests** that spin up the whole stack via Docker Compose. 
2. Generate **API documentation** (GraphQL schema docs, gRPC proto docs). 
3. Draft a **runbook** covering deployment, rollback, and scaling procedures.

---

## How to Use This Plan
- Follow the tasks **in order**; later steps depend on artifacts produced earlier.
- Each bullet is a **stand‑alone work item** you can assign to a ticket (e.g., GitHub Issue or Jira ticket).
- After completing a sub‑project, run the local Docker Compose to verify the integration before moving on.

Feel free to ask for deeper details on any specific task, or let me know if you’d like a more granular breakdown for a particular service.
