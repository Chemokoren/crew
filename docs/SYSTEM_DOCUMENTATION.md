# AMY MIS — Backend System Documentation

> **Version:** 1.3 | **Last Updated:** 2026-05-07 | **Go:** 1.25 | **Framework:** Gin 1.12

---

## 1. System Overview

AMY MIS (Management Information System) is a **Workforce Financial Operating System** for Kenya's informal transport sector. It digitizes the financial lifecycle of matatu, boda-boda, and tuk-tuk crews — from shift assignments and earnings calculation to wallet management, payroll processing, and SACCO operations.

### 1.1 Technology Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.25 |
| HTTP Framework | Gin | 1.12 |
| ORM | GORM | 1.31 |
| Database | PostgreSQL | 16 |
| Cache/Queue Backend | Redis | 7 |
| Object Storage | MinIO (S3-compat) | latest |
| Migrations | golang-migrate | 4.19 |
| Auth | golang-jwt (HS256) | 5.3 |
| API Docs | Swaggo/Swagger | 1.16 |
| Containerization | Docker + Compose | multi-stage |
| Hot Reload | Air | latest |

### 1.2 External Integrations

| Service | Purpose | Pattern | Status |
|---------|---------|---------|--------|
| **Optimize SMS** | SMS notifications (default) | Strategy (primary) | ✅ Implemented |
| **Africa's Talking** | SMS notifications (fallback) | Strategy (fallback) | ✅ Implemented |
| **JamboPay v2** | M-Pesa STK push (collections) + B2C / bank / paybill payouts | Strategy | ✅ Implemented |
| **PerPay** | Payroll & statutory remittance | Strategy | ✅ Implemented |
| **IPRS** | KYC / national ID verification | Strategy | ✅ Implemented |
| **MinIO** | Document/file storage | Direct client | ✅ Implemented |

---

## 2. Architecture

### 2.1 Layered Architecture (Clean Architecture)

```
┌─────────────────────────────────────────────────┐
│                  cmd/server/main.go              │  Entry point + DI wiring
├─────────────────────────────────────────────────┤
│                  middleware/                      │  CORS, Auth, RBAC, Rate Limit (Redis),
│                                                  │  Metrics (Atomic), Logger, Recovery, Security
├─────────────────────────────────────────────────┤
│                  handler/ + dto/                 │  14 HTTP handler files + structured DTOs
├─────────────────────────────────────────────────┤
│                  service/                        │  24 Business logic services
├─────────────────────────────────────────────────┤
│                  repository/ (interfaces)        │  19 Data access contracts
│                  repository/postgres/            │  Postgres implementation (GORM)
│                  repository/mock/                │  100% Mock parity (19/19)
├─────────────────────────────────────────────────┤
│                  models/                         │  15 GORM entity models
├─────────────────────────────────────────────────┤
│                  database/                       │  PostgreSQL + Redis connections
│                  external/                       │  Strategy pattern API clients
├─────────────────────────────────────────────────┤
│                  pkg/                            │  Shared utilities (jwt, errs,
│                                                  │  money, validator, types)
└─────────────────────────────────────────────────┘
```

**Key principle:** Dependencies flow inward. Services depend on repository *interfaces*, never on GORM implementations. Handlers never touch the database directly.

### 2.2 Directory Structure

```
backend/
├── cmd/server/main.go          — Single entry point, all DI wiring
├── internal/
│   ├── config/                 — Env-based config + validation
│   ├── database/
│   │   ├── postgres.go         — PostgreSQL (GORM) connection
│   │   ├── redis.go            — Redis connection
│   │   └── tx.go               — TxManager (atomic transactions)
│   ├── models/                 — GORM models for all entities
│   ├── repository/
│   │   ├── interfaces.go       — 19 repository interfaces
│   │   ├── postgres/           — GORM implementations
│   │   └── mock/               — 19 mocks (100% coverage)
│   ├── service/                — 24 business logic services
│   ├── handler/                — 14 handler files + DTOs
│   ├── middleware/             — 8 middleware layers (Auth, Metrics, etc.)
│   ├── worker/
│   │   ├── scheduler.go        — Goroutine-based job scheduler
│   │   ├── daily_summary.go    — Earnings aggregation
│   │   ├── insurance_lapse.go  — Policy status monitor
│   │   ├── payroll_submit.go   — Auto-payroll processor
│   │   └── wallet_reconciliation.go — Financial audit worker
│   └── external/
│       ├── sms/                — SMS Provider strategy
│       ├── payment/            — Payment Provider strategy
│       ├── payroll/            — Payroll Provider strategy
│       ├── identity/           — Identity Provider strategy
│       └── storage/minio.go    — MinIO client
├── pkg/
│   ├── errs/                   — Domain error sentinels
│   ├── jwt/                    — JWT Manager
│   ├── money/                  — Financial math utilities
│   ├── validator/              — Domain-specific validation (Phone, ID)
│   └── types/                  — Shared type definitions
├── migrations/                 — 8 migration sets (16 files, 22 tables)
├── docs/                       — Swagger + Gap Analysis docs
└── Dockerfile                  — Multi-stage Alpine build
```

