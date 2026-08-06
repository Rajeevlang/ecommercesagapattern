# Serverless Container Deployment Plan (Option 2)

This plan outlines the step-by-step roadmap to deploy our Go microservices stack serverlessly using **Google Cloud Run** and Managed Serverless Infrastructure.

---

## 🏛️ Architecture & Platform Choices

### 1. Cloud Provider Choice (For Learning)
* **Recommended Choice: Google Cloud Run**
  * **Why**: Google Cloud Run is a fully managed serverless container platform. It supports **scaling to zero instances** when there is no traffic. This makes it **virtually free** for learning and development.
  * **gRPC Support**: Cloud Run natively supports HTTP/2 and gRPC, making it a perfect fit for our microservice mesh.

### 2. Serverless Database & Broker Strategy
Since Cloud Run containers are serverless and spin up/down dynamically, we should use serverless SaaS platforms with generous free tiers for our stateful infrastructure:

* **PostgreSQL (Neon / Supabase)**:
  * **Neon** provides serverless Postgres that automatically sleeps when inactive. This fits the scale-to-zero model perfectly.
  * We will create 6 database branches or credentials (one for each microservice).
* **Kafka Broker (Upstash / Confluent Cloud)**:
  * **Upstash Kafka** is serverless, bills per message, and has a free tier.
  * **Confluent Cloud** offers $400 of free credits for new accounts, which is more than enough to test Kafka clusters.

### 3. Networking & gRPC Communication
* **Internal Services**: Auth, Account, Catalog, Order, Payment, and Notification services are deployed with Cloud Run ingress restricted to **"Internal" only**, preventing public access.
* **Service Accounts (IAM)**: Gateway uses Google IAM tokens to authenticate gRPC requests sent to internal backend services.
* **Public Gateway**: The API Gateway is deployed as a public Cloud Run service, serving the GraphQL playground on port 8080.

---

## 📅 Implementation Roadmap

### Phase 1: Accounts & Tool Setup
1. Create a Google Cloud Platform (GCP) account.
2. Install the **Google Cloud SDK** (`gcloud` CLI).
3. Create a serverless Postgres instance on **Neon** or **Supabase** and run the database migration tables.
4. Create a serverless Kafka cluster on **Upstash** or **Confluent Cloud** and retrieve the broker URLs and credentials.

### Phase 2: Create GCP Project & Registry
1. Initialize project:
   ```bash
   gcloud init
   gcloud projects create ecom-saga-learning --set-as-default
   ```
2. Enable Cloud Run and Artifact Registry APIs:
   ```bash
   gcloud services enable run.googleapis.com artifactregistry.googleapis.com
   ```
3. Create a Docker repository in Artifact Registry:
   ```bash
   gcloud artifacts repositories create ecom-repo --repository-format=docker --location=us-central1
   ```

### Phase 3: Push Images to GCP
Configure local Docker to auth with GCP, then build and push each microservice container:
```bash
# Authenticate docker
gcloud auth configure-docker us-central1-docker.pkg.dev

# Build & Push Example (API Gateway)
docker build -t us-central1-docker.pkg.dev/ecom-saga-learning/ecom-repo/apigateway:latest -f apigateway/Dockerfile .
docker push us-central1-docker.pkg.dev/ecom-saga-learning/ecom-repo/apigateway:latest
```
*(Repeat build/push steps for all 7 microservices).*

### Phase 4: Deploying Backend Services to Cloud Run
Deploy services serverlessly, enabling the `--use-http2` flag so gRPC works properly:
```bash
gcloud run deploy auth-service \
  --image us-central1-docker.pkg.dev/ecom-saga-learning/ecom-repo/auth:latest \
  --platform managed \
  --region us-central1 \
  --ingress internal \
  --use-http2 \
  --set-env-vars="DATABASE_URL=postgres://...",JWT_SECRET="secret"
```
*(Deploy all backend services: auth, account, catalog, order, payment, notification).*

### Phase 5: Deploying & Connecting API Gateway
1. Get the URLs of the deployed backend services:
   ```bash
   AUTH_URL=$(gcloud run services describe auth-service --format='value(status.url)' --region=us-central1)
   # (Collect URLs for account, catalog, and order services)
   ```
2. Deploy the API Gateway, pointing to the backend Cloud Run URLs:
   ```bash
   gcloud run deploy api-gateway \
     --image us-central1-docker.pkg.dev/ecom-saga-learning/ecom-repo/apigateway:latest \
     --platform managed \
     --region us-central1 \
     --ingress all \
     --allow-unauthenticated \
     --set-env-vars="AUTH_SVC_ADDR=${AUTH_URL}:443,ACCOUNT_SVC_ADDR=${ACCOUNT_URL}:443,..."
   ```

### Phase 6: Verification
1. Retrieve the public Gateway URL.
2. Run the integration test runner:
   ```bash
   GATEWAY_URL="https://<api-gateway-cloudrun-url>/query" go run integration_tests/main.go
   ```
