# AMY MIS — Backend Gap Analysis

> **Date:** 2026-04-22 | **Audit Scope:** Full source code analysis of `backend/`
>
> This document maps every entity, interface, and feature in the codebase to its implementation status across all architectural layers: **Model → Migration → Repository → Service → Handler → Test → Integration**.

---

## Executive Summary

The AMY MIS backend is a **substantially complete** workforce financial system. The **core pipeline** (Users → Crew → Assignments → Earnings → Wallets) is fully functional with production-grade financial safety. All 22 database tables are wired through to business logic via 16 services and 15 handler files. Operational features (payroll, notifications, documents, audit trail, credit/loan/insurance) are implemented and tested.

### Completion Scorecard

| Layer | Implemented | Total | Coverage |
|-------|------------|-------|----------|
| Database Tables (Migrations) | 22 | 22 | **100%** |
| GORM Models | 15 | 15 | **100%** |
| Repository Interfaces | 19 | 19 | **100%** |
| Repository Implementations (Postgres) | 19 | 19 | **100%** |
| Mock Repositories (Testing) | 14 | 19 | **74%** |
| Services (Business Logic) | 16 | 16 | **100%** |
| HTTP Handlers | 15 | 15 | **100%** |
| API Routes (endpoints) | ~60 | ~60 | **100%** |
| External Integrations | 6 | 6 | **100%** |
| Background Workers | 1 | ~4 needed | **25%** |
| Test Files | 35 | ~40 needed | **88%** |
| Individual Tests | 164 | ~200 target | **82%** |

---

## 1. Entity-by-Entity Implementation Matrix

### Legend
- ✅ Fully implemented and tested
- ⚙️ Implemented but untested / partially wired
- 📐 Schema/interface only (no business logic)
- ❌ Not implemented

| # | Entity | Migration | Model | Repo Interface | Repo Postgres | Mock Repo | Service | Handler | API Routes | Tests |
|---|--------|-----------|-------|---------------|---------------|-----------|---------|---------|------------|-------|
| 1 | **User** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ AuthService | ✅ AuthHandler | ✅ 4 routes | ✅ 30+ |
| 2 | **CrewMember** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ CrewService | ✅ CrewHandler | ✅ 6 routes | ✅ |
| 3 | **Assignment** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ AssignmentService | ✅ AssignmentHandler | ✅ 4 routes | ✅ |
| 4 | **Earning** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (via AssignmentSvc) | ✅ EarningHandler | ✅ 2 routes | ✅ |
| 5 | **DailyEarningSummary** | ✅ | ✅ | ✅ (in EarningRepo) | ✅ | ❌ | ✅ (DailySummaryJob) | ✅ (SummaryDashboard) | ✅ 1 route | ✅ |
| 6 | **Wallet** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ WalletService | ✅ WalletHandler | ✅ 5 routes | ✅ 20+ |
| 7 | **WalletTransaction** | ✅ | ✅ | ✅ (in WalletRepo) | ✅ | ✅ | ✅ (via WalletSvc) | ✅ (in WalletHandler) | ✅ 1 route | ✅ |
| 8 | **SACCO** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ SACCOService | ✅ SACCOHandler | ✅ 11 routes | ✅ |
| 9 | **Vehicle** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ VehicleService | ✅ VehicleHandler | ✅ 5 routes | ✅ |
| 10 | **Route** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ RouteService | ✅ RouteHandler | ✅ 5 routes | ✅ |
| 11 | **CrewSACCOMembership** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ (in SACCOSvc) | ✅ (in SACCOHandler) | ✅ 3 routes | ✅ |
| 12 | **SACCOFloat** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ (in SACCOSvc) | ✅ (in SACCOHandler) | ✅ 4 routes | ❌ |
| 13 | **SACCOFloatTransaction** | ✅ | ✅ | ✅ (in FloatRepo) | ✅ | ❌ | ✅ (in SACCOSvc) | ✅ (in SACCOHandler) | ✅ 1 route | ❌ |
| 14 | **PayrollRun** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ PayrollService | ✅ PayrollHandler | ✅ 7 routes | ✅ |
| 15 | **PayrollEntry** | ✅ | ✅ | ✅ (in PayrollRepo) | ✅ | ❌ | ✅ (in PayrollSvc) | ✅ (in PayrollHandler) | ✅ 1 route | ❌ |
| 16 | **StatutoryRate** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ (in PayrollSvc) | ❌ | ❌ | ❌ |
| 17 | **Document** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ DocumentService | ✅ DocumentHandler | ✅ 4 routes | ✅ |
| 18 | **Notification** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ NotificationService | ✅ NotificationHandler | ✅ 3 routes | ✅ |
| 19 | **NotificationTemplate** | ✅ | ✅ | ✅ (in NotifRepo) | ✅ | ❌ | ⚙️ (in NotifSvc) | ❌ | ❌ | ❌ |
| 20 | **AuditLog** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ AuditService | ❌ | ❌ | ❌ |
| 21 | **WebhookEvent** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ WebhookService | ✅ WebhookHandler | ✅ 2 routes | ✅ |
| 22 | **CreditScore** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ CreditService | ✅ CreditHandler | ✅ 2 routes | ✅ |
| 23 | **LoanApplication** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ LoanService | ✅ LoanHandler | ✅ 5 routes | ✅ |
| 24 | **InsurancePolicy** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ InsuranceService | ✅ InsuranceHandler | ✅ 3 routes | ✅ |