### 2.3 Request Flow

```
Client → Gin Router
  → CORS → SecureHeaders → RequestID → RateLimit (Redis) → Timeout (Context) → Metrics (Atomic) → Logger → Recovery
    → [Public routes: health, auth, swagger]
    → [Secured routes: JWTAuth → RequireRole]
      → Handler (parse req, bind DTO, validate)
        → Service (business logic, validation)
          → Repository Interface → Postgres Implementation (GORM)
        ← Service returns result/error
      ← Handler maps to HTTP response (DTO)
    ← Middleware logs, records metrics
  ← JSON Response to Client
```

### 2.4 Startup Sequence (main.go)

1. Initialize structured `slog` logger
2. Load + validate configuration from `.env`
3. Connect to PostgreSQL (GORM pool tuning)
4. Run database migrations (`golang-migrate`)
5. Connect to Redis
6. Connect to MinIO + ensure bucket exists
7. Initialize 19 repositories
8. Initialize transaction manager (`TxManager`)
9. Initialize JWT Manager
10. Initialize 18 services with dependency injection
11. Initialize 16 handlers
12. Initialize 4 background workers (Scheduler)
13. Configure Gin router + register middleware + routes
14. Start HTTP server with graceful shutdown (30s drain)

---

## 3. Data Model

### 3.1 Entity Relationship Overview

```
Users ──────┬──→ CrewMembers ──→ CrewSACCOMemberships ──→ SACCOs
            │         │                                      │
            │         ├──→ Documents (MinIO)                 ├──→ Vehicles ──→ Routes
            │         ├──→ Wallets ──→ WalletTransactions    ├──→ SACCOFloats
            │         ├──→ Assignments ──→ Earnings          │
            │         ├──→ CreditScores                      └──→ PayrollRuns ──→ PayrollEntries
            │         ├──→ LoanApplications
            │         └──→ InsurancePolicies
            │
            ├──→ Notifications
            └──→ AuditLogs

WebhookEvents (inbound from JamboPay/Perpay/IPRS)
NotificationTemplates (event→channel templates)
StatutoryRates (SHA, NSSF, Housing Levy)
```

### 3.2 Migration Sets (7 total, 22 tables)

| # | Domain | Tables |
|---|--------|--------|
| 1 | Users | `users` |
| 2 | Identity Registry | `saccos`, `crew_members`, `routes`, `vehicles`, `crew_sacco_memberships` + FK constraints |
| 3 | Operations | `assignments`, `earnings`, `daily_earnings_summaries` |
| 4 | Financial | `wallets`, `wallet_transactions`, `sacco_floats`, `sacco_float_transactions` |
| 5 | Payroll | `payroll_runs`, `payroll_entries`, `statutory_rates` |
| 6 | Infrastructure | `webhook_events`, `notifications`, `notification_templates`, `documents`, `audit_logs` |
| 7 | Financial Services | `credit_scores`, `loan_applications`, `insurance_policies` |

### 3.3 Key Design Decisions

- **All money as `int64` cents** — no floating-point anywhere in the pipeline
- **Currency:** KES (Kenyan Shilling) hardcoded as default
- **Soft deletes** via GORM `DeletedAt` on CrewMembers, SACCOs, Vehicles, Routes
- **UUID primary keys** with `gen_random_uuid()` (pgcrypto extension)
- **Optimistic locking** via `version` column on Wallets and SACCOFloats
- **Idempotency keys** with unique indexes on financial transaction tables
- **Partial indexes** for active records (e.g., `WHERE deleted_at IS NULL`)
- **Crew IDs** use a PostgreSQL sequence: `CRW-00001`, `CRW-00002`, etc.

