# Core Banking Engine v1

A microservices-style core banking engine written in Go, designed for Kubernetes (GKE).

## Architecture

```
Client → [Processor :8082] → [Core :8081] → [Ledger :8080] → SQLite (PVC)
```

| Service | Port | Role |
|---------|------|------|
| **Ledger** | 8080 | Account management & double-entry ledger (SQLite) |
| **Core** | 8081 | Business logic — deposits & transfers |
| **Processor** | 8082 | Request validation & routing gateway |

## Prerequisites

- Docker
- `kubectl` configured for your GKE cluster
- A GCP project with Container Registry (`gcr.io`)

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

## Project Structure

```
core-banking-v1/
├── models/          # Shared types
│   └── models.go
├── ledger/          # Ledger microservice (SQLite)
│   ├── main.go
│   ├── db.go
│   ├── handlers.go
│   └── Dockerfile
├── core/            # Core microservice (business logic)
│   ├── main.go
│   ├── banking.go
│   ├── handlers.go
│   └── Dockerfile
├── processor/       # Processor microservice (gateway)
│   ├── main.go
│   ├── processor.go
│   ├── handlers.go
│   └── Dockerfile
├── k8s/             # Kubernetes manifests
│   ├── namespace.yaml
│   ├── ledger.yaml
│   ├── core.yaml
│   └── processor.yaml
├── go.mod
├── Makefile
└── README.md
```