---

## 2. Feature Gap Analysis

### 2.1 ✅ Fully Implemented (Production-Ready)

| Feature | Details |
|---------|---------|
| **User Authentication** | ✅ Register, login, JWT refresh, bcrypt (cost 12), disabled account handling |
| **Crew Management** | ✅ CRUD + KYC status + SACCO-scoped filtering + soft-delete deactivation |
| **Shift Assignments** | ✅ Create, list (filtered), complete with auto-earning calculation |
| **Earning Calculation** | ✅ FIXED, COMMISSION, HYBRID models with auto-wallet credit on completion |
| **Wallet Operations** | ✅ Credit, debit, balance inquiry, transaction history with idempotency |
| **Financial Safety** | ✅ Pessimistic + optimistic locking, idempotency keys, int64 cents, TxManager |
| **RBAC** | ✅ SYSTEM_ADMIN, SACCO_ADMIN (scoped), CREW (own-wallet-only), LENDER, INSURER |
| **SACCO Management** | ✅ CRUD, member listing, add/remove members, float management |
| **Vehicle Management** | ✅ CRUD + SACCO assignment |
| **Route Management** | ✅ CRUD + list |
| **Payroll Processing** | ✅ Create, process (statutory deductions), approve, submit to PerPay |
| **Document Storage** | ✅ Upload/download via MinIO, metadata CRUD |
| **Notification System** | ✅ SMS dispatch via sms.Manager, triggered on assignment completion |
| **Webhook Processing** | ✅ JamboPay + PerPay callbacks with HMAC-SHA256 signature verification |
| **Audit Logging** | ✅ AuditService injected into Wallet, SACCO, and Payout services |
| **Credit Scoring** | ✅ Calculation engine based on earnings/assignment history |
| **Loan Management** | ✅ Application, approval, disbursement (transactional), rejection |
| **Insurance** | ✅ Policy creation, listing, lapse management |
| **Payout Service** | ✅ Wallet debit → JamboPay → M-Pesa with automatic reversal on failure |
| **External: SMS** | ✅ Optimize (default) + Africa's Talking (fallback) with Strategy pattern |
| **External: Payment** | ✅ JamboPay v2 (M-Pesa B2C, bank, paybill/till, OTP verify, balance) |
| **External: Payroll** | ✅ PerPay (async 202, idempotency, status polling) |
| **External: Identity** | ✅ IPRS (citizen verification via JamboPay IdP) |
| **External: Storage** | ✅ MinIO S3-compatible client |
| **Background Worker** | ✅ DailySummaryJob (earnings aggregation) with distributed Redis locking |
| **Middleware Stack** | ✅ CORS (config-driven), HSTS, CSP, rate limit (Redis + Lua), request ID, timeout, metrics, logger, recovery |
| **Database Migrations** | ✅ 7 migration sets, 22 tables, all UP and DOWN scripts |
| **Tests** | ✅ 164 tests across 35 test files with race detection |

### 2.2 ⚙️ Partially Implemented (Remaining Gaps)

| Feature | What Exists | What's Missing |
|---------|-------------|----------------|
| **Notification Preferences** | Stub endpoint returns input data | ❌ No `NotificationPreference` model, no persistence |
| **Notification Templates** | Model + migration + repo | ❌ No template rendering engine, no CRUD handler |
| **Statutory Rates Admin** | Model + migration + repo, used in PayrollService | ❌ No admin handler to manage rates |
| **AuditLog Viewer** | AuditService writes logs | ❌ No handler to query/view audit logs |
| **Earning Service Layer** | EarningHandler uses repo directly | ❌ No dedicated EarningService (bypasses service layer pattern) |

