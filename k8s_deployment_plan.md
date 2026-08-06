# Kubernetes Deployment Plan (Option 1)

This plan outlines the step-by-step roadmap to deploy our Go microservices stack to a **Managed Kubernetes Cluster**. It evaluates the best platform choices, database/broker strategies, Helm setup, and verification.

---

## 🏛️ Architecture & Platform Choices

### 1. Cloud Provider Choice (For Learning)
* **Recommended Choice: DigitalOcean Kubernetes (DOKS)**
  * **Why**: DigitalOcean provides a **free control plane** (saving ~$73/month compared to GKE/EKS) and cheap node VMs (starting at $12/month). You can run a 2-node cluster for $24/month.
  * **Alternative**: **Google Kubernetes Engine (GKE) Autopilot**. Safe, fully managed, but costs accumulate faster.

### 2. Database & Kafka Choices
* **PostgreSQL**: Deploy a self-hosted PostgreSQL cluster inside Kubernetes using the **CloudNativePG Operator**. This is a great learning exercise for running stateful applications in K8s using Custom Resources (CRDs).
* **Kafka**: Deploy a single-node Kafka instance using the **Strimzi Kafka Operator** inside the cluster (under a dev profile to minimize memory footprint).

### 3. Networking & Gateway Ingress
* **Ingress Controller**: Deploy the **NGINX Ingress Controller** via Helm.
* **Internal Service Discovery**: Microservices communicate internally using Kubernetes service names (e.g. `grpc://auth-service:50051`).
* **Ingress Rule**: Ingress routes external HTTP `/query` and `/` traffic to the `apigateway` service, keeping backend gRPC services locked inside the cluster.

---

## 📅 Implementation Roadmap

### Phase 1: Local Tools & Cluster Setup
1. Install K8s tools: `kubectl`, `helm`.
2. Spin up a Kubernetes cluster (DOKS or GKE).
3. Connect your local terminal: `doctl kubernetes cluster kubeconfig save <cluster-name>`.

### Phase 2: Deploying Stateful Infrastructure
1. **Install Strimzi Operator**:
   ```bash
   helm repo add strimzi https://strimzi.io/charts/
   helm install strimzi-operator strimzi/strimzi-kafka-operator
   ```
2. **Apply Kafka Custom Resource**: Create a minimal Kafka cluster config YAML and apply it.
3. **Deploy PostgreSQL**: Install CloudNativePG operator or deploy PostgreSQL using a simple StatefulSet + PersistentVolumeClaim.

### Phase 3: Writing Helm Charts for Microservices
We will create a unified Helm chart structured as follows:
```text
helm/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── configmap.yaml
    ├── secrets.yaml
    ├── ingress.yaml
    └── deployment-template.yaml  # Reusable deployment pattern
```
Every microservice (Auth, Account, Catalog, Order, Payment, Notification, API Gateway) will have:
1. A **Deployment** mapping environment variables (e.g. `DATABASE_URL` referencing the DB service host).
2. A **Service** exposing the port (50051-50056) internally.

### Phase 4: Deploying & Routing
1. Install Nginx Ingress:
   ```bash
   helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
   helm install ingress-nginx ingress-nginx/ingress-nginx
   ```
2. Build and push your Docker images to a Container Registry (Docker Hub or DigitalOcean Container Registry).
3. Deploy the Helm chart:
   ```bash
   helm install ecommerce-stack ./helm -f values.yaml
   ```

### Phase 5: Verification
1. Get the External IP of the Nginx Ingress:
   ```bash
   kubectl get svc ingress-nginx-controller
   ```
2. Execute the custom integration tests against this IP:
   ```bash
   GATEWAY_URL="http://<ingress-external-ip>/query" go run integration_tests/main.go
   ```
