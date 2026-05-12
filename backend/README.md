# AMY MIS — Backend API

> **A Workforce Financial Operating System for Kenya's Informal Transport Sector**

AMY MIS digitizes the financial lifecycle of matatu (minibus), boda-boda (motorcycle), and tuk-tuk crews — from shift assignments and earnings calculation to wallet management, payroll processing, and SACCO operations.

---

## 🚀 Quick Start

### Prerequisites

- **Go** 1.25+
- **PostgreSQL** 16+
- **Redis** 7+
- **MinIO** (or S3-compatible storage)
- **Docker & Docker Compose** (recommended)

### 1. Clone & Configure

```bash
git clone https://github.com/Chemokoren/crew.git
cd crew/backend
cp .env .env.local
# Edit .env.local with your credentials
```

### 2. Run with Docker Compose

```bash
docker compose up -d        # Start PostgreSQL, Redis, MinIO
go run cmd/server/main.go   # Start the API server
```

### 3. Verify

```bash
curl http://localhost:8080/health     # → {"status":"healthy"}
curl http://localhost:8080/ready      # → {"status":"ready"}
curl http://localhost:8080/swagger/index.html  # → Swagger UI
```

---

## 📡 API Endpoints

### Authentication (Public)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/auth/register` | Register a new user |
| `POST` | `/api/v1/auth/login` | Login with phone + password |
| `POST` | `/api/v1/auth/refresh` | Refresh JWT token pair |

### User Profile (JWT Required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/auth/me` | Get current user profile |

### Crew Members (Admin/SACCO)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/crew` | Create crew member |
| `GET` | `/api/v1/crew` | List crew (filter: role, kyc, search) |
| `GET` | `/api/v1/crew/:id` | Get crew member by ID |
| `PUT` | `/api/v1/crew/:id/kyc` | Update KYC status (supports `reason` for unverification/rejection) |
| `DELETE` | `/api/v1/crew/:id` | Deactivate crew member |

### Assignments (Admin/SACCO)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/assignments` | Create shift assignment |
| `GET` | `/api/v1/assignments` | List (filter: sacco, crew, date, status) |
| `GET` | `/api/v1/assignments/:id` | Get assignment details |
| `POST` | `/api/v1/assignments/:id/complete` | Complete & calculate earnings |

### Wallets (JWT Required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/wallets/:crew_member_id` | Get wallet balance |
| `GET` | `/api/v1/wallets/:crew_member_id/transactions` | Transaction history |
| `POST` | `/api/v1/wallets/credit` | Credit wallet (ownership enforced) |
| `POST` | `/api/v1/wallets/debit` | Debit wallet (ownership enforced) |

### Atomic Transactions (JWT Required)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/transactions/employee-payout` | Atomic: debit org float (gross) + credit wallet (net) — Admin only |
| `POST` | `/api/v1/transactions/transfer` | Atomic: debit sender + credit recipient wallet |

### Organization Float (Admin)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/organizations/:id/float` | Get float balance |
| `POST` | `/api/v1/organizations/:id/float/topup` | Top up float (mobile/bank/card) |
| `POST` | `/api/v1/organizations/:id/float/topup/:tx_id/confirm` | Admin: confirm pending top-up |
| `POST` | `/api/v1/organizations/:id/float/topup/:tx_id/reject` | Admin: reject pending top-up |
| `POST` | `/api/v1/organizations/:id/float/credit` | Direct credit float |
| `POST` | `/api/v1/organizations/:id/float/debit` | Debit float (payout) |
| `GET` | `/api/v1/organizations/:id/float/transactions` | Float transaction history |

### Webhooks (Public — checksum-verified)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/webhooks/jambopay` | JamboPay collection + payout callbacks |
| `POST` | `/api/v1/webhooks/perpay` | PerPay payroll callbacks |

