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
    Phase 2: Authentication & Middleware  :active, p2, after p1, 4d
    section Domain Services
    Phase 3: Catalog Service & Search      :active, p3, after p2, 5d
    Phase 4: Order Service & Saga Orchestration :active, p4, after p3, 7d
    Phase 5: Payment & Inventory Integrations   :active, p5, after p4, 5d
    section Ops & Hardening
    Phase 6: Notifications & Event Bus    :active, p6, after p5, 4d
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
  - [x] Refactor the Gateway (`apigateway/server.go`) to use the **Chi Router** instead of the standard library router.
  - [x] Write a custom HTTP Middleware in the Gateway to extract JWT tokens from incoming headers.
  - [x] Wire the Gateway to talk to the Auth Service via gRPC to validate tokens and inject the `userID` into the request context.
* **Status**: **Completed** (Gateway routing, JWT authentication middleware, and resolvers fully wired).

---

### Phase 3: Catalog Service & Search
* **Objective**: Build the product catalog and pagination systems.
* **Tasks**:
  - [x] Scaffold the `catalog/` microservice module (gRPC server).
  - [x] Integrate **PostgreSQL** to store product listings.
  - [ ] Integrate **Elasticsearch** (or simple database indices) for full-text search.
  - [x] Implement `GetProduct` and `ListProducts` gRPC endpoints.
  - [x] Wire the GraphQL Gateway product queries (`product`, `products`) to communicate with the Catalog Service gRPC client.
  - [x] Implement the **Relay Connection Pattern** for cursor-based pagination in the resolver.
* **Status**: **Completed** (Services, gateway clients, and connection resolvers wired; database full-text search is next).

---

### Phase 4: Order Service & Saga Orchestration
* **Objective**: Implement the transactional core and start the Saga distributed transaction flow.
* **Tasks**:
  - [x] Scaffold the `order/` microservice module (gRPC server + DB).
  - [x] Design the Order database schema (orders, order items).
  - [x] Implement the **Saga Orchestrator** pattern inside the Order Service.
  - [x] Implement the success flow: Create Order (Pending) $\rightarrow$ Reserve Inventory.
  - [x] Implement the compensation flows: If inventory reservation fails, cancel the order and update status to `FAILED`.
  - [x] Implement gRPC client communication from Gateway to Order Service to initiate this Saga.
* **Status**: **Completed** (Saga Orchestration end-to-end flow wired to GraphQL client mutation).

---

### Phase 5: Payment, Inventory & Compensating Actions
* **Objective**: Add transactional safety layers and external integrations.
* **Tasks**:
  - [x] Scaffold the `payment/` microservice module (gRPC server).
  - [ ] Integrate **Stripe SDK** in payment service using secure payment tokens (Currently runs using full simulation gateway).
  - [x] Update the Saga Orchestrator in Order Service to call the Payment Service after stock is reserved.
  - [x] Implement compensating actions for payment failures (releasing reserved stock in Catalog Service).
  - [x] Implement **Idempotency Keys** on payment and stock reservation endpoints to prevent double-charging/double-allocation.
* **Status**: **Completed** (Mock/simulation gateway complete and integrated).

---

### Phase 6: Notifications & Event-Driven Subsystems
* **Objective**: Decouple post-transaction tasks using an asynchronous event bus.
* **Tasks**:
  - [x] Set up **Kafka** as our local message broker.
  - [x] Emit events (`events.order.completed`, `events.order.failed`) from the Saga Orchestrator to the event bus.
  - [x] Scaffold the `notification/` microservice.
  - [x] Make the notification service subscribe to these events and send simulated emails/SMS (SMTP/Mock).
* **Status**: **Completed** (Notification service, consumer, templates, and publisher wired).

---

### Phase 7: Observability & DevOps (CI/CD)
* **Objective**: Hardening for production, containerization, and automated deployments.
* **Tasks**:
  - [ ] Integrate **OpenTelemetry** instrumentation across all services for distributed tracing (Jaeger) and metrics (Prometheus).
  - [x] Write multi-stage, production-optimized **Dockerfiles** for each service to keep image sizes minimal.
  - [x] Create a comprehensive `docker-compose.yml` file to run the entire stack locally with databases, cache, and broker.
  - [x] Create an automated, dependency-free **Integration Test Runner** (`integration_tests/main.go`) to verify the end-to-end Saga flow.
  - [ ] Create **Helm Charts** for Kubernetes deployment.
  - [ ] Setup a **GitHub Actions** CI/CD pipeline that:
    - Runs Go test suites and linters.
    - Builds and pushes Docker images to a container registry.
    - Deploys changes to a cloud Kubernetes cluster (e.g. AWS EKS, DigitalOcean Kubernetes).
* **Status**: **In Progress** (Docker files, compose configuration, and integration test runner completed; CI/CD and charts remaining).

---

## 🎯 Next Immediate Step
Our next step is to run the local microservices stack via `docker-compose up --build` and execute the newly created integration test runner (`go run integration_tests/main.go`) to verify the end-to-end Saga orchestration, compensation rollbacks, and GraphQL gateway operations.