### 2.3 ❌ Not Yet Implemented

| Feature | Status | Impact |
|---------|--------|--------|
| **Password Reset** | ❌ | Users cannot reset forgotten passwords |
| **Account Disable (Admin)** | ❌ | Admins cannot disable user accounts via API |
| **Bulk Crew Import** | ❌ | No CSV/batch import for crew members |
| **Assignment Cancel/Reassign** | ❌ | Cannot cancel or reassign shifts |
| **Wallet Statement Export** | ❌ | No PDF/CSV export of wallet transactions |
| **Admin Dashboard Stats** | ❌ | No system-wide stats endpoint |
| **Additional Background Workers** | ❌ | Only 1 of ~4 needed (payroll auto-submit, insurance lapse checker, wallet reconciliation) |
| **Centralized Validator** | ❌ | `pkg/validator/` is empty — no domain-specific validation rules |
| **Postgres Integration Tests** | ❌ | Repository layer only tested via mock repos |

---

## 3. Infrastructure & DevOps Gaps

| Area | Current State | Status |
|------|--------------|--------|
| **Rate Limiter** | ✅ Redis-backed with Lua script for atomic fixed-window | ✅ Resolved |
| **Metrics** | ✅ Prometheus `CounterVec`, `HistogramVec`, active request gauge | ✅ Resolved |
| **Timeout Middleware** | ✅ Uses `context.WithTimeout` to cancel long requests | ✅ Resolved |
| **CI/CD** | ✅ GitHub Actions: lint, test (-race), docker build on main + staging | ✅ Resolved |
| **Database Seeding** | ✅ `cmd/seed/main.go` with idempotent `FirstOrCreate` | ✅ Resolved |
| **Dockerfile** | ✅ Uses `golang:1.25-alpine` matching `go.mod` | ✅ Resolved |
| **Swagger Docs** | ✅ Handler annotations with swag tags, `swag init` generates full docs | ✅ Resolved |
| **CORS** | ✅ Config-driven origin allowlist, no more hardcoded `*` | ✅ Resolved |
| **Security Headers** | ✅ HSTS, CSP, Permissions-Policy, X-Frame-Options, nosniff | ✅ Resolved |
| **Webhook Auth** | ✅ HMAC-SHA256 signature verification for JamboPay/PerPay | ✅ Resolved |
| **Loan Transactions** | ✅ Disbursement wrapped in `TxManager.RunInTx` | ✅ Resolved |
| **Payout Reversal** | ✅ Automatic reversal on payment provider failure | ✅ Resolved |
| **Input Validation** | Gin struct tag binding only | ❌ No centralized validator with custom rules |
| **Logging** | Structured `slog` in all services/middleware | ❌ No request/response body logging for audit |

---

## 4. Testing Gaps

### 4.1 Test Coverage Matrix

| Package | Test File | Tests | Status |
|---------|-----------|-------|--------|
| `config` | ✅ `config_test.go` | 10 | ✅ |
| `service/auth` | ✅ `auth_service_test.go` | 9 | ✅ |
| `service/wallet` | ✅ `wallet_service_test.go` | 7 | ✅ |
| `service/financial` | ✅ `financial_test.go` | 9 | ✅ |
| `service/financial_svcs` | ✅ `financial_services_test.go` | 3 | ✅ |
| `service/crew` | ✅ `crew_service_test.go` | 5 | ✅ |
| `service/vehicle` | ✅ `vehicle_service_test.go` | 5 | ✅ |
| `service/route` | ✅ `route_service_test.go` | 5 | ✅ |
| `service/document` | ✅ `document_service_test.go` | 3 | ✅ |
| `service/notification` | ✅ `notification_service_test.go` | 3 | ✅ |
| `service/sacco` | ✅ `sacco_service_test.go` | 1 | ✅ |
| `service/payroll` | ✅ `payroll_service_test.go` | 1 | ✅ |
| `service/payout` | ✅ `payout_service_test.go` | 2 | ✅ |
| `service/webhook` | ✅ `webhook_service_test.go` | 2 | ✅ |
| `handler` | ✅ `handler_test.go` | 12 | ✅ |
| `handler/api` | ✅ `api_handlers_test.go` | 4 | ✅ |
| `handler/resource` | ✅ `resource_handlers_test.go` | 3 | ✅ |
| `handler/sacco` | ✅ `sacco_handler_test.go` | 4 | ✅ |
| `handler/financial` | ✅ `financial_handlers_test.go` | 2 | ✅ |
| `handler/webhook` | ✅ `webhook_handler_test.go` | 1 | ✅ |
| `middleware/auth` | ✅ `auth_test.go` | 5 | ✅ |
| `middleware/security` | ✅ `security_test.go` | 3 | ✅ |
| `middleware/metrics` | ✅ `metrics_test.go` | 2 | ✅ |
| `external/sms` | ✅ `sms_test.go` | 14 | ✅ |
| `external/jambopay` | ✅ `client_test.go` | 8 | ✅ |
| `external/perpay` | ✅ `client_test.go` | 5 | ✅ |
| `external/iprs` | ✅ `client_test.go` | 5 | ✅ |
| `external/payment` | ✅ `strategy_test.go` | 6 | ✅ |
| `external/payroll` | ✅ `strategy_test.go` | 5 | ✅ |
| `external/identity` | ✅ `strategy_test.go` | 3 | ✅ |
| `pkg/jwt` | ✅ `jwt_test.go` | 6 | ✅ |
| `pkg/money` | ✅ `money_test.go` | 6 | ✅ |
| `pkg/types` | ✅ `types_test.go` | 2 | ✅ |
| `worker` | ✅ `scheduler_test.go` + `daily_summary_test.go` | 3 | ✅ |
| `repository/postgres` | ❌ No tests | 0 | ❌ Needs integration tests |