### System

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/ready` | Readiness probe |
| `GET` | `/metrics` | Request metrics (JSON) |
| `GET` | `/` | Redirects to Swagger API docs (`/swagger/index.html`) |
| `GET` | `/swagger/*` | Swagger API docs |

---

## 🏗️ Architecture

```
cmd/server/              — Entry point + dependency wiring (15-step startup)
internal/
  config/                — Environment configuration + validation
  database/              — PostgreSQL (GORM) + Redis + TxManager
  handler/               — HTTP handlers + DTOs (request/response)
  handler/dto/           — Data Transfer Objects
  middleware/            — Auth (JWT), RBAC, CORS, rate limiting,
                           metrics, security headers, logging, recovery
  models/                — GORM data models (15 entities)
  repository/            — Data access interfaces
  repository/postgres/   — GORM repository implementations (tx-aware)
  repository/mock/       — In-memory mocks for testing
  service/               — Business logic layer (transactional)
  worker/                — Background jobs (Scheduler + DailySummaryJob)
  external/
    sms/                 — SMS Strategy: Optimize (default) + Africa's Talking (fallback)
    payment/             — Payment Strategy interface
    jambopay/            — JamboPay v2: M-Pesa B2C, bank, paybill payouts
    payroll/             — Payroll Strategy interface
    perpay/              — PerPay: async payroll submission + status polling
    identity/            — Identity Strategy interface
    iprs/                — IPRS: national ID verification (OAuth2 via JamboPay IdP)
    storage/             — MinIO (S3-compatible) file storage
pkg/
  errs/                  — Shared domain errors
  jwt/                   — JWT token manager
  money/                 — Cents ↔ KES conversion utilities
  pagination/            — Pagination helpers
  types/                 — System roles + shared types
docs/                    — Swagger/OpenAPI generated docs
migrations/              — PostgreSQL migration files (7 sets, 22 tables)
```

### Design Principles

- **Clean Architecture**: Handlers → Services → Repositories (all via interfaces)
- **Strategy Pattern**: All external integrations (SMS, Payment, Payroll, Identity) use a common Provider interface — swap or stack providers without code changes
- **Transactional Integrity**: Multi-step financial operations (employee payout, wallet transfers) wrapped in database transactions via `TxManager`. Both float and wallet repos participate in externally-managed transactions via context injection (`getDB(ctx)`).
- **Financial Safety**: All money stored as `int64` cents — no floats in the pipeline
- **Wallet Concurrency**: `SELECT ... FOR UPDATE` + optimistic version checks
- **Idempotency**: Financial endpoints require `Idempotency-Key` header or `idempotency_key` in JSON body. Derived keys ensure both sides of atomic operations are individually idempotent.
- **Atomic Payouts**: Employee payouts (float debit + wallet credit) and wallet-to-wallet transfers execute in a single DB transaction — if either side fails, everything rolls back
- **SACCO-Scoped Isolation**: SACCO_ADMIN users see only their own SACCO's data
- **Ownership Enforcement**: CREW users can only access their own wallet

---

## 🔌 External Integrations

All integrations use the **Strategy design pattern** with automatic fallback chains and runtime provider switching.

| Integration | Provider(s) | Auth Method | Key Operations |
|------------|-------------|-------------|----------------|
| **SMS** | Optimize (default), Africa's Talking (fallback) | OAuth2 → JWT / API key header | `Send`, `SendBulk`, runtime `SetPrimary` |
| **Payment** | JamboPay v2 | OAuth2 client_credentials | `InitiateCollection` (STK push), `InitiatePayout` (M-Pesa/bank/paybill), `VerifyPayout` (OTP), `CheckBalance`, `VerifyBankTransfer` |
| **Payroll** | PerPay | JWT (15min TTL) | `SubmitPayroll` (async 202), `GetStatus` (polling), idempotency replay |
| **Identity** | IPRS | OAuth2 via JamboPay IdP (scope=iprs) | `VerifyCitizen` (national ID → name, DOB, photo) |

### Adding a New Provider

1. Implement the interface (e.g. `sms.Provider`)
2. Register in `main.go`: `smsProviders = append(smsProviders, sms.NewTwilioProvider(cfg, logger))`
3. Done — it's automatically part of the fallback chain

---

## 💰 Earning Models

| Model | Formula | Use Case |
|-------|---------|----------|
| `FIXED` | `fixed_amount_cents` | Daily flat rate |
| `COMMISSION` | `revenue × commission_rate` | Percentage of collections |
| `HYBRID` | `base_cents + (revenue × commission_rate)` | Base pay + commission |

---

## 💳 Organization Float Top-Up Flow

Float can be funded via mobile money (M-Pesa STK push), bank transfer, or card. Bank and card top-ups use a **configurable verification workflow** (API, Manual, or Hybrid):

| Method | Flow | Response |
|--------|------|----------|
| **Mobile Money** | Create PENDING tx → STK push → User enters PIN → Callback confirms → Balance credited | `202 Accepted` (async) |
| **Bank (HYBRID/default)** | Try bank API verification → if API unavailable, create PENDING for admin review | `201 Created` or `202 Accepted` |
| **Bank (API mode)** | Verify via bank API → reject if API unavailable or ref invalid | `201 Created` or `422`/`503` |
| **Bank (MANUAL mode)** | Always create PENDING → admin confirms/rejects in Wallet Dashboard | `202 Accepted` |
| **Card** | Same as bank MANUAL mode → admin confirms/rejects | `202 Accepted` |

**Verification mode** is tenant-configurable via **Tenant Settings → Finance** tab (`TenantConfig.topup_verification_mode`).

**Key safety guarantee:** For mobile money, the float balance is **never** credited until the payment provider confirms via webhook callback. For bank/card, the balance is **never** credited until verified by API or manually confirmed by an admin.

**Float transaction types:** `FUND` (inbound), `PAYOUT` (outbound), `ADJUSTMENT` (corrections)

### 💸 Employee Payout Flow (Atomic)

Employee payouts execute in a **single database transaction** to prevent partial state:

| Step | Action | Table Affected |
|------|--------|----------------|
| 1 | Debit org float by **gross** amount | `sacco_float_transactions` + `sacco_floats` |
| 2 | Credit employee wallet by **net** amount | `wallet_transactions` + `wallets` |
| ✔️ | If both succeed → commit | Both tables updated atomically |
| ❌ | If either fails → rollback | Neither table changed |

The difference between gross and net (statutory deductions: NSSF, SHA, Housing Levy, etc.) is retained by the organization.

Endpoint: `POST /api/v1/transactions/employee-payout`

**Idempotency:** Safe to retry — the same `idempotency_key` returns the original result.

---

## 🔐 Authentication & Authorization

- **JWT** with short-lived access tokens + long-lived refresh tokens
- **bcrypt** password hashing (cost factor 12)
- **Role-Based Access Control (RBAC)**:

| Role | Permissions |
|------|-------------|
| `SYSTEM_ADMIN` | Full access to all resources |
| `EMPLOYER` | Manage crew, vehicles, assignments within own organization |
| `EMPLOYEE` | View own profile, wallet, transactions (KYC-gated) |
| `LENDER` | View loan-related data |
| `INSURER` | View insurance-related data |

### KYC Enforcement

Employees with unverified KYC (`PENDING` or `REJECTED`) are restricted to `/profile` and `/notifications` only. All other routes are blocked by the frontend `kycGuard` until verification is completed. Admins can unverify employees via `PUT /crew/:id/kyc` with a `reason` field — the employee receives an IN_APP notification explaining why.

---

## 🛡️ Security

- Rate limiting: 100 requests/minute per IP (sliding window)
- SACCO-scoped data isolation for multi-tenant security
- Wallet ownership enforcement for CREW users
- Security headers: `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`
- CORS with exposed `Idempotency-Key`, `X-Request-Id`, `X-Response-Time`
- Request ID tracking on every request
- Structured JSON logging with `slog`

---

## 🧪 Testing

```bash
# Run all tests with race detector
go test ./... -race -count=1 -v

# Run specific test suites
go test ./internal/service/... -v           # Service layer
go test ./internal/handler/... -v           # HTTP handlers
go test ./internal/middleware/... -v        # Auth middleware
go test ./internal/external/sms/... -v     # SMS providers
go test ./internal/external/jambopay/... -v # JamboPay client
go test ./internal/external/perpay/... -v  # PerPay client
go test ./internal/external/iprs/... -v    # IPRS client
```

**425+ tests** across **20 test packages** covering:
- Auth flows (register, login, refresh, disabled accounts)
- Wallet operations (credit, debit, idempotency, insufficient balance)
- Earning calculations (FIXED, COMMISSION, HYBRID)
- Financial edge cases (large amounts, exact balance, 1-cent overdraw)
- Concurrent wallet access with race detector
- **Atomic transactions:** Employee payout validation (zero/negative/net>gross), wallet transfer validation (zero/negative/self-transfer)
- HTTP handler responses + RBAC enforcement
- **KYC lifecycle:** Verify, unverify (→PENDING with reason), reject (→REJECTED with reason), timestamp clearing, notification dispatch
- **Float verification:** Bank/card top-up pending workflow, confirm/reject handlers, verification mode branching
- JWT middleware (missing, invalid, expired tokens)
- **SMS**: Manager fallback chain, Optimize token caching, Africa's Talking bulk
- **JamboPay**: OAuth2 auth, M-Pesa/bank payouts, OTP verify, balance check
- **PerPay**: JWT auth, async submission (202), idempotency replay (409), status polling
- **IPRS**: OAuth2 scope=iprs, citizen verification, token caching

---

## ⚙️ Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `REDIS_URL` | ✅ | — | Redis connection string |
| `JWT_SECRET` | ✅ | — | JWT signing key (≥32 chars) |
| `MINIO_ENDPOINT` | ✅ | — | MinIO/S3 endpoint |
| `MINIO_ACCESS_KEY` | ✅ | — | MinIO access key |
| `MINIO_SECRET_KEY` | ✅ | — | MinIO secret key |
| `PORT` | | `8080` | HTTP server port |
| `ENVIRONMENT` | | `development` | `development`, `staging`, or `production` |
| `JWT_EXPIRY_MINUTES` | | `15` | Access token lifetime |
| `JWT_REFRESH_DAYS` | | `7` | Refresh token lifetime |
| `MINIO_BUCKET` | | `amy-mis` | Default bucket name |
| `RATE_LIMIT_RPM` | | `100` | Requests per minute per IP |

### SMS — Optimize (default)
| `SMS_CLIENT_ID` | | — | OAuth2 client ID |
| `SMS_CLIENT_SECRET` | | — | OAuth2 client secret |
| `SMS_TOKEN_URL` | | — | OAuth2 token endpoint |
| `SMS_URL` | | — | SMS send endpoint |
| `SMS_SENDER_ID` | | `AMY-MIS` | Sender name |

### SMS — Africa's Talking (fallback)
| `AT_API_KEY` | | — | API key |
| `AT_USERNAME` | | `sandbox` | Username |
| `AT_SHORTCODE` | | — | Short code |

### JamboPay (payment/payout)
| `JAMBOPAY_CLIENT_ID` | Prod | — | OAuth2 client ID |
| `JAMBOPAY_CLIENT_SECRET` | | — | OAuth2 client secret |
| `JAMBOPAY_BASE_URL` | | — | API base URL |

### PerPay (payroll)
| `PERPAY_CLIENT_ID` | | — | OAuth2 client ID |
| `PERPAY_CLIENT_SECRET` | | — | OAuth2 client secret |
| `PERPAY_BASE_URL` | | — | API base URL |

### IPRS (identity verification)
| `IPRS_CLIENT_ID` | | — | OAuth2 client ID |
| `IPRS_CLIENT_SECRET` | | — | OAuth2 client secret |
| `IPRS_BASE_URL` | | — | API base URL |
| `IPRS_TOKEN_ENDPOINT` | | — | OAuth2 token endpoint |

See [.env](.env) for a complete template.

---

## ⚡ Background Workers

| Job | Schedule | Description |
|-----|----------|-------------|
| `DailySummaryJob` | Every 24h | Aggregates daily earnings per crew member into `daily_earnings_summaries` |

Workers use a goroutine-based scheduler with graceful shutdown integration.

---

## 📊 Database

- **15 GORM models** across 7 migration sets (22 tables)
- Domains: Users, Crew, SACCOs, Vehicles, Routes, Assignments, Earnings, Wallets, Payroll, Documents, Notifications
- Automatic migration on startup via `golang-migrate`
- All repositories are transaction-aware via context-injected `TxManager`

---

## 🐳 Docker

```bash
# Development
docker compose up -d

# Production build
docker build -t amy-mis:latest .
```

---

## 📝 License

MIT License — see [LICENSE](LICENSE) for details.
