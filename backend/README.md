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
cmd/
  server/                — Entry point + dependency wiring (15-step startup)
  seed/                  — Database seeder (users, orgs, RBAC roles)
internal/
  config/                — Environment configuration + validation
  database/              — PostgreSQL (GORM) + Redis + TxManager
  handler/               — HTTP handlers + DTOs (request/response)
  handler/dto/           — Data Transfer Objects
  middleware/            — Auth (JWT), RBAC permission checker, CORS,
                           rate limiting, security headers, logging, recovery
  models/                — GORM data models + permission key constants
  rbac/                  — Permission registry + role templates (system & industry)
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
  types/                 — System roles + SystemRoleSlug() mapping
docs/                    — Swagger/OpenAPI generated docs
migrations/              — PostgreSQL migration files (26 sets, 30+ tables)
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
- **Dynamic RBAC** — all permissions are database-driven with zero hardcoded role logic

### System Roles

Every user has a `system_role` stored in the `users` table. Each system role maps to a **database-managed RBAC role** via `SystemRoleSlug()`. Permissions are resolved dynamically at runtime — never hardcoded.

| System Role | RBAC Slug | Default Permissions | Use Case |
|---|---|---|---|
| `SYSTEM_ADMIN` | `platform-super-admin` | All 141 permissions | Full platform access |
| `PLATFORM_ADMIN` | `platform-super-admin` | All 141 permissions | Platform management |
| `PLATFORM_AUDITOR` | `platform-auditor` | Read-only audit trails | Compliance oversight |
| `PLATFORM_SUPPORT` | `platform-support-agent` | User/worker/assignment view | Customer support |
| `PLATFORM_FINANCE` | `platform-finance` | Financial module access | Finance team |
| `EMPLOYER` | `system-employer` | 80 permissions (org ops) | SACCO/employer admin |
| `EMPLOYEE` | `system-employee` | 13 permissions (worker view) | Driver/conductor |
| `LENDER` | `system-lender` | 11 permissions (loans/credit) | Lending partner |
| `INSURER` | `system-insurer` | 7 permissions (insurance) | Insurance partner |

### How Permission Resolution Works

```
User logs in → JWT includes system_role
                    ↓
Middleware injects system_role into request context
                    ↓
RBACService.HasPermissionWithContext() checks:
  1. User's explicit RBAC role assignments (user_roles → role_permissions)
  2. System role's RBAC permissions (SystemRoleSlug → roles → role_permissions)
  3. Active RBAC policies (time/IP/attribute conditions)
                    ↓
Merged permission set → Allow or Deny
```

### Feature Gating (Staged Rollouts)

All system role permissions are editable through the **Roles & Permissions** UI. This enables feature gating without code changes:

1. Navigate to **Platform → Roles & Permissions**
2. Select a system role (e.g. **Employee**)
3. **Uncheck** permissions for features not yet ready (e.g. `loans.view`, `loans.apply`, `insurance.view`)
4. **Save** — affected users instantly lose access to those modules
5. When the feature is ready, **re-check** the permissions

> **Note:** System role templates are re-synced on server startup from the code templates in `internal/rbac/templates.go`. To make permission changes permanent, update the template. UI changes persist between restarts as long as `SyncSystemRoles()` doesn't overwrite them (it uses `ON CONFLICT ... DO UPDATE`).

### Permission Modules (141 total)

Permissions are organized into modules. Each permission key follows the pattern `module.action`:

| Module | Example Keys | Description |
|---|---|---|
| `workers` | `workers.view`, `workers.create`, `workers.delete`, `workers.verify_kyc` | Crew member management |
| `assignments` | `assignments.view`, `assignments.create`, `assignments.approve` | Shift/task management |
| `earnings` | `earnings.view`, `earnings.create`, `earnings.approve` | Earnings tracking |
| `wallet` | `wallet.view`, `wallet.fund_float`, `wallet.approve_payout` | Wallet & float |
| `payroll` | `payroll.view`, `payroll.run`, `payroll.approve` | Payroll processing |
| `loans` | `loans.view`, `loans.apply`, `loans.approve`, `loans.disburse` | Loan lifecycle |
| `insurance` | `insurance.view`, `insurance.enroll`, `insurance.cancel` | Insurance policies |
| `credit` | `credit.view`, `credit.score_compute` | Credit scoring |
| `roles` | `roles.view`, `roles.create`, `roles.assign`, `roles.manage_permissions` | RBAC management |
| `users` | `users.view`, `users.create`, `users.manage_roles` | User management |
| `documents` | `documents.view`, `documents.upload`, `documents.verify` | Document management |
| `organizations` | `organizations.view`, `organizations.create` | Org management |
| `vehicles` | `vehicles.view`, `vehicles.create`, `vehicles.delete` | Vehicle fleet |
| `routes` | `routes.view`, `routes.create`, `routes.delete` | Route management |
| `work_sites` | `work_sites.view`, `work_sites.create`, `work_sites.delete` | Work site management |
| `platform` | `platform.manage_roles`, `platform.manage_finance` | Platform-level ops |
| `audit` | `audit.view`, `audit.export` | Audit trail |
| `reports` | `reports.view`, `reports.export`, `reports.create_custom` | Reporting |
| `settings` | `settings.view`, `settings.update`, `settings.manage_tenant` | Tenant settings |
| `compliance` | `compliance.view`, `compliance.generate_reports` | Statutory compliance |
| `notifications` | `notifications.view`, `notifications.send` | Notifications |