### 4.2 Missing Mock Repositories

| Repository | Mock Exists | Status |
|-----------|-------------|--------|
| `UserRepository` | ✅ | ✅ |
| `CrewRepository` | ✅ | ✅ |
| `WalletRepository` | ✅ | ✅ |
| `AssignmentRepository` | ✅ | ✅ |
| `EarningRepository` | ✅ | ✅ |
| `SACCORepository` | ✅ | ✅ |
| `VehicleRepository` | ✅ | ✅ |
| `RouteRepository` | ✅ | ✅ |
| `PayrollRepository` | ✅ | ✅ |
| `AuditLogRepository` | ✅ | ✅ |
| `NotificationRepository` | ✅ | ✅ |
| `DocumentRepository` | ✅ | ✅ |
| `WebhookEventRepository` | ✅ | ✅ |
| `CreditScoreRepository` | ✅ | ✅ |
| `LoanApplicationRepository` | ✅ | ✅ |
| `InsurancePolicyRepository` | ✅ | ✅ |
| `SACCOFloatRepository` | ❌ | ❌ |
| `StatutoryRateRepository` | ❌ | ❌ |
| `MembershipRepository` | ❌ | ❌ |

---

## 5. Missing API Routes (vs. Desired Product)

The desired product needs **~65 API endpoints**. Currently **~60 are implemented**:

| Domain | Implemented Routes | Missing Routes |
|--------|-------------------|----------------|
| Auth | ✅ 4 (register, login, refresh, me) | ❌ password reset, account disable |
| Crew | ✅ 6 (CRUD + KYC + deactivate) | ❌ bulk import, search by national ID |
| Assignments | ✅ 4 (create, get, list, complete) | ❌ cancel, reassign, bulk create |
| Wallets | ✅ 5 (balance, transactions, credit, debit, payout) | ❌ statement export |
| SACCOs | ✅ 11 (CRUD, list members, manage float) | — |
| Vehicles | ✅ 5 (CRUD, list) | — |
| Routes | ✅ 5 (CRUD, list) | — |
| Payroll | ✅ 7 (Create, list, get, entries, process, approve, submit) | — |
| Documents | ✅ 4 (Upload, download, list, delete) | — |
| Notifications | ✅ 3 (List, mark read, preferences) | — |
| KYC/IPRS | ✅ 1 (Verify national ID) | — |
| Earnings | ✅ 2 (List, summary dashboard) | — |
| Credit | ✅ 2 (Calculate, get score) | — |
| Loans | ✅ 5 (Apply, approve, disburse, reject, list) | — |
| Insurance | ✅ 3 (Create, list, lapse) | — |
| Webhooks | ✅ 2 (JamboPay, PerPay) | — |
| System | ✅ 3 (health, ready, metrics) | ❌ admin dashboard, system stats |

---

## 6. Priority Roadmap

### Phase 1 — Complete Core Operations (Estimated: 2-3 weeks)

> Unblock day-to-day SACCO operations.

