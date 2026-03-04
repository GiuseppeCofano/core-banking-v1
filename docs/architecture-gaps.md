# Core Banking Engine — Architecture Gaps & Roadmap

> Assessment of the current implementation and prioritized improvements.

---

## 🔴 Critical

### 1. Saga Pattern for Transaction Safety
The `Transfer` flow in `core/banking.go` makes two sequential HTTP calls (debit, then credit). If the credit fails after the debit succeeds, money is lost. An **orchestration-based Saga** in the Core service would track each step's state and trigger compensating actions on failure.

### 2. Authentication & Authorization
All endpoints are currently open. Minimum requirements:
- **API keys or OAuth2/JWT** on the Processor gateway for external clients
- **mTLS or service-account tokens** between internal services on the K8s cluster
- **RBAC** — not every client should be able to create accounts or initiate transfers

### 3. SQLite → PostgreSQL
SQLite is single-writer and file-based — a single point of failure with no horizontal scaling. Swap to **PostgreSQL** (Cloud SQL on GKE) for production. The `ledger/db.go` abstraction makes this a contained migration. Consider connection pooling (e.g. PgBouncer).

### 4. Idempotency
If a client retries a request (timeout, network glitch), the operation executes again — duplicating deposits or transfers. Add an `idempotency_key` to requests, store it in a DB table, and return the cached response on duplicate submissions.

---

## 🟡 Important

### 5. Floating-Point → Integer Cents
`float64` for monetary amounts will cause rounding errors. Use `int64` representing the smallest currency unit (e.g. cents) or a decimal library like `shopspring/decimal`.

### 6. Structured Logging & Observability
Replace `log.Printf` with structured JSON logging (`slog` in Go 1.22). Add **OpenTelemetry** distributed tracing so requests can be followed across Processor → Core → Ledger. Export metrics (request latency, error rates) to Cloud Monitoring.

### 7. Input Validation & Rate Limiting
Current validation is minimal. Add:
- Maximum transaction amount caps
- Currency whitelist validation
- Rate limiting per client/IP at the Processor level
- Request size limits

### 8. CI/CD Pipeline
A GitHub Actions workflow should:
1. `go build` + `go test` on every PR
2. Build & push Docker images on merge to `main`
3. Deploy to GKE staging automatically
4. Promote to production with manual approval

### 9. WebApp Hardening
The WebApp reverse proxy (`webapp/main.go`) currently has no CORS policy, no Content-Security-Policy headers, and no session/cookie management. For production:
- Add **CORS** headers restricting origins to the WebApp domain
- Set `Content-Security-Policy`, `X-Frame-Options`, and `X-Content-Type-Options` response headers
- Serve static assets with **cache-busting** hashes
- Terminate TLS at the Ingress / LoadBalancer level
- Consider adding **rate limiting** to the proxy endpoints

---

## 🟢 Nice to Have

### 10. Pagination on Ledger Entries
`GET /ledger/entries/{account_id}` returns all entries. Add `?limit=50&offset=0` or cursor-based pagination before any account accumulates thousands of entries.

### 11. API Versioning
Prefix all routes with `/v1/` (e.g. `/v1/accounts`) to allow non-breaking API evolution.

### 12. Graceful Shutdown
The HTTP servers don't handle `SIGTERM`. During K8s rolling deployments, in-flight requests get killed. Use `signal.NotifyContext` + `http.Server.Shutdown()` for clean draining.

### 13. Unit & Integration Tests
Add Go unit tests for the business logic in `core/banking.go` and integration tests that spin up the Ledger against an in-memory SQLite database.

---

## Suggested Priority Order

| Sprint | Items | Effort |
|--------|-------|--------|
| **1** | Float→int cents, idempotency | ~1 day |
| **2** | Saga pattern | ~2 days |
| **3** | PostgreSQL migration, graceful shutdown | ~1–2 days |
| **4** | Auth (JWT + mTLS), rate limiting, WebApp hardening | ~2 days |
| **5** | Observability (slog + OpenTelemetry) | ~1 day |
| **6** | CI/CD, pagination, API versioning, tests | ~2 days |
