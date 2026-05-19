# AMY MIS — Backend System Documentation

> **Version:** 1.5 | **Last Updated:** 2026-05-19 | **Go:** 1.25 | **Framework:** Gin 1.12

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
│                                                  │  Metrics (Atomic), Logger, Recovery, Security,
│                                                  │  Maintenance Mode
├─────────────────────────────────────────────────┤
│                  handler/ + dto/                 │  16 HTTP handler files + structured DTOs
├─────────────────────────────────────────────────┤
│                  service/                        │  24 Business logic services
├─────────────────────────────────────────────────┤
│                  repository/ (interfaces)        │  21 Data access contracts
│                  repository/postgres/            │  Postgres implementation (GORM)
│                  repository/mock/                │  100% Mock parity (19/19)
├─────────────────────────────────────────────────┤
│                  models/                         │  17 GORM entity models
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
│   │   ├── interfaces.go       — 21 repository interfaces
│   │   ├── postgres/           — GORM implementations
│   │   └── mock/               — 21 mocks (100% coverage)
│   ├── service/                — 24 business logic services
│   ├── handler/                — 16 handler files + DTOs
│   ├── middleware/             — 9 middleware layers (Auth, Metrics, Maintenance, etc.)
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
    → [Public routes: health, auth, swagger, system/status]
    → [Secured routes: JWTAuth → RBAC → MaintenanceMode]
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
11. Initialize 18 handlers (including SupportHandler, SystemSettingsHandler)
12. Initialize 4 background workers (Scheduler)
13. Configure Gin router + register middleware + routes (including MaintenanceMode on secured group)
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
SystemSettings (key-value platform config: feature flags, maintenance mode)
SystemAnnouncements (platform-wide user banners)
Roles / Permissions / RolePermissions (RBAC engine)
RoleTemplates (industry bootstrap templates)
```

### 3.2 Migration Sets (8 total, 27+ tables)

| # | Domain | Tables |
|---|--------|--------|
| 1 | Users | `users` |
| 2 | Identity Registry | `saccos`, `crew_members`, `routes`, `vehicles`, `crew_sacco_memberships` + FK constraints |
| 3 | Operations | `assignments`, `earnings`, `daily_earnings_summaries` |
| 4 | Financial | `wallets`, `wallet_transactions`, `sacco_floats`, `sacco_float_transactions` |
| 5 | Payroll | `payroll_runs`, `payroll_entries`, `statutory_rates` |
| 6 | Infrastructure | `webhook_events`, `notifications`, `notification_templates`, `documents`, `audit_logs` |
| 7 | Financial Services | `credit_scores`, `loan_applications`, `insurance_policies` |
| 8 | Platform Admin | `system_settings`, `system_announcements`, `roles`, `permissions`, `role_permissions`, `role_templates`, `work_sites`, `tenant_job_types`, `pay_schedules`, `pay_periods` |

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
| System Admin | `SYSTEM_ADMIN` | Full access to all resources. Only role that bypasses maintenance mode. |
| Platform Admin | `PLATFORM_ADMIN` | Platform-wide administration (settings, support, finance). Blocked during maintenance. |
| Platform Support | `PLATFORM_SUPPORT` | Support Center access — user lookup, OTP resend, wallet recovery |
| Platform Finance | `PLATFORM_FINANCE` | Financial oversight — wallet reconciliation, payroll review, reporting |
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

| Route Group | Auth Required | Role Required | KYC Required | Maintenance Exempt |
|-------------|--------------|---------------|---------------|--------------------|
| `/health`, `/ready`, `/metrics` | ❌ | — | — | ✅ |
| `/swagger/*` | ❌ | — | — | ✅ |
| `/api/v1/auth/register,login,refresh` | ❌ | — | — | ✅ |
| `/api/v1/system/status` | ❌ | — | — | ✅ (public) |
| `/api/v1/auth/me` | ✅ JWT | Any | ❌ | ❌ (blocked) |
| `/profile`, `/notifications` | ✅ JWT | Any | ❌ (KYC-exempt) | ❌ (blocked) |
| `/api/v1/crew/*` | ✅ JWT | SYSTEM_ADMIN or EMPLOYER | ❌ (admin routes) | ❌ |
| `/api/v1/assignments/*` | ✅ JWT | SYSTEM_ADMIN or EMPLOYER | ❌ (admin routes) | ❌ |
| `/api/v1/admin/*` | ✅ JWT | SYSTEM_ADMIN or PLATFORM_ADMIN | — | Only SYSTEM_ADMIN bypass |
| `/api/v1/admin/support/*` | ✅ JWT | PLATFORM_SUPPORT+ | — | Only SYSTEM_ADMIN bypass |
| `/system-admin` (frontend) | ❌ | — | — | ✅ (always accessible, backdoor admin login) |
| `/dashboard`, `/earnings`, `/wallets`, etc. | ✅ JWT | Any | ✅ (employees) | ❌ (blocked) |
| `/api/v1/wallets/:id` (GET) | ✅ JWT | Any (ownership enforced) | — | ❌ |
| `/api/v1/transactions/employee-payout` | ✅ JWT | SYSTEM_ADMIN or EMPLOYER | — | ❌ |
| `/api/v1/transactions/transfer` | ✅ JWT | Any (sender derived from JWT) | — | ❌ |

---

## 5. Middleware Stack

Applied in order on every request:

| # | Middleware | Purpose | Scope |
|---|-----------|---------|-------|
| 1 | CORS | Allows all origins (dev), exposes `X-Request-Id`, `X-Response-Time` | Global |
| 2 | SecureHeaders | `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`, `Cache-Control` | Global |
| 3 | RequestID | UUID per request, propagated via `X-Request-ID` header | Global |
| 4 | RateLimit | 100 req/min per IP, in-memory sliding window with goroutine cleanup | Global |
| 5 | Timeout | 30s deadline tracking (header-based, not context-based) | Global |
| 6 | Metrics | In-memory request counters, per-path latency histograms | Global |
| 7 | Logger | Structured `slog` logging with method, path, status, latency, IP | Global |
| 8 | Recovery | Panic recovery with full stack trace logging | Global |
| 9 | **MaintenanceMode** | Checks `maintenance.active` system setting. Returns 503 for all non-SYSTEM_ADMIN users. Only `SYSTEM_ADMIN` bypasses. Auth, health, and `/system/status` endpoints are always allowed. | Secured group only |

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
| **JamboPay v2** | OAuth2 client_credentials → Bearer token | `InitiateCollection` (STK push), `InitiatePayout` (M-Pesa B2C/bank/paybill), `VerifyPayout` (OTP), `CheckBalance`, `VerifyBankTransfer` |

- **Channels:** `MOMO_B2C` (mobile money), `BANK` (bank transfer), `MOMO_B2B` (paybill/till)
- **Collection flow (STK push):** `InitiateCollection` → phone prompt → user enters PIN → JamboPay callback → `ConfirmPendingTopUp`
- **Payout OTP flow:** `InitiatePayout` → `pending_otp` → `VerifyPayout(ref, otp)` → `completed`
- **Bank verification:** `VerifyBankTransfer(BankVerificationRequest)` → returns `VERIFIED`, `NOT_FOUND`, `MISMATCH`, or `UNAVAILABLE`. Providers that don't support verification return `ErrNotImplemented` and are skipped by the Manager.

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

Organization (SACCO) float can be funded via three methods. **Mobile money** uses an asynchronous STK push flow; **bank** and **card** use a configurable verification workflow.

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
- The float balance is **never** credited until the payment is confirmed via callback or polling
- Pending transactions are idempotency-protected (`idempotency_key` as the JamboPay `orderId`)
- The system supports multiple synchronization methods tracked via `sync_method` (`CALLBACK`, `POLL`, `MANUAL`) and `synced_at` timestamps on the transaction
- Frontend dashboards actively poll JamboPay for STK push completion using `/organizations/:id/float/poll-stk`, enabling faster confirmation when webhooks are delayed
- Failed STK pushes immediately mark the pending tx as `FAILED`
- The webhook handler first checks for pending float tx (collection), then payout tx

#### Tenant-Level Top-Up Configuration

Top-up availability is governed by `TenantConfig.AllowedTopUpMethods` (`mobile_money`, `bank`, `card`). Handlers validate the requested method against this configuration and return `403 METHOD_DISABLED` if a tenant has disabled a specific channel. By default (nil/empty array), all methods are allowed.

#### Bank & Card (Configurable Verification)

Bank and card top-ups use a **tenant-configurable verification workflow** controlled by `TenantConfig.TopUpVerificationMode`. This prevents unauthorized float inflation from unverified bank references.

| Mode | Behavior | Response |
|------|----------|----------|
| **HYBRID** (default) | Try bank API verification first; if API unavailable, fall back to manual admin approval | `201 Created` (API verified) or `202 Accepted` (pending manual) |
| **API** | Strictly verify via bank API; reject if API unavailable or reference invalid | `201 Created` (verified) or `422`/`503` (rejected) |
| **MANUAL** | All top-ups create PENDING transactions requiring admin confirmation | `202 Accepted` (always pending) |

```
 Bank top-up submitted
   │
   ▼
 Load TenantConfig → ResolvedTopUpVerificationMode()
   │
   ├── API mode ──────────────────────────────────────────┐
   │   ▼                                                  │
   │   paymentMgr.VerifyBankTransfer(ref, amount)         │
   │   ├── VERIFIED → CreditFloat immediately (201)       │
   │   ├── NOT_FOUND/MISMATCH → Reject (422)              │
   │   └── UNAVAILABLE → Block (503)                      │
   │                                                      │
   ├── HYBRID mode ───────────────────────────────────────┤
   │   ▼                                                  │
   │   paymentMgr.VerifyBankTransfer(ref, amount)         │
   │   ├── VERIFIED → CreditFloat immediately (201)       │
   │   ├── NOT_FOUND/MISMATCH → Create PENDING (202)      │
   │   └── UNAVAILABLE → Create PENDING (202)             │
   │                                                      │
   └── MANUAL mode ───────────────────────────────────────┘
       ▼
       Create PENDING transaction (202)
       ▼
       Admin reviews in Wallet Dashboard
       ├── Confirm → POST /:id/float/topup/:tx_id/confirm → Credit balance
       └── Reject  → POST /:id/float/topup/:tx_id/reject  → Mark FAILED
```

**Admin approval endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/organizations/:id/float/topup/:tx_id/confirm` | Confirm PENDING top-up → atomically credit float balance |
| `POST` | `/organizations/:id/float/topup/:tx_id/reject` | Reject PENDING top-up with reason → mark FAILED, no balance change |

**Configuration:** Admins set the verification mode in **Tenant Settings → Finance** tab. The setting is stored in `TenantConfig.topup_verification_mode` (JSONB) and read by the handler on each top-up request.

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
| `PENDING` | Awaiting confirmation — STK push, bank API verification, or manual admin review |
| `COMPLETED` | Payment confirmed (via callback, API, or admin), balance updated |
| `FAILED` | Payment failed, STK push error, or admin rejected |
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
4. **Pending→Confirm pattern:** For STK push and bank/card top-ups, a `PENDING` record is created first (no balance change), then atomically confirmed when verified — preventing premature balance inflation
5. **Configurable verification:** Bank/card top-ups are gated by tenant-level `TopUpVerificationMode` (`API`, `MANUAL`, `HYBRID`). API mode verifies references via bank integration before crediting. HYBRID tries API first and falls back to manual admin approval. MANUAL always requires admin confirmation.
6. **Context-injected transactions:** Float repo uses `getDB(ctx)` to participate in externally-managed DB transactions (e.g., from `TransactionService`)

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
| `ErrServiceUnavailable` | 503 | `MAINTENANCE` |

---

## 9. Testing

### 9.1 Test Coverage

- **440+ tests** across **21 test packages** (all passing)
- **55 test files** covering services, handlers, middleware, integrations, and workers
- **Mock repositories** for User, Crew, Wallet, Organization Float, SystemSetting, SystemAnnouncement, and all others with thread-safe operations
- **httptest servers** for all external integration tests (no real API calls)
- **Support Handler tests:** 14 dedicated tests covering search, timeline, OTP resend, and error paths

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
| **Support Center** | User search (server-side ILIKE), user timeline, OTP resend (audit-logged), non-existent user errors |
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
| 27 | ~~Bank/card top-ups credited without verification~~ | ✅ Configurable verification system with 3 modes: **API** (verify via bank integration), **MANUAL** (admin approval), **HYBRID** (try API, fall back to manual). Tenant-level config in `TenantConfig.topup_verification_mode`. Admin confirm/reject endpoints. Frontend Pending Approvals panel in Wallet Dashboard. Finance tab in Tenant Settings. |
| 28 | ~~No control over available top-up methods~~ | ✅ Tenant-level configuration `AllowedTopUpMethods` implemented. Handlers enforce availability dynamically. Frontend admin panel allows enabling/disabling of mobile money, bank, and card top-ups per tenant. |
| 29 | ~~Silent failures on delayed JamboPay callbacks~~ | ✅ Active polling mechanism implemented. `SyncMethod` (`CALLBACK`, `POLL`, `MANUAL`) and `SyncedAt` added to transactions. Frontend Wallet Dashboard auto-polls pending STK transactions to provide real-time status updates and manual resolution options. |
| 30 | ~~Static admin filters lack searchability~~ | ✅ Replaced plain select dropdowns with dynamic Autocomplete components in Compliance, Documents, and Team modules for enhanced searchability and UX. |
| 31 | ~~No API Key or Integration management~~ | ✅ Added IntegrationHandler with secure API Key generation (masked storage via SystemSettings), provider toggling, and health status reporting. Frontend integrations page is fully operational. |
| 32 | ~~No Support Center Dashboard~~ | ✅ Added Platform Support Component with User Lookup, Wallet Recovery, Payroll Reprocessing, Quick Actions, and Activity Timeline. Fixed backend struct models for `SystemStats` and `AuditLog` fields, and `OTPService` methods, to ensure robust full-stack support actions. |
| 33 | ~~User search returns all users~~ | ✅ Extended `UserRepository.List()` to accept a `search` parameter. Postgres implementation uses `ILIKE` filtering on `phone` and `email`. `AdminHandler.ListUsers` extracts `phone`/`email`/`search` query params. Frontend now relies on server-side filtering, replacing the O(n) client-side scan. |
| 34 | ~~Maintenance mode is purely cosmetic~~ | ✅ Full maintenance lockdown implemented. Backend: `MaintenanceMode` middleware returns 503 for all non-`SYSTEM_ADMIN` users. Frontend: `maintenanceGuard` on all routes redirects to a dedicated `/maintenance` page (professional full-page design with animations, contact info, and auto-recovery polling). HTTP interceptor catches 503 `MAINTENANCE` responses. A `/system-admin` backdoor login page is always accessible and rejects non-SYSTEM_ADMIN roles after authentication. |
| 35 | ~~Announcements not visible to employers/employees~~ | ✅ Root cause: CSS `position: fixed` overlapped with the `margin-top` of the Dashboard content area, causing the banner text to be hidden behind the main heading. Additionally, dismissing a banner stored the state in `sessionStorage` which persisted across different user logins on the same browser (common in testing). Fixed by moving the banner into the normal document flow inside `<main>` with negative margins, and clearing `sessionStorage` in `auth.service.ts` on logout. Added `X-Skip-Error-Toast` headers for background polling (notifications/org load) to prevent 403 toasts for employees. |
| 36 | ~~Payroll mock missing DeleteEntries~~ | ✅ Added `DeleteEntries(ctx, runID)` to `mock.PayrollRepo` to fix pre-existing test compilation errors in the `internal/service` package. |
| 37 | ~~USSD mock uses stale List signature~~ | ✅ Updated `mockUserRepo.List` in `ussd/session_handler_test.go` to include the new `search string` parameter. |

### 11.2 Roadmap & Future Recommendations

With the core system at 100% feature parity, future work should focus on:
- **Mobile App Integration:** Finalizing PUSH notification providers.
- **Reporting Engine:** Advanced PDF/Excel generation for SACCO-wide financial reports.
- **Advanced Fraud Detection:** Real-time analysis of assignment locations vs. route geo-fencing.
- **Scheduled Maintenance Windows:** Auto-enable/disable maintenance based on `maintenance.start` and `maintenance.end` timestamps.
- **Granular Support Permissions:** Role-based support actions (e.g., only supervisors can credit wallets, agents can resend OTPs).
- **Announcement Targeting:** Allow announcements to target specific user roles, organizations, or regions.

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
| `SERVICE_API_KEY` | ❌ | — | API key for service-to-service authentication |
| `CORS_ALLOWED_ORIGINS` | ❌ | * | Comma-separated allowed CORS origins |

---

## 14. Platform Administration Features

### 14.1 Maintenance Mode

Platform admins can enable **Maintenance Mode** from the Settings → Maintenance tab. When active, the platform enters a **full lockdown**:

#### Backend Enforcement

1. **Middleware** (`MaintenanceMode`) intercepts all secured API requests
2. Non-admin users receive a `503 Service Unavailable` response with code `MAINTENANCE`
3. **Only `SYSTEM_ADMIN`** users bypass the check — all other roles (including `PLATFORM_ADMIN`) are blocked
4. Public endpoints (`/auth/*`, `/system/status`, `/health`) are always accessible

#### Frontend Lockdown

1. A **`maintenanceGuard`** on all routes checks `/api/v1/system/status` and redirects to `/maintenance`
2. The `/maintenance` page is a **full-page, dedicated maintenance screen** with:
   - Animated dark background with floating particles and glowing orbs
   - Company branding with spinning ring animation around logo
   - "SYSTEM MAINTENANCE" status badge with pulsing indicator
   - "We'll Be Back Shortly" hero text with gradient
   - Rotating gear animations
   - Emergency contact cards: phone, email, WhatsApp
3. **No login form, no sidebar, no menu** — complete visual lockdown
4. The page **auto-polls** `/system/status` every 30 seconds and redirects to login when maintenance ends
5. The **HTTP interceptor** catches any `503 MAINTENANCE` responses and redirects to `/maintenance`

#### System Admin Backdoor (`/system-admin`)

A dedicated admin login page at `/system-admin` is **always accessible**, even during maintenance:

- **Not guarded** by `maintenanceGuard` — whitelisted at the route level
- Amber/gold themed login form clearly marked "System Administrator — Restricted access"
- Validates role after login — only `SYSTEM_ADMIN` is allowed through; all other roles are rejected with "Access denied" and immediately logged out
- "Back to maintenance page" link returns to `/maintenance`
- The `auth.interceptor` uses `isExemptPage()` check (via `window.location.pathname`) to suppress both 503→redirect and 401→logout on `/system-admin` and `/maintenance` pages, preventing redirect loops
- `AppComponent.ngOnInit()` skips `fetchProfile()` when on exempt pages to avoid triggering 503 responses during bootstrap

**System Settings used:**

| Key | Type | Description |
|-----|------|-------------|
| `maintenance.active` | `bool` | Whether maintenance mode is enabled |
| `maintenance.message` | `string` | Message shown to users |
| `maintenance.start` | `string` | Scheduled start time (informational) |
| `maintenance.end` | `string` | Scheduled end time (informational) |

### 14.2 System Announcements

Platform-wide announcements are managed via the Settings → Announcements tab and displayed to all authenticated users via the `AnnouncementBannerComponent`.

**Severity levels:** `INFO` (purple), `WARNING` (amber), `CRITICAL` (red)

**Visibility rules:**
- Announcements with `is_active = true` and no date bounds (NULL `start_at`/`end_at`) are always visible
- Announcements with date bounds are only visible during the specified window
- Users can dismiss banners per-session (persisted in `sessionStorage`)
- The banner component polls for new announcements every 5 minutes

**API Endpoints:**

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/announcements/active` | ✅ JWT | Returns active announcements for the current user |
| `GET` | `/api/v1/admin/announcements` | ✅ Admin | List all announcements (paginated) |
| `POST` | `/api/v1/admin/announcements` | ✅ Admin | Create announcement |
| `PUT` | `/api/v1/admin/announcements/:id` | ✅ Admin | Update announcement |
| `DELETE` | `/api/v1/admin/announcements/:id` | ✅ Admin | Delete announcement |

### 14.3 Support Center

The Support Center (`/platform/support`) provides first-line customer support capabilities:

| Feature | Endpoint | Description |
|---------|----------|-------------|
| **User Search** | `GET /admin/support/search?q=...` | Server-side ILIKE search on phone and email |
| **User Timeline** | `GET /admin/support/users/:id/timeline` | Audit log filtered by user ID |
| **Resend OTP** | `POST /admin/support/users/:id/resend-otp` | Triggers OTP dispatch with mandatory audit logging |
| **Disable Account** | `POST /admin/users/:id/disable` | Disables user login (with confirmation dialog) |
| **Enable Account** | `POST /admin/users/:id/enable` | Re-enables user login (with confirmation dialog) |
| **Wallet Credit** | `POST /wallets/credit` | Manual wallet adjustment (with confirmation dialog) |

**Security measures:**
- All support actions generate audit log entries
- Destructive actions (disable, wallet credit) require frontend confirmation dialogs
- Resend OTP is logged with the admin's user ID for compliance traceability

### 14.4 System Settings (Key-Value Store)

The `system_settings` table stores global platform configuration using dot-notation namespacing:

| Category | Example Keys | Purpose |
|----------|-------------|--------|
| `feature.*` | `feature.loans_enabled`, `feature.ussd_enabled` | Feature flags (boolean toggles) |
| `maintenance.*` | `maintenance.active`, `maintenance.message` | Maintenance mode configuration |
| `defaults.*` | `defaults.kyc_required`, `defaults.topup_verification_mode` | Default tenant configuration |

**API Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/admin/system-settings?prefix=...` | List settings (optional prefix filter) |
| `PUT` | `/api/v1/admin/system-settings` | Upsert single setting |
| `PUT` | `/api/v1/admin/system-settings/bulk` | Bulk upsert multiple settings |
| `DELETE` | `/api/v1/admin/system-settings/:key` | Delete a setting |
| `GET` | `/api/v1/system/status` | **Public** — returns maintenance state (no auth) |

---

*Document updated from source code analysis on 2026-05-19.*