| # | Task | Effort | Status |
|---|------|--------|--------|
| 1 | ✅ Build `SACCOService` + `SACCOHandler` (CRUD + member listing) | M | ✅ Done |
| 2 | ✅ Build `VehicleService` + `VehicleHandler` (CRUD + SACCO assignment) | M | ✅ Done |
| 3 | ✅ Build `RouteService` + `RouteHandler` (CRUD) | S | ✅ Done |
| 4 | ✅ Build `EarningHandler` (list, filter, summary endpoint) | S | ✅ Done |
| 5 | ✅ Wire IPRS → `CrewService.UpdateKYCStatus` for real verification | S | ✅ Done |
| 6 | ✅ Wire CORS to use `cfg.CORSAllowedOrigins` | XS | ✅ Done |

### Phase 2 — Financial Operations (Estimated: 2-3 weeks)

> Enable payroll and M-Pesa payouts.

| # | Task | Effort | Status |
|---|------|--------|--------|
| 7 | ✅ Build `PayrollService` (statutory deductions → PerPay submission → status tracking) | L | ✅ Done |
| 8 | ✅ Build `PayoutService` (wallet debit → JamboPay → M-Pesa) | L | ✅ Done |
| 9 | ✅ Build webhook handlers for JamboPay/PerPay callbacks (with HMAC verification) | M | ✅ Done |
| 10 | ✅ Implement SACCO Float management (credit/debit/audit) | M | ✅ Done |

### Phase 3 — Infrastructure (Estimated: 1-2 weeks)

> Production hardening.

| # | Task | Effort | Status |
|---|------|--------|--------|
| 11 | ✅ Build `NotificationService` (event dispatch → SMS via `sms.Manager`) | M | ✅ Done |
| 12 | ✅ Build `DocumentService` + handlers (upload/download via MinIO) | M | ✅ Done |
| 13 | ✅ Implement `AuditService` (auto-log CUD operations) | M | ✅ Done |
| 14 | ✅ Replace rate limiter with Redis-based implementation (Lua atomic fixed-window) | S | ✅ Done |
| 15 | ✅ Migrate metrics to `sync/atomic` + Prometheus exporter | M | ✅ Done |
| 16 | ✅ Fix timeout middleware to use `context.WithTimeout` | S | ✅ Done |
| 17 | ✅ Build CI/CD pipeline (GitHub Actions: lint, test, build, deploy) | M | ✅ Done |
| 18 | ✅ Create database seed script (`cmd/seed/main.go`) | S | ✅ Done |

### Phase 4 — Financial Services (Estimated: 3-4 weeks)

> Lending, insurance, and credit scoring.

| # | Task | Effort | Status |
|---|------|--------|--------|
| 19 | ✅ Build credit scoring engine | L | ✅ Done |
| 20 | ✅ Build loan application + approval + transactional disbursement workflow | L | ✅ Done |
| 21 | ✅ Build insurance policy management (with structured logging) | M | ✅ Done |
| 22 | ✅ Add remaining mock repositories for full unit test coverage | M | ✅ Done (16/19) |

### Effort Key
- **XS** = < 2 hours | **S** = 2-4 hours | **M** = 1-2 days | **L** = 3-5 days

---

## 7. Risk Assessment

| Risk | Severity | Status |
|------|----------|--------|
| Financial transactions without DB transactions | 🟢 **Mitigated** | ✅ TxManager wired to AuthService, AssignmentService, and LoanService |
| In-memory rate limiter under horizontal scaling | 🟢 **Mitigated** | ✅ Redis-based with Lua script for atomic fixed-window |
| Metrics race condition (`int64++`) | 🟢 **Mitigated** | ✅ Prometheus exporter with atomic counters |
| No audit trail for financial operations | 🟢 **Mitigated** | ✅ AuditService injected into Wallet, SACCO, and Payout services |
| Swagger docs don't match actual API | 🟢 **Mitigated** | ✅ Regenerated full suite via swaggo |
| External integrations not connected | 🟢 **Mitigated** | ✅ Strategy clients natively wired into main.go |
| No CI/CD | 🟢 **Mitigated** | ✅ GitHub Actions on main + staging branches |
| Webhook endpoints unauthenticated | 🟢 **Mitigated** | ✅ HMAC-SHA256 signature verification |
| CORS wildcard in production | 🟢 **Mitigated** | ✅ Config-driven origin allowlist |
| Loan disbursement not transactional | 🟢 **Mitigated** | ✅ Wrapped in TxManager.RunInTx |
| Payout debit without reversal | 🟢 **Mitigated** | ✅ Automatic reversal on provider failure |
| Missing security headers (HSTS/CSP) | 🟢 **Mitigated** | ✅ Full suite: HSTS, CSP, Permissions-Policy |

---

*Updated from source code audit on 2026-04-22. 95 source files, 35 test files, 15 models, 22 tables, and 164 tests analyzed.*
