# AMY MIS — Backend API

> **A Workforce Financial Operating System for Kenya's Informal Transport Sector**

AMY MIS digitizes the financial lifecycle of matatu (minibus), boda-boda (motorcycle), and tuk-tuk crews — from shift assignments and earnings calculation to wallet management, payroll processing, and SACCO operations.

---

## 🚀 Quick Start

### Prerequisites

- **Go** 1.21+
- **PostgreSQL** 15+
- **Redis** 7+
- **MinIO** (or S3-compatible storage)
- **Docker & Docker Compose** (recommended)

### 1. Clone & Configure

```bash
git clone https://github.com/Chemokoren/crew.git
cd crew/backend
cp .env.example .env
# Edit .env with your database credentials
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
| `PUT` | `/api/v1/crew/:id/kyc` | Update KYC status |
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
| `POST` | `/api/v1/wallets/credit` | Credit wallet (System Admin) |
| `POST` | `/api/v1/wallets/debit` | Debit wallet (System Admin) |

### System

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/ready` | Readiness probe |
| `GET` | `/metrics` | Request metrics (JSON) |
| `GET` | `/swagger/*` | Swagger API docs |

---

## 🏗️ Architecture

```
cmd/server/            — Entry point + dependency wiring
internal/
  config/              — Environment configuration + validation
  database/            — PostgreSQL (GORM) + Redis connections
  handler/             — HTTP handlers + DTOs (request/response)
  handler/dto/         — Data Transfer Objects
  middleware/          — Auth (JWT), RBAC, CORS, rate limiting,
                         metrics, security headers, logging, recovery
  models/              — GORM data models (15 entities)
  repository/          — Data access interfaces
  repository/postgres/ — GORM repository implementations
  repository/mock/     — In-memory mocks for testing
  service/             — Business logic layer
  worker/              — Background jobs (Asynq)
  external/            — External API clients (MinIO, etc.)
pkg/
  errs/                — Shared domain errors
  jwt/                 — JWT token manager
  money/               — Cents ↔ KES conversion utilities
  pagination/          — Pagination helpers
  types/               — System roles + shared types
docs/                  — Swagger/OpenAPI generated docs
migrations/            — PostgreSQL migration files (7 sets)
```

### Design Principles

- **Clean Architecture**: Handlers → Services → Repositories (all via interfaces)
- **Financial Safety**: All money stored as `int64` cents — no floats in the pipeline
- **Wallet Concurrency**: `SELECT ... FOR UPDATE` + optimistic version checks
- **Idempotency**: Financial endpoints require `Idempotency-Key` header
- **No Raw Models in API**: DTOs strip internal fields at the boundary

---

## 💰 Earning Models

| Model | Formula | Use Case |
|-------|---------|----------|
| `FIXED` | `fixed_amount_cents` | Daily flat rate |
| `COMMISSION` | `revenue × commission_rate` | Percentage of collections |
| `HYBRID` | `base_cents + (revenue × commission_rate)` | Base pay + commission |

---

## 🔐 Authentication & Authorization

- **JWT** with short-lived access tokens + long-lived refresh tokens
- **bcrypt** password hashing (cost factor 10)
- **Role-Based Access Control (RBAC)**:

| Role | Permissions |
|------|-------------|
| `SYSTEM_ADMIN` | Full access to all resources |
| `SACCO_ADMIN` | Manage crew, vehicles, assignments within SACCO |
| `CREW` | View own profile, wallet, transactions |
| `LENDER` | View loan-related data |
| `INSURER` | View insurance-related data |

---

## 🛡️ Security

- Rate limiting: 100 requests/minute per IP (sliding window)
- Security headers: `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`
- CORS with exposed `Idempotency-Key`, `X-Request-Id`, `X-Response-Time`
- Request ID tracking on every request
- Structured JSON logging with `slog`

---

## 🧪 Testing

```bash
# Run all tests with race detector
go test ./... -race -count=1 -v

# Run specific test suite
go test ./internal/service/... -v    # Service layer
go test ./internal/handler/... -v    # HTTP handlers
go test ./internal/middleware/... -v  # Auth middleware
```

**104 tests** covering:
- Auth flows (register, login, refresh, disabled accounts)
- Wallet operations (credit, debit, idempotency, insufficient balance)
- Earning calculations (FIXED, COMMISSION, HYBRID)
- Financial edge cases (large amounts, exact balance, 1-cent overdraw)
- Concurrent wallet access with race detector
- HTTP handler responses + RBAC enforcement
- JWT middleware (missing, invalid, expired tokens)

---

## ⚙️ Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `REDIS_URL` | ✅ | — | Redis connection string |
| `JWT_SECRET` | ✅ | — | JWT signing key (≥32 chars) |
| `JWT_EXPIRY_MINUTES` | | `15` | Access token lifetime |
| `JWT_REFRESH_DAYS` | | `7` | Refresh token lifetime |
| `MINIO_ENDPOINT` | ✅ | — | MinIO/S3 endpoint |
| `MINIO_ACCESS_KEY` | ✅ | — | MinIO access key |
| `MINIO_SECRET_KEY` | ✅ | — | MinIO secret key |
| `MINIO_BUCKET` | ✅ | — | Default bucket name |
| `PORT` | | `8080` | HTTP server port |
| `ENVIRONMENT` | | `development` | `development` or `production` |
| `MIGRATIONS_PATH` | | `./migrations` | SQL migrations directory |

See [.env.example](.env.example) for a complete template.

---

## 📊 Database

- **15 GORM models** across 7 migration sets
- Domains: Users, Crew, SACCOs, Vehicles, Routes, Assignments, Earnings, Wallets, Payroll, Documents, Notifications
- Automatic migration on startup via `golang-migrate`

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
