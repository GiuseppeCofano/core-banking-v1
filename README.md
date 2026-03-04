# Core Banking Engine v1

A microservices-style core banking engine written in Go, designed for Kubernetes (GKE).

## Architecture

```
Client → [Processor :8082] → [Core :8081] → [Ledger :8080] → SQLite (PVC) / Cloud Spanner
```

| Service | Port | Role |
|---------|------|------|
| **Ledger** | 8080 | Account management & double-entry ledger (SQLite or Spanner) |
| **Core** | 8081 | Business logic — deposits & transfers (Saga pattern) |
| **Processor** | 8082 | Request validation & routing gateway |

The **Ledger** supports two database backends, selectable via the `DB_BACKEND` env var:
- `sqlite` (default) — embedded SQLite with PVC storage
- `spanner` — Google Cloud Spanner for production-grade distributed transactions

## Prerequisites

- Docker
- `kubectl` configured for your GKE cluster
- A GCP project with Container Registry (`gcr.io`)
- (Spanner backend) Terraform >= 1.5 and a GCP project with Spanner API enabled

## Quick Start

### 1. Build & Push Docker Images

```bash
export PROJECT_ID=your-gcp-project-id
make docker-build
make docker-push
```

### 2. Update K8s Manifests

Replace `PROJECT_ID` in `k8s/*.yaml` with your actual GCP project ID:

```bash
sed -i '' "s/PROJECT_ID/$PROJECT_ID/g" k8s/ledger.yaml k8s/core.yaml k8s/processor.yaml
```

### 3. Deploy to GKE

```bash
make k8s-apply
```

### 4. Access the Services

```bash
# Option A: port-forward for local testing
kubectl port-forward -n core-banking svc/ledger 8080:8080 &
kubectl port-forward -n core-banking svc/processor 8082:8082 &

# Option B: The processor service uses LoadBalancer —
# get its external IP with:
kubectl get svc -n core-banking processor
```

## API Examples

### Create Accounts

```bash
curl -s -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"owner":"Alice","currency":"EUR"}' | jq

curl -s -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"owner":"Bob","currency":"EUR"}' | jq
```

### Deposit

```bash
curl -s -X POST http://localhost:8082/process/deposit \
  -H "Content-Type: application/json" \
  -d '{"account_id":"<ALICE_ID>","amount":500.00}' | jq
```

### Transfer

```bash
curl -s -X POST http://localhost:8082/process/transfer \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":"<ALICE_ID>","to_account_id":"<BOB_ID>","amount":150.00}' | jq
```

### Check Balances

```bash
curl -s http://localhost:8080/accounts/<ALICE_ID> | jq  # → 350.00
curl -s http://localhost:8080/accounts/<BOB_ID> | jq    # → 150.00
```

## Tear Down

```bash
make k8s-delete
```

## Cloud Spanner Setup (Terraform)

To provision the Spanner instance and database:

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your GCP project and preferences
terraform init
terraform plan
terraform apply
```

Then deploy the ledger with Spanner:

```bash
# In k8s/ledger.yaml, set:
#   DB_BACKEND: "spanner"
#   SPANNER_DATABASE: <value from terraform output spanner_database_path>
make k8s-apply
```

## Project Structure

```
core-banking-v1/
├── models/          # Shared types
│   └── models.go
├── ledger/          # Ledger microservice (SQLite / Spanner)
│   ├── main.go      # Entrypoint with backend selection
│   ├── store.go     # Store interface
│   ├── db.go        # SQLite backend (SQLiteStore)
│   ├── spanner.go   # Spanner backend (SpannerStore)
│   ├── handlers.go  # HTTP handlers
│   └── Dockerfile
├── core/            # Core microservice (business logic + Saga)
│   ├── main.go
│   ├── banking.go
│   ├── saga.go
│   ├── handlers.go
│   └── Dockerfile
├── processor/       # Processor microservice (gateway)
│   ├── main.go
│   ├── processor.go
│   ├── handlers.go
│   └── Dockerfile
├── terraform/       # Spanner infrastructure (IaC)
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   └── terraform.tfvars.example
├── k8s/             # Kubernetes manifests
│   ├── namespace.yaml
│   ├── ledger.yaml
│   ├── core.yaml
│   └── processor.yaml
├── go.mod
├── Makefile
└── README.md
```
