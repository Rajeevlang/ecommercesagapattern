# E-Commerce Microservices Project Status & Phased Plan

This document outlines the sequential phases for building our e-commerce platform using Go microservices, GraphQL Gateway, gRPC, Saga orchestration, and modern DevOps practices.

---

## 📊 High-Level Roadmap

```mermaid
gantt
    title E-Commerce Microservices Roadmap
    dateFormat  YYYY-MM-DD
    section Core Infrastructure
    Phase 1: Bootstrapping & Gateway Setup :active, p1, 2026-07-23, 2d
    Phase 2: Authentication & Middleware  : p2, after p1, 4d
    section Domain Services
    Phase 3: Catalog Service & Search      : p3, after p2, 5d
    Phase 4: Order Service & Saga Orchestration : p4, after p3, 7d
    Phase 5: Payment & Inventory Integrations   : p5, after p4, 5d
    section Ops & Hardening
    Phase 6: Notifications & Event Bus    : p6, after p5, 4d
    Phase 7: Observability & DevOps CI/CD  : p7, after p6, 6d
```

---

## 🛠️ Phase-by-Phase Breakdown

### Phase 1: Bootstrapping & Gateway Setup
* **Objective**: Create a robust multi-module workspace and scaffold the entry point (API Gateway).
* **Tasks**:
  - [x] Clean up duplicate Protobuf schema files under `shared/protofiles/`.
  - [x] Configure Go Workspaces (`go.work`) with `./shared` and `./apigateway`.
  - [x] Compile `.proto` files into Go gRPC stubs inside `shared/pb/` using `make gen`.
  - [x] Bootstrap the GraphQL Gateway using `gqlgen`.
  - [x] Define the production-ready GraphQL schema (`schema.graphqls`).
* **Status**: **Completed** (Baseline ready).

---

### Phase 2: Authentication & Middleware
* **Objective**: Establish secure communications and set up the foundation of the web gateway.
* **Tasks**:
  - [x] Scaffold the `auth/` microservice module (gRPC server).
  - [x] Implement JWT token signing and verification inside `auth/`.
  - [ ] Refactor the Gateway (`apigateway/server.go`) to use the **Chi Router** instead of the standard library router.
  - [ ] Write a custom HTTP Middleware in the Gateway to extract JWT tokens from incoming headers.
  - [ ] Wire the Gateway to talk to the Auth Service via gRPC to validate tokens and inject the `userID` into the request context.
* **Status**: **In Progress**.

---

### Phase 3: Catalog Service & Search
* **Objective**: Build the product catalog and pagination systems.
* **Tasks**:
  - [ ] Scaffold the `catalog/` microservice module (gRPC server).
  - [ ] Integrate **PostgreSQL** to store product listings.
  - [ ] Integrate **Elasticsearch** (or simple database indices) for full-text search.
  - [ ] Implement `GetProduct` and `ListProducts` gRPC endpoints.
  - [ ] Wire the GraphQL Gateway product queries (`product`, `products`) to communicate with the Catalog Service gRPC client.
  - [ ] Implement the **Relay Connection Pattern** for cursor-based pagination in the resolver.
* **Status**: **Planned**.

---

### Phase 4: Order Service & Saga Orchestration
* **Objective**: Implement the transactional core and start the Saga distributed transaction flow.
* **Tasks**:
  - [ ] Scaffold the `order/` microservice module (gRPC server + DB).
  - [ ] Design the Order database schema (orders, order items).
  - [ ] Implement the **Saga Orchestrator** pattern inside the Order Service.
  - [ ] Implement the success flow: Create Order (Pending) $\rightarrow$ Reserve Inventory.
  - [ ] Implement the compensation flows: If inventory reservation fails, cancel the order and update status to `FAILED`.
  - [ ] Implement gRPC client communication from Gateway to Order Service to initiate this Saga.
* **Status**: **Planned**.

---

### Phase 5: Payment, Inventory & Compensating Actions
* **Objective**: Add transactional safety layers and external integrations.
* **Tasks**:
  - [ ] Scaffold the `payment/` microservice module (gRPC server).
  - [ ] Integrate **Stripe SDK** in payment service using secure payment tokens.
  - [ ] Update the Saga Orchestrator in Order Service to call the Payment Service after stock is reserved.
  - [ ] Implement compensating actions for payment failures (releasing reserved stock in Catalog Service).
  - [ ] Implement **Idempotency Keys** on payment and stock reservation endpoints to prevent double-charging/double-allocation.
* **Status**: **Planned**.

---

### Phase 6: Notifications & Event-Driven Subsystems
* **Objective**: Decouple post-transaction tasks using an asynchronous event bus.
* **Tasks**:
  - [ ] Set up **Kafka** (or **NATS JetStream**) as our local message broker.
  - [ ] Emit events (`OrderCompleted`, `OrderFailed`, `ShipmentDispatched`) from the Saga Orchestrator to the event bus.
  - [ ] Scaffold the `notification/` microservice.
  - [ ] Make the notification service subscribe to these events and send simulated emails/SMS (e.g., using SendGrid/Twilio mock setups).
* **Status**: **Planned**.

---

### Phase 7: Observability & DevOps (CI/CD)
* **Objective**: Hardening for production, containerization, and automated deployments.
* **Tasks**:
  - [ ] Integrate **OpenTelemetry** instrumentation across all services for distributed tracing (Jaeger) and metrics (Prometheus).
  - [ ] Write multi-stage, production-optimized **Dockerfiles** for each service to keep image sizes minimal.
  - [ ] Create a comprehensive `docker-compose.yml` file to run the entire stack locally with databases, cache, and broker.
  - [ ] Create **Helm Charts** for Kubernetes deployment.
  - [ ] Setup a **GitHub Actions** CI/CD pipeline that:
    - Runs Go test suites and linters.
    - Builds and pushes Docker images to a container registry.
    - Deploys changes to a cloud Kubernetes cluster (e.g. AWS EKS, DigitalOcean Kubernetes).
* **Status**: **Planned**.

---

## 🎯 Next Immediate Step
When we resume, we will begin **Phase 2** by scaffolding the **Auth Microservice** and linking its gRPC client to the API Gateway.