### RBAC API Endpoints

| Method | Path | Guard | Description |
|---|---|---|---|
| `GET` | `/api/v1/rbac/roles` | `roles.view` | List all roles |
| `POST` | `/api/v1/rbac/roles` | `roles.create` | Create a custom role |
| `GET` | `/api/v1/rbac/roles/:id` | `roles.view` | Get role details |
| `PUT` | `/api/v1/rbac/roles/:id` | `roles.update` | Update role name/desc |
| `DELETE` | `/api/v1/rbac/roles/:id` | `roles.delete` | Soft-delete a role |
| `POST` | `/api/v1/rbac/roles/:id/clone` | `roles.create` | Clone a role |
| `GET` | `/api/v1/rbac/roles/:id/permissions` | `roles.view` | List role's permissions |
| `PUT` | `/api/v1/rbac/roles/:id/permissions` | `roles.manage_permissions` | Set role permissions (bulk) |
| `POST` | `/api/v1/rbac/roles/compare` | `roles.view` | Compare two roles |
| `GET` | `/api/v1/rbac/permissions` | `roles.view` | List all permissions |
| `GET` | `/api/v1/rbac/permissions/modules` | `roles.view` | List permission modules |
| `GET` | `/api/v1/rbac/users/:id/roles` | `roles.assign` | Get user's roles |
| `POST` | `/api/v1/rbac/users/:id/roles` | `roles.assign` | Assign role to user |
| `DELETE` | `/api/v1/rbac/users/:id/roles/:roleId` | `roles.assign` | Revoke role from user |
| `GET` | `/api/v1/rbac/users/:id/permissions` | Self-service | Get user's effective permissions |
| `GET` | `/api/v1/rbac/templates` | `roles.view` | List role templates |
| `POST` | `/api/v1/rbac/templates/:id/apply` | `roles.apply_templates` | Apply template to tenant |
| `GET` | `/api/v1/rbac/policies` | `roles.manage_permissions` | List RBAC policies |
| `POST` | `/api/v1/rbac/policies` | `roles.manage_permissions` | Create RBAC policy |
| `GET` | `/api/v1/rbac/matrix` | `roles.view` | Full permission matrix |

### KYC Enforcement

Employees with unverified KYC (`PENDING` or `REJECTED`) are restricted to `/profile` and `/notifications` only. All other routes are blocked by the frontend `kycGuard` until verification is completed. Admins can unverify employees via `PUT /crew/:id/kyc` with a `reason` field — the employee receives an IN_APP notification explaining why.

---

## 🌱 Database Seeding

The seeder creates test users, organizations, vehicles, routes, crew members, wallets, and assignments. It also syncs all RBAC permissions, role templates, and system roles.

```bash
cd backend && go run cmd/seed/main.go
```

### Test Accounts

| Role | Phone | Password | Description |
|---|---|---|---|
| **SYSTEM_ADMIN** | +254700000000 | `masai123` | Full platform access |
| **SACCO_ADMIN** (EMPLOYER) | +254711111111 | `masai123` | Operations management |
| **CREW** (Driver) | +254722000000 | `masai123` | Worker view |
| **CREW** (Conductor) | +254722111111 | `masai123` | Worker view |
| **LENDER** | +254733333333 | `masai123` | Financial services partner |
| **INSURER** | +254744444444 | `masai123` | Insurance partner |

> The seeder is **idempotent** — safe to run multiple times. It uses `FirstOrCreate` for entities and force-updates all passwords at the end.

---

## 🛡️ Security

- **Dynamic RBAC**: All authorization flows through the database — no hardcoded role-to-permission logic
- **Privilege escalation prevention**: Users cannot grant permissions they don't possess
- **Tenant isolation**: Non-platform users can only access data within their organization
- **Self-service permissions**: Users can fetch their own permissions without prior RBAC grants (avoids chicken-and-egg)
- **Permission caching**: Redis-backed with distributed invalidation
- Rate limiting: 100 requests/minute per IP (sliding window)
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

- **20+ GORM models** across 26 migration sets (30+ tables)
- Domains: Users, Crew, SACCOs, Vehicles, Routes, Assignments, Earnings, Wallets, Payroll, Documents, Notifications, RBAC (permissions, roles, role_permissions, user_roles, policies, role_templates)
- Automatic migration on startup via `golang-migrate`
- All repositories are transaction-aware via context-injected `TxManager`
- RBAC tables seeded on startup: permissions registry, system role templates, industry templates

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