---

## 4. Authentication & Authorization

### 4.1 JWT Token System

- **Algorithm:** HS256 (HMAC-SHA256)
- **Access token TTL:** 15 minutes (configurable)
- **Refresh token TTL:** 7 days (configurable)
- **Secret:** Minimum 32 characters (validated at startup)
- **Claims:** `user_id`, `phone`, `system_role`, `crew_member_id`, `sacco_id`
- **Password hashing:** bcrypt cost factor 12

### 4.2 RBAC Roles

| Role | Code | Permissions |
|------|------|-------------|
| System Admin | `SYSTEM_ADMIN` | Full access to all resources |
| Employer | `EMPLOYER` | Manage crew, vehicles, assignments within organization |
| Employee | `EMPLOYEE` | View own profile, wallet, transactions |
| Lender | `LENDER` | View loan-related data |
| Insurer | `INSURER` | View insurance-related data |

### 4.3 KYC Enforcement

Employees with non-verified KYC status (`PENDING` or `REJECTED`) are restricted to:
- `/profile` — to upload/update KYC documents
- `/notifications` — to view notifications (including unverification reasons)

All other frontend routes are blocked by a `kycGuard`, and the sidebar visually disables locked items with a lock icon and warning banner.

**KYC Unverification Workflow:**
1. Admin clicks "Unverify" on a verified employee's KYC row in `/documents`
2. A prompt dialog captures the reason for unverification
3. `PUT /api/v1/crew/:id/kyc` is called with `{ kyc_status: "PENDING", reason: "..." }`
4. Backend clears `KYCVerifiedAt`, updates status, and dispatches an IN_APP notification
5. Employee's navigation is immediately restricted until re-verification

### 4.4 Route Protection

| Route Group | Auth Required | Role Required | KYC Required |
|-------------|--------------|---------------|---------------|
| `/health`, `/ready`, `/metrics` | ❌ | — | — |
| `/swagger/*` | ❌ | — | — |
| `/api/v1/auth/register,login,refresh` | ❌ | — | — |
| `/api/v1/auth/me` | ✅ JWT | Any | ❌ |
| `/profile`, `/notifications` | ✅ JWT | Any | ❌ (KYC-exempt) |
| `/api/v1/crew/*` | ✅ JWT | SYSTEM_ADMIN or EMPLOYER | ❌ (admin routes) |
| `/api/v1/assignments/*` | ✅ JWT | SYSTEM_ADMIN or EMPLOYER | ❌ (admin routes) |
| `/dashboard`, `/earnings`, `/wallets`, etc. | ✅ JWT | Any | ✅ (employees) |
| `/api/v1/wallets/:id` (GET) | ✅ JWT | Any (ownership enforced) | — |
| `/api/v1/transactions/employee-payout` | ✅ JWT | SYSTEM_ADMIN or EMPLOYER | — |
| `/api/v1/transactions/transfer` | ✅ JWT | Any (sender derived from JWT) | — |

---

## 5. Middleware Stack

Applied in order on every request:

| # | Middleware | Purpose |
|---|-----------|---------|
| 1 | CORS | Allows all origins (dev), exposes `X-Request-Id`, `X-Response-Time` |
| 2 | SecureHeaders | `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`, `Cache-Control` |
| 3 | RequestID | UUID per request, propagated via `X-Request-ID` header |
| 4 | RateLimit | 100 req/min per IP, in-memory sliding window with goroutine cleanup |
| 5 | Timeout | 30s deadline tracking (header-based, not context-based) |
| 6 | Metrics | In-memory request counters, per-path latency histograms |
| 7 | Logger | Structured `slog` logging with method, path, status, latency, IP |
| 8 | Recovery | Panic recovery with full stack trace logging |

---

## 6. External Integration Architecture

### 6.1 Strategy Pattern

All third-party integrations follow the **Strategy design pattern**. Each integration category defines a `Provider` interface and a `Manager` that orchestrates providers with automatic fallback:

```
┌─────────────────────┐
│      Manager        │  ← Holds ordered list of providers
│  (fallback chain)   │
└────────┬────────────┘
         │ try primary, then fallback
    ┌────▼────┐    ┌────────────┐
    │Provider A│    │Provider B  │
    │(primary) │    │(fallback)  │
    └──────────┘    └────────────┘
```

**Runtime switching:** `manager.SetPrimary("provider_name")` reorders providers without restart.

### 6.2 SMS — `sms.Manager`

| Provider | Role | Auth | Capability |
|----------|------|------|------------|
| **Optimize** | Default | OAuth2 → JWT token (cached, thread-safe) | `Send`, `SendBulk` |
| **Africa's Talking** | Fallback | `apiKey` header | `Send`, `SendBulk` (native comma-separated) |

- Automatic fallback: if Optimize fails, Africa's Talking is tried transparently
- Token caching: Optimize tokens are cached with configurable TTL (default 3600s), refreshed 60s early

### 6.3 Payment — `payment.Manager`

| Provider | Auth | Endpoints |
|----------|------|-----------|
| **JamboPay v2** | OAuth2 client_credentials → Bearer token | `InitiateCollection` (STK push), `InitiatePayout` (M-Pesa B2C/bank/paybill), `VerifyPayout` (OTP), `CheckBalance` |

- **Channels:** `MOMO_B2C` (mobile money), `BANK` (bank transfer), `MOMO_B2B` (paybill/till)
- **Collection flow (STK push):** `InitiateCollection` → phone prompt → user enters PIN → JamboPay callback → `ConfirmPendingTopUp`
- **Payout OTP flow:** `InitiatePayout` → `pending_otp` → `VerifyPayout(ref, otp)` → `completed`

### 6.4 Payroll — `payroll.Manager`

| Provider | Auth | Endpoints |
|----------|------|-----------|
| **PerPay** | JWT via `/auth/issue` (15min TTL) | `SubmitPayroll` (async 202), `GetStatus` (polling) |

- **Async processing:** Submit returns `202 Accepted` with `correlation_id`
- **Status polling:** `received` → `validating_input` → `calculating_deductions` → `completed` / `failed`
- **Idempotency:** `Idempotency-Key` header prevents duplicate submissions (409 replay)

### 6.5 Identity — `identity.Manager`

| Provider | Auth | Endpoints |
|----------|------|-----------|
| **IPRS** | OAuth2 via JamboPay IdP (scope=`iprs`) | `VerifyCitizen` (POST `/citizen-details`) |

- **Response:** First name, surname, gender, DOB, place of birth, citizenship, photo (base64)
- **Token endpoint:** Separate from IPRS API (uses JamboPay's identity server)

### 6.6 Adding a New Provider

1. Create a file in the appropriate package (e.g. `internal/external/sms/twilio.go`)
2. Implement the `Provider` interface (e.g. `sms.Provider` with `Name()`, `Send()`, `SendBulk()`)
3. Add config fields to `internal/config/config.go`
4. Register in `cmd/server/main.go`:
   ```go
   smsProviders = append(smsProviders, sms.NewTwilioProvider(cfg, logger))
   ```

---

## 7. Business Logic — Earning Models

| Model | Formula | Use Case |
|-------|---------|----------|
| `FIXED` | `fixed_amount_cents` | Daily flat rate |
| `COMMISSION` | `revenue × commission_rate` | Percentage of collections |
| `HYBRID` | `base_cents + (revenue × commission_rate)` | Base pay + commission |

When an assignment is completed:
1. Earnings are calculated based on the model
2. An `Earning` record is created
3. The crew member's wallet is **automatically credited** with an idempotency key

### 7.5 Organization Float Top-Up Flow

Organization (SACCO) float can be funded via three methods. **Mobile money** uses an asynchronous STK push flow; **bank** and **card** are immediate manual entries.

#### Mobile Money (Async — STK Push)

```
                              ┌────────────────────────────────┐
  Admin clicks               │  POST /organizations/:id/      │
  "Top Up" (M-Pesa)   ──────►│       float/topup               │
                              │  method: "mobile_money"        │
                              └──────────┬─────────────────────┘
                                         │
                              ┌──────────▼─────────────────────┐
                              │  1. Create PENDING float tx    │  No balance change
                              │  2. Trigger JamboPay STK push  │  Phone prompt sent
                              │  3. Return HTTP 202 Accepted   │  "Check your phone"
                              └──────────┬─────────────────────┘
                                         │
                              ┌──────────▼─────────────────────┐
                              │  User enters M-Pesa PIN        │
                              │  on their phone                │
                              └──────────┬─────────────────────┘
                                         │
                              ┌──────────▼─────────────────────┐
                              │  JamboPay sends callback       │
                              │  POST /api/v1/webhooks/        │
                              │       jambopay                 │
                              └──────────┬─────────────────────┘
                                         │
                          ┌──────────────┴──────────────┐
                    SUCCESS                         FAILED
                          │                              │
               ┌──────────▼──────────┐        ┌─────────▼─────────┐
               │ ConfirmPendingTopUp │        │ FailPendingTopUp  │
               │ • Credit balance    │        │ • Mark tx FAILED  │
               │ • Status→COMPLETED  │        │ • No balance chg  │
               └─────────────────────┘        └───────────────────┘
```

**Key design decisions:**
- The float balance is **never** credited until the payment is confirmed via callback
- Pending transactions are idempotency-protected (`idempotency_key` as the JamboPay `orderId`)
- Failed STK pushes immediately mark the pending tx as `FAILED`
- The webhook handler first checks for pending float tx (collection), then payout tx

#### Bank & Card (Immediate)

Bank and card top-ups credit the float balance immediately as they represent manual administrative entries:

```
POST /organizations/:id/float/topup  →  CreditFloat()  →  HTTP 201 Created
```

#### Float Transaction Types

The `sacco_float_transactions` table enforces a check constraint:

| Type | Usage |
|------|-------|
| `FUND` | Inbound: mobile money STK push, bank transfer, card payment |
| `PAYOUT` | Outbound: disbursement to crew wallets or external accounts |
| `ADJUSTMENT` | Administrative corrections |

#### Float Transaction Statuses

| Status | Meaning |
|--------|--------|
| `PENDING` | STK push initiated, awaiting payment confirmation |
| `COMPLETED` | Payment confirmed, balance updated |
| `FAILED` | Payment failed or STK push error |
| `REVERSED` | Previously completed transaction reversed |

---

## 8. Financial Safety Mechanisms

### 8.1 Wallet Concurrency Control

The wallet repository uses **belt-and-suspenders** concurrency:
1. **Pessimistic locking:** `SELECT ... FOR UPDATE` on the wallet row
2. **Optimistic locking:** Version check before committing
3. **Idempotency:** Duplicate transactions with the same key return the original

### 8.2 Float Concurrency Control

The organization float repository mirrors wallet safety mechanisms:
1. **Pessimistic locking:** `SELECT ... FOR UPDATE` on float row during credit/debit
2. **Optimistic locking:** Version check prevents concurrent modification
3. **Idempotency:** Float transactions keyed by `idempotency_key` (unique index)
4. **Pending→Confirm pattern:** For STK push collections, a `PENDING` record is created first (no balance change), then atomically confirmed when the callback arrives — preventing premature balance inflation
5. **Context-injected transactions:** Float repo uses `getDB(ctx)` to participate in externally-managed DB transactions (e.g., from `TransactionService`)

### 8.3 Atomic Multi-Repository Transactions (TransactionService)

Operations that span **multiple repositories** (e.g., debiting org float AND crediting employee wallet) are wrapped in a single database transaction via `database.TxManager.RunInTx`. If either side fails, the entire operation is rolled back — **no partial state is possible**.

#### Employee Payout (float → wallet)

```
POST /api/v1/transactions/employee-payout
{
  "crew_member_id": "uuid",
  "gross_cents": 10000,      ← Total cost to organization
  "net_cents": 8000,          ← Amount after statutory deductions (NSSF, SHA, Housing)
  "idempotency_key": "uuid",
  "description": "Gross: 100 KES, Net: 80 KES | Deductions: NSSF: 10, SHA: 10"
}
```

Inside a single DB transaction:
1. **Debit** org float by `gross_cents` (total cost to organization)
2. **Credit** employee wallet by `net_cents` (take-home pay after deductions)
3. If either fails → full rollback, no funds moved

Derived idempotency keys: `{key}` for float debit, `{key}:wallet` for wallet credit.

#### Wallet-to-Wallet Transfer

```
POST /api/v1/transactions/transfer
{
  "to_crew_member_id": "uuid",
  "amount_cents": 5000,
  "idempotency_key": "uuid",
  "description": "Lunch money"
}
```

Inside a single DB transaction:
1. **Debit** sender wallet (derived from JWT `crew_member_id`)
2. **Credit** recipient wallet
3. If either fails → full rollback, no funds moved

Derived idempotency keys: `{key}:debit` for sender, `{key}:credit` for recipient.

### 8.4 Idempotency

- Financial endpoints (`/wallets/credit`, `/wallets/debit`) require an `Idempotency-Key` HTTP header
- Atomic transaction endpoints (`/transactions/employee-payout`, `/transactions/transfer`) use `idempotency_key` in the JSON body
- Earning-to-wallet credits use `earn-{earning_id}` as the key
- Float top-up transactions use the frontend-generated `idempotency_key` as the JamboPay `orderId`
- Duplicate requests safely return the original transaction

### 8.5 Error Handling

Domain errors in `pkg/errs/`:

| Error | HTTP Status | Code |
|-------|-------------|------|
| `ErrInvalidCredentials` | 401 | `UNAUTHORIZED` |
| `ErrPhoneAlreadyExists` | 409 | `CONFLICT` |
| `ErrAccountDisabled` | 403 | `FORBIDDEN` |
| `ErrNotFound` | 404 | `NOT_FOUND` |
| `ErrInsufficientBalance` | 422 | `INSUFFICIENT_BALANCE` |
| `ErrOptimisticLock` | 409 | `CONCURRENT_MODIFICATION` |
| `ErrValidation` | 400 | `VALIDATION_ERROR` |

---

## 9. Testing

### 9.1 Test Coverage

- **425+ tests** across **20 test packages** (all passing)
- **54 test files** covering services, handlers, middleware, integrations, and workers
- **Mock repositories** for User, Crew, Wallet, Organization Float, and all others with thread-safe operations
- **httptest servers** for all external integration tests (no real API calls)

### 9.2 Test Categories

| Category | Coverage |
|----------|----------|
| Auth flows | Register, login, refresh, disabled accounts |
| Wallet operations | Credit, debit, idempotency, insufficient balance |
| Earning calculations | FIXED, COMMISSION, HYBRID models |
| Financial edge cases | Large amounts (40B KES), exact balance debit, 1-cent overdraw |
| Concurrency | 20 parallel credits with race detector |
| Atomic transactions | Employee payout validation (zero/negative/net>gross), wallet transfer validation (zero/negative/self-transfer) |
| HTTP handlers | Register, login, refresh, /me, RBAC enforcement |
| **KYC lifecycle** | Verify, unverify (→PENDING with reason), reject (→REJECTED with reason), timestamp clearing, notification dispatch |
| JWT middleware | Missing token, invalid token, expired token |
| SMS integration | Manager fallback chain, SetPrimary, Optimize token caching, AT bulk send |
| JamboPay integration | OAuth2 auth, M-Pesa/bank/paybill payout, OTP verify, balance, token cache, collection STK push |
| Webhook processing | JamboPay payout callback (COMPLETED/FAILED/reversal), collection callback (float confirm/fail) |
| PerPay integration | JWT auth, async submit (202), idempotency replay (409), status polling |
| IPRS integration | OAuth2 scope=iprs, citizen lookup, not found, auth failure, token cache |

### 9.3 Running Tests

```bash
go test ./... -race -count=1 -v          # All tests with race detector
go test ./internal/service/... -v         # Service layer only
go test ./internal/handler/... -v         # HTTP handlers
go test -run TestCrewService_UpdateKYC -v ./internal/service/...  # KYC unverification tests
go test -run TestCrewHandler_UpdateKYC -v ./internal/handler/...  # KYC handler tests
go test -run TestEmployeePayout -v ./internal/service/...  # Transaction tests only
make test-coverage                        # HTML coverage report
```

---

## 10. Infrastructure

### 10.1 Docker Compose Services

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| app | Custom Dockerfile | 8080 | API server |
| postgres | postgres:16-alpine | 5432 | Primary database |
| redis | redis:7-alpine | 6379 | Cache + job queue backend |
| minio | minio/minio:latest | 9000/9001 | Object storage (API/Console) |
| asynqmon | hibiken/asynqmon | 8081 | Background job dashboard |

### 10.2 Dockerfile

- Multi-stage build: `golang:1.24-alpine` → `alpine:3.20`
- Non-root user (`appuser:appgroup`, UID/GID 1001)
- Health check: `wget -qO- http://localhost:8080/health`
- Migrations copied into runtime image

### 10.3 HTTP Server Configuration

| Setting | Value |
|---------|-------|
| Read Timeout | 30s |
| Write Timeout | 30s |
| Idle Timeout | 60s |
| Graceful Shutdown | 30s drain |
| DB Pool Max Open | 25 |
| DB Pool Max Idle | 10 |
| DB Conn Max Lifetime | 5 min |
| DB Conn Max Idle Time | 1 min |

---

## 11. Existing Gaps & Recommendations

### 11.1 ✅ Resolved Critical Gaps

| # | Gap | Resolution |
|---|-----|------------|
| 1 | ~~No database transactions in services~~ | ✅ `TxManager` in `database/tx.go` — context-injected transactions. All repos tx-aware via `getDB(ctx)`. Auth registration and assignment completion are atomic. |
| 2 | ~~External integrations are empty stubs~~ | ✅ Strategy-pattern clients implemented: SMS (Optimize + Africa's Talking), Payment (JamboPay v2), Payroll (PerPay), Identity (IPRS). |
| 3 | ~~Background worker system is empty~~ | ✅ `worker/scheduler.go` + `worker/daily_summary.go` — goroutine-based scheduler with graceful shutdown and daily earnings aggregation. |
| 4 | ~~No SACCO-scoped data isolation~~ | ✅ Handlers enforce SACCO-scoped filtering for `SACCO_ADMIN` users using JWT `sacco_id` claim. |
| 5 | ~~Wallet balance access not scoped~~ | ✅ `enforceWalletAccess()` in handlers — CREW users can only access their own wallet. |
| 6 | ~~No payroll service~~ | ✅ `PayrollService` fully implemented with PerPay integration + statutory deduction calculations. |
| 7 | ~~No SACCO/Vehicle/Route handlers~~ | ✅ All CRUD services and HTTP handlers implemented for core logistics entities. |
| 8 | ~~CORS allows all origins~~ | ✅ Read `cfg.CORSAllowedOrigins` and apply dynamically in middleware. |
| 9 | ~~Rate limiter is in-memory only~~ | ✅ Replaced with Redis-based sliding window limiter for horizontal scale. |
| 10 | ~~Metrics are in-memory~~ | ✅ Use `sync/atomic` counters and wired into Prometheus exporter. |
| 11 | ~~No audit logging implementation~~ | ✅ `AuditService` implemented and injected into critical paths (Wallet, SACCO, Loans). |
| 12 | ~~Notification system not wired~~ | ✅ `NotificationService` handles event-driven SMS and in-app alerts with opt-in preferences. |
| 13 | ~~MinIO client not wired to handlers~~ | ✅ Fully integrated into `DocumentHandler` for secure file storage/retrieval. |
| 14 | ~~`pkg/validator/` is empty~~ | ✅ Implemented centralized validator for Phone, National ID, and Amounts. |
| 15 | ~~scripts/ directory is empty~~ | ✅ Added `cmd/seed/main.go` with idempotent test data. |
| 16 | ~~Align Go versions~~ | ✅ Dockerfile and `go.mod` synchronized to Go 1.25. |
| 17 | ~~`KYCVerifiedAt` timestamp never set~~ | ✅ Fixed in `crew_service.go` — properly set on verification. |
| 18 | ~~Swagger docs are minimal stubs~~ | ✅ Regenerated full suite from handler annotations via `make swagger`. |
| 19 | ~~No request validation middleware~~ | ✅ Centralized validator used alongside struct-tag binding. |
| 20 | ~~Timeout middleware doesn't cancel context~~ | ✅ Uses `context.WithTimeout` for true deadline enforcement across service calls. |
| 21 | ~~Metrics data race~~ | ✅ Switched to `atomic.AddInt64` for thread-safe request counting. |
| 22 | ~~No CI/CD pipeline~~ | ✅ GitHub Actions configured for lint, test (-race), and docker build. |
| 23 | ~~No database seeding~~ | ✅ Idempotent seed script implemented for dev/staging environments. |
| 24 | ~~Missing mock repositories~~ | ✅ 100% Mock parity achieved (19/19) for all repository interfaces. |
| 25 | ~~Financial services logic missing~~ | ✅ Fully implemented Credit Scoring, Loan management, and Insurance workflows. |
| 26 | ~~No KYC unverification workflow~~ | ✅ Admins can unverify employees with a reason; IN_APP notification dispatched; `KYCVerifiedAt` cleared; frontend navigation blocked for unverified employees (KYC guard + sidebar lock). |

### 11.2 Roadmap & Future Recommendations

With the core system at 100% feature parity, future work should focus on:
- **Mobile App Integration:** Finalizing PUSH notification providers.
- **Reporting Engine:** Advanced PDF/Excel generation for SACCO-wide financial reports.
- **Advanced Fraud Detection:** Real-time analysis of assignment locations vs. route geo-fencing.

### 10.4 Gap Priority Matrix

---

## 12. API Response Format

### Success
```json
{ "success": true, "data": { ... } }
```

### List (Paginated)
```json
{ "success": true, "data": [...], "meta": { "page": 1, "per_page": 20, "total": 150, "total_pages": 8 } }
```

### Error
```json
{ "success": false, "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

---

## 13. Environment Configuration

Required variables validated at startup:

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `REDIS_URL` | ✅ | — | Redis connection string |
| `JWT_SECRET` | ✅ | — | ≥32 characters |
| `MINIO_ENDPOINT` | ✅ | — | MinIO server address |
| `MINIO_ACCESS_KEY` | ✅ | — | MinIO credentials |
| `MINIO_SECRET_KEY` | ✅ | — | MinIO credentials |
| `JAMBOPAY_CLIENT_ID` | Prod only | — | Required when `ENVIRONMENT=production` |
| `SMS_CLIENT_ID` or `AT_API_KEY` | Prod only | — | At least one SMS provider required in production |
| `PORT` | ❌ | 8080 | HTTP server port |
| `ENVIRONMENT` | ❌ | development | `development` / `staging` / `production` |
| `JWT_EXPIRY_MINUTES` | ❌ | 15 | Access token TTL |
| `JWT_REFRESH_DAYS` | ❌ | 7 | Refresh token TTL |
| `MINIO_BUCKET` | ❌ | amy-mis | Default bucket name |
| `RATE_LIMIT_RPM` | ❌ | 100 | Requests per minute per IP |
| `SMS_CLIENT_ID` | ❌ | — | Optimize SMS OAuth2 client ID |
| `SMS_CLIENT_SECRET` | ❌ | — | Optimize SMS OAuth2 client secret |
| `SMS_TOKEN_URL` | ❌ | — | Optimize SMS token endpoint |
| `SMS_URL` | ❌ | — | Optimize SMS send endpoint |
| `SMS_SENDER_ID` | ❌ | AMY-MIS | SMS sender name |
| `AT_USERNAME` | ❌ | sandbox | Africa's Talking username |
| `JAMBOPAY_CLIENT_SECRET` | ❌ | — | JamboPay OAuth2 secret |
| `JAMBOPAY_BASE_URL` | ❌ | — | JamboPay API base URL |
| `PERPAY_CLIENT_ID` | ❌ | — | PerPay OAuth2 client ID |
| `PERPAY_CLIENT_SECRET` | ❌ | — | PerPay OAuth2 secret |
| `PERPAY_BASE_URL` | ❌ | — | PerPay API base URL |
| `IPRS_CLIENT_ID` | ❌ | — | IPRS OAuth2 client ID |
| `IPRS_CLIENT_SECRET` | ❌ | — | IPRS OAuth2 secret |
| `IPRS_BASE_URL` | ❌ | — | IPRS API base URL |
| `IPRS_TOKEN_ENDPOINT` | ❌ | — | IPRS OAuth2 token endpoint |

---

*Document updated from source code analysis on 2026-05-07.*
