# AMY MIS — Angular PWA Implementation Plan

> Cross-referenced against: `crew_spec.txt` (PRD), `crew_spec.txt` Phases 1-3, all 93 backend handler methods, and 72 API service methods in the frontend.

---

## Phase 1 — Foundation & Core Infrastructure

> **Goal**: Scaffold the Angular project, establish the design system, wire up authentication, and create the layout shell.

### 1.1 Project Scaffolding
| # | Task | Status |
|---|------|--------|
| 1 | Angular 20 project creation with SCSS + routing | ✅ Done |
| 2 | Environment files (`environment.ts`, `environment.prod.ts`) pointing to `localhost:8080/api/v1` | ✅ Done |
| 3 | `index.html` — SEO meta tags, Google Fonts (Inter, Outfit), Material Icons | ✅ Done |
| 4 | `main.ts` — Bootstrap with `AppComponent` | ✅ Done |

### 1.2 Design System (`styles.scss`)
| # | Task | Status |
|---|------|--------|
| 5 | CSS custom properties (colors, spacing, radius, shadows, z-index, transitions) | ✅ Done |
| 6 | Dark mode first — navy/indigo primary palette | ✅ Done |
| 7 | Glassmorphism card system (`.glass-card`) | ✅ Done |
| 8 | Button system (`.btn-primary`, `.btn-secondary`, `.btn-danger`, `.btn-ghost`, sizes) | ✅ Done |
| 9 | Form inputs (`.form-input`, `.form-select`, `.form-textarea`, `.form-error`, validation states) | ✅ Done |
| 10 | Badge/chip system (`.badge-success`, `.badge-warning`, `.badge-danger`, `.badge-info`, `.badge-accent`) | ✅ Done |
| 11 | Data table (`.data-table-wrapper`, `.data-table`) | ✅ Done |
| 12 | Modal/dialog (`.modal-backdrop`, `.modal-content`, header/body/footer) | ✅ Done |
| 13 | Toast notifications (`.toast-container`, `.toast-success/error/warning/info`) | ✅ Done |
| 14 | Skeleton loaders (`.skeleton` with shimmer animation) | ✅ Done |
| 15 | Stats card (`.stat-card` with hover glow & icon) | ✅ Done |
| 16 | Pagination (`.pagination`, `.page-btn`) | ✅ Done |
| 17 | Tab navigation (`.tab-nav`, `.tab-item`) | ✅ Done |
| 18 | Empty state (`.empty-state`) | ✅ Done |
| 19 | Search bar (`.search-input-wrapper`) | ✅ Done |
| 20 | Filters bar (`.filters-bar`) | ✅ Done |
| 21 | Custom scrollbar styling | ✅ Done |
| 22 | Responsive breakpoints (768px, 480px) | ✅ Done |
| 23 | Animations (fadeIn, slideUp, slideInLeft, pulse, slideInFromRight, skeleton-pulse) | ✅ Done |

### 1.3 Core TypeScript Layer
| # | Task | Status |
|---|------|--------|
| 24 | **Models** (`core/models/index.ts`) — All TypeScript interfaces mapping to backend DTOs (User, CrewMember, Assignment, Wallet, WalletTransaction, SACCO, Vehicle, Route, PayrollRun, PayrollEntry, Earning, DailySummary, CreditScore, LoanApplication, LoanTier, InsurancePolicy, Notification, AuditLog, SystemStats, SACCOFloat, SACCOMembership, enums) | ✅ Done |
| 25 | **AuthService** — JWT login/register/refresh, token storage, user profile signal, `hasRole()`, `isAdmin()` | ✅ Done |
| 26 | **ApiService** — Centralized HTTP client covering all 72+ endpoints across 14 domains | ✅ Done |
| 27 | **ToastService** — Signal-based toast notification queue with auto-dismiss | ✅ Done |
| 28 | **Auth Interceptor** — JWT Bearer injection, transparent 401 → token refresh | ✅ Done |
| 29 | **Error Interceptor** — Global HTTP error → user-friendly toast messages | ✅ Done |
| 30 | **Auth Guard** — Redirect unauthenticated users to `/auth/login` | ✅ Done |
| 31 | **Role Guard** — Factory function restricting routes by `SystemRole` | ✅ Done |

### 1.4 Shared Pipes
| # | Task | Status |
|---|------|--------|
| 32 | **CurrencyKesPipe** — Cents → `KES 1,234.56` formatting | ✅ Done |
| 33 | **RelativeTimePipe** — ISO date → `5m ago`, `2d ago`, or formatted date | ✅ Done |

### 1.5 Layout Shell
| # | Task | Status |
|---|------|--------|
| 34 | **AppComponent** — Conditional layout (auth pages vs. sidebar+topbar layout) | ✅ Done |
| 35 | **SidebarComponent** — Collapsible, role-filtered nav, section labels, user badge, mobile overlay | ✅ Done |
| 36 | **TopbarComponent** — User avatar, notification bell link, mobile menu toggle | ✅ Done |
| 37 | **ToastComponent** — Toast overlay renderer | ✅ Done |
| 38 | **App Routes** — All routes with lazy loading, auth guards, role guards | ✅ Done |
| 39 | **App Config** — HTTP interceptors registration, preload strategy | ✅ Done |
| 40 | Mobile sidebar: backdrop overlay, slide-in animation, auto-close on nav | ✅ Done |

---

## Phase 2 — Authentication

> **Goal**: Complete login/register flows, profile page, and password change.

| # | Task | Status |
|---|------|--------|
| 41 | **Login page** — Phone + password form, glassmorphism card, password toggle, loading state, error handling | ✅ Done |
| 42 | **Register page** — Role selector (Crew/SACCO Admin), conditional crew fields (National ID, Crew Role), all field validations | ✅ Done |
| 43 | Auto-redirect to `/dashboard` after successful login/register | ✅ Done |
| 44 | Bi-directional links between login ↔ register | ✅ Done |
| 45 | **Profile page** (`/profile`) — View current user info from `GET /auth/me`, edit email | ✅ Done |
| 46 | **Change password** page/modal — `POST /auth/change-password` with old/new password form | ✅ Done |
| 47 | Fetch user profile on app init (call `GET /auth/me` to refresh stale localStorage) | ✅ Done |
| 48 | Guest guard (redirect authenticated users away from auth pages) | ✅ Done |

---

## Phase 3 — Dashboard

> **Goal**: Role-aware dashboard with real-time stats, wallet overview, and actionable shortcuts.

| # | Task | Status |
|---|------|--------|
| 49 | **Admin dashboard** — System stats cards (crew, users, SACCOs, vehicles, assignments, wallet float) from `GET /admin/stats` | ✅ Done |
| 50 | **Crew member dashboard** — Wallet hero card with balance, total earned, total withdrawn | ✅ Done |
| 51 | Quick action cards with colored icons and descriptions (role-filtered) | ✅ Done |
| 52 | Time-based greeting ("Good morning/afternoon/evening") | ✅ Done |
| 53 | Live status indicator | ✅ Done |
| 54 | Today's earnings summary card (call `GET /earnings/summary/:crew_member_id` with today's date) | ✅ Done |
| 55 | Active assignments count for current user | ✅ Done |
| 56 | Recent transactions widget (last 5 wallet transactions) | ✅ Done |
| 57 | Earnings sparkline/mini-chart (last 7 days trend) | ✅ Done |

---

## Phase 4 — Crew Management (Identity & Registry)

> **Goal**: Full CRUD for crew members, KYC verification, bulk import, search by national ID.
> **PRD Reference**: §4.1 Identity & Registry — Unique crew ID, KYC (IPRS), Role management, Multi-SACCO association.

| # | Task | Status |
|---|------|--------|
| 58 | **Crew list page** — Data table with crew ID, name, role, KYC status, active status, joined date | ✅ Done |
| 59 | Search by name | ✅ Done |
| 60 | Filter by role (Driver/Conductor/Rider/Other) | ✅ Done |
| 61 | Filter by KYC status (Pending/Verified/Rejected) | ✅ Done |
| 62 | Pagination with page controls | ✅ Done |
| 63 | **Create crew modal** — National ID, First Name, Last Name, Role | ✅ Done |
| 64 | **Crew detail page** — Profile card with all fields | ✅ Done |
| 65 | KYC verification action (call `POST /crew/:id/verify`) | ✅ Done |
| 66 | KYC status update (`PUT /crew/:id/kyc`) | ✅ Done (via detail page) |
| 67 | Deactivate crew member (`DELETE /crew/:id`) | ✅ Done |
| 68 | **Search by National ID** — Call `GET /crew/search?national_id=...` | ✅ Done |
| 69 | **Bulk import modal** — CSV/form upload calling `POST /crew/bulk-import` | ✅ Done |
| 70 | Crew detail: link to wallet, assignments, and earnings for that crew member | ✅ Done |
| 71 | Filter by SACCO ID | ✅ Done |

---

## Phase 5 — Assignment & Operations Engine

> **Goal**: Create, complete, cancel, and reassign shift assignments with multi-model earnings.
> **PRD Reference**: §4.2 — Crew ↔ Vehicle ↔ SACCO ↔ Route mapping, Shift tracking, Multi-assignment per day.

| # | Task | Status |
|---|------|--------|
| 72 | **Assignment list** — Data table with shift date, status, earning model, actions | ✅ Done |
| 73 | Filter by status (Active/Completed/Cancelled) | ✅ Done |
| 74 | Filter by date | ✅ Done |
| 75 | **Create assignment modal** — Crew member, vehicle, SACCO, shift date/time, earning model (FIXED/COMMISSION/HYBRID), conditional amount/rate fields | ✅ Done |
| 76 | **Complete assignment** — Prompt for total revenue → `POST /assignments/:id/complete` | ✅ Done |
| 77 | **Cancel assignment** — Prompt for reason → `POST /assignments/:id/cancel` | ✅ Done |
| 78 | **Reassign assignment** — `POST /assignments/:id/reassign` with new crew member ID | ✅ Done (via API, prompt-based) |
| 79 | **Assignment detail page** — Full view with all fields, crew member name, vehicle reg, route | ✅ Done |
| 80 | Filter by crew member ID | ✅ Done |
| 81 | Filter by SACCO ID | ✅ Done |
| 82 | Show crew member name and vehicle reg instead of raw UUIDs in list | ✅ Done |
| 83 | Dropdown selectors for crew/vehicle/SACCO in create modal (fetched from API) | ✅ Done |

---

## Phase 6 — Wallet & Payments

> **Goal**: Balance view, transaction history, payout initiation, credit/debit (admin), CSV export.
> **PRD Reference**: §4.4 — Wallet ledger, SACCO float, Automated payouts via JamboPay, Transaction tracking.

| # | Task | Status |
|---|------|--------|
| 84 | **Wallet dashboard** — Balance card, total credited, total debited | ✅ Done |
| 85 | **Transaction history** — List with credit/debit icons, category, description, amount, relative time | ✅ Done |
| 86 | **CSV export** — Download wallet statement as CSV | ✅ Done |
| 87 | **Payout initiation** — Form calling `POST /wallets/:crew_member_id/payout` with channel (MOMO_B2C, BANK, MOMO_B2B), recipient details, idempotency key | ✅ Done |
| 88 | **Admin: Credit wallet** — Form for `POST /wallets/credit` with crew member ID, amount, category, description, idempotency key | ✅ Done |
| 89 | **Admin: Debit wallet** — Form for `POST /wallets/debit` | ✅ Done |
| 90 | Admin: Wallet lookup (search by crew member, view any wallet) | ✅ Done |
| 91 | Transaction filtering (by type, date range, category) | ✅ Done |
| 92 | Transaction pagination | ✅ Done |

---

## Phase 7 — SACCO Management

> **Goal**: Full SACCO CRUD, membership management, float operations.
> **PRD Reference**: §4.1, §4.4 — Multi-SACCO association, SACCO float management.

| # | Task | Status |
|---|------|--------|
| 93 | **SACCO list** — Name, reg no, county, contact, status | ✅ Done |
| 94 | Search SACCOs | ✅ Done |
| 95 | **Create SACCO modal** | ✅ Done |
| 96 | Delete SACCO | ✅ Done |
| 97 | **SACCO detail page** — Full info + member list + float overview | ✅ Done |
| 98 | **Update SACCO** — Edit form calling `PUT /saccos/:id` | ✅ Done |
| 99 | **Member management** — List members (`GET /saccos/:id/members`), add (`POST`), remove (`DELETE`) | ✅ Done |
| 100 | **Float overview** — Balance from `GET /saccos/:id/float` | ✅ Done |
| 101 | **Credit float** — `POST /saccos/:id/float/credit` | ✅ Done |
| 102 | **Debit float** — `POST /saccos/:id/float/debit` | ✅ Done |
| 103 | **Float transactions** — `GET /saccos/:id/float/transactions` | ✅ Done |

---

## Phase 8 — Vehicle & Route Management

> **Goal**: Complete fleet and route management CRUD.
> **PRD Reference**: §4.2 — Crew ↔ Vehicle ↔ SACCO ↔ Route mapping.

| # | Task | Status |
|---|------|--------|
| 104 | **Vehicle list** — Reg no, type, capacity, status | ✅ Done |
| 105 | **Create vehicle modal** | ✅ Done |
| 106 | Delete vehicle | ✅ Done |
| 107 | **Update vehicle** — Edit form calling `PUT /vehicles/:id` | ✅ Done |
| 108 | **Vehicle detail page** — Full info, assigned route, linked SACCO | ✅ Done |
| 109 | Filter by SACCO or vehicle type | ✅ Done |
| 110 | **Route list** — Code, name, origin → destination, status | ✅ Done |
| 111 | **Create route modal** | ✅ Done |
| 112 | Delete route | ✅ Done |
| 113 | **Update route** — Edit form calling `PUT /routes/:id` | ✅ Done |
| 114 | **Route detail page** — Full info, distance, assigned vehicles | ✅ Done |

---

## Phase 9 — Earnings Engine

> **Goal**: Earnings reports, daily summaries, per-crew/vehicle/SACCO breakdowns.
> **PRD Reference**: §4.3 — Earnings per assignment/shift, Daily aggregation across Vehicles/SACCOs.

| # | Task | Status |
|---|------|--------|
| 115 | **Earnings list** — Date, type, amount with date range filters | ✅ Done |
| 116 | **Earnings summary dashboard** — Call `GET /earnings/summary/:crew_member_id` | ❌ Not started |
| 117 | Filter by crew member ID (admin view) | ❌ Not started |
| 118 | Filter by assignment ID | ❌ Not started |
| 119 | Daily / weekly / monthly aggregation view | ❌ Not started |
| 120 | Earnings chart (bar chart showing daily earnings over time) | ❌ Not started |

---

## Phase 10 — Payroll & Compliance

> **Goal**: Full payroll lifecycle (Draft → Process → Approve → Submit), entry inspection, statutory deductions.
> **PRD Reference**: §4.5 — SHA, NSSF, Housing Levy deductions, Payroll processing via Perpay.

| # | Task | Status |
|---|------|--------|
| 121 | **Payroll list** — Period, status, gross/deductions/net, entry count, actions | ✅ Done |
| 122 | **Create payroll run modal** — SACCO ID, period start/end | ✅ Done |
| 123 | **Process payroll** — `POST /payroll/:id/process` | ✅ Done |
| 124 | **Approve payroll** — `POST /payroll/:id/approve` | ✅ Done |
| 125 | **Submit to PerPay** — `POST /payroll/:id/submit` | ✅ Done |
| 126 | **Payroll detail page** — Full run info + entries table | ❌ Not started |
| 127 | **Payroll entries table** — Per-crew breakdown (gross, SHA, NSSF, Housing Levy, net) from `GET /payroll/:id/entries` | ❌ Not started |
| 128 | Payroll status badge progression visualization | ❌ Not started |
| 129 | SACCO dropdown selector in create modal | ❌ Not started |
| 130 | **Statutory rates view** — `GET /admin/statutory-rates` | ❌ Not started |

---

## Phase 11 — Credit Scoring & Loans

> **Goal**: Credit score visualization, loan application, loan lifecycle management.
> **PRD Reference**: §4.6 — Credit scoring (earnings consistency), Loan lifecycle management.

| # | Task | Status |
|---|------|--------|
| 131 | **Loan list** — Category, amount, tenure, status, repaid, actions | ✅ Done |
| 132 | **Repay loan** — Prompt for amount → `POST /loans/:id/repay` | ✅ Done |
| 133 | **Credit score page** (`/credit`) — Score gauge/visualization from `GET /credit/:crew_member_id` | ❌ Not started |
| 134 | **Detailed score breakdown** — `GET /credit/:crew_member_id/detailed` | ❌ Not started |
| 135 | **Score history chart** — `GET /credit/:crew_member_id/history` | ❌ Not started |
| 136 | **Calculate score** — Trigger `POST /credit/:crew_member_id/calculate` | ❌ Not started |
| 137 | **Loan tier display** — `GET /loans/tier/:crew_member_id` showing max amount, rate, tenure | ❌ Not started |
| 138 | **Apply for loan modal** — Amount, tenure, category, purpose → `POST /loans` | ❌ Not started |
| 139 | **Admin: Approve loan** — `POST /loans/:id/approve` with approved amount and interest rate | ❌ Not started |
| 140 | **Admin: Reject loan** — `POST /loans/:id/reject` | ❌ Not started |
| 141 | **Admin: Disburse loan** — `POST /loans/:id/disburse` | ❌ Not started |
| 142 | Loan detail page with full lifecycle timeline | ❌ Not started |

---

## Phase 12 — Insurance

> **Goal**: Insurance policy management, creation (admin), and lapse tracking.
> **PRD Reference**: §4.6 — Insurance enrollment & premium deduction.

| # | Task | Status |
|---|------|--------|
| 143 | **Insurance list** — Provider, type, premium, frequency, period, status | ✅ Done |
| 144 | **Create insurance policy** (admin) — Form calling `POST /insurance` | ❌ Not started |
| 145 | **Lapse policy** (admin) — `POST /insurance/:id/lapse` | ❌ Not started |
| 146 | Filter by crew member, status, policy type | ❌ Not started |

---

## Phase 13 — Notifications & Preferences

> **Goal**: Notification center, read/unread management, notification preference settings.

| # | Task | Status |
|---|------|--------|
| 147 | **Notification list** — Title, body, channel, time, unread indicator | ✅ Done |
| 148 | **Mark as read** — Click to call `PUT /notifications/:id/read` | ✅ Done |
| 149 | **Notification preferences page** — `GET /notifications/preferences` + `PUT /notifications/preferences` | ❌ Not started |
| 150 | Unread count badge in sidebar & topbar | ❌ Not started |
| 151 | Filter by read/unread, channel | ❌ Not started |

---

## Phase 14 — System Administration

> **Goal**: System stats, user account management, audit logs, statutory rates, notification templates.

| # | Task | Status |
|---|------|--------|
| 152 | **Admin dashboard** — System stats cards from `GET /admin/stats` | ✅ Done |
| 153 | **Audit logs table** — Action, resource, time from `GET /admin/audit-logs` | ✅ Done |
| 154 | **Disable user account** — `POST /admin/users/:id/disable` | ❌ Not started |
| 155 | **Enable user account** — `POST /admin/users/:id/enable` | ❌ Not started |
| 156 | **Reset user password** — `POST /admin/users/:id/reset-password` | ❌ Not started |
| 157 | **Statutory rates view** — `GET /admin/statutory-rates` | ❌ Not started |
| 158 | **Notification templates** — List (`GET /admin/notifications/templates`), Create (`POST`), Update (`PUT`) | ❌ Not started |
| 159 | Audit log filtering (by resource, actor, date range) | ❌ Not started |
| 160 | Audit log pagination | ❌ Not started |
| 161 | User management table (list all users, search, enable/disable toggle) | ❌ Not started |

---

## Phase 15 — Documents

> **Goal**: Document upload/download management.

| # | Task | Status |
|---|------|--------|
| 162 | **Documents list** — `GET /documents` | ❌ Not started |
| 163 | **Upload document** — `POST /documents/upload` (multipart file upload) | ❌ Not started |
| 164 | **Download document** — `GET /documents/:id/download` | ❌ Not started |
| 165 | **Delete document** — `DELETE /documents/:id` | ❌ Not started |
| 166 | Add `documents` route, guard, and lazy-loaded component | ❌ Not started |
| 167 | Add `documents` API methods to ApiService | ❌ Not started |

---

## Phase 16 — UX Polish & Advanced Features

> **Goal**: Production-grade polish, better UX patterns, and missing micro-features.

| # | Task | Status |
|---|------|--------|
| 168 | Replace raw UUID inputs with **dropdown selectors** (crew, vehicle, SACCO, route) fetched from API | ❌ Not started |
| 169 | **Confirm dialog component** — Reusable modal replacing `window.confirm()` and `window.prompt()` | ❌ Not started |
| 170 | **Loading spinner component** — Reusable full-page or inline loading indicator | ❌ Not started |
| 171 | **Breadcrumb component** on detail pages | ❌ Not started |
| 172 | **Form validation** — Reactive forms with inline error messages (replace `FormsModule` with `ReactiveFormsModule` where needed) | ❌ Not started |
| 173 | **404 page** — Styled "not found" page for unknown routes | ❌ Not started |
| 174 | **Session expiry handling** — Redirect + toast when refresh token expires | ❌ Not started |
| 175 | **PWA manifest** + service worker for offline shell | ❌ Not started |
| 176 | Idempotency key generation helper (UUID v4 for wallet operations) | ❌ Not started |
| 177 | **Dark/light mode toggle** (currently dark-only) | ❌ Not started |
| 178 | Table sorting (click column header to sort) | ❌ Not started |
| 179 | Accessibility audit (ARIA labels, keyboard navigation, focus management) | ❌ Not started |

---

## Summary

| Phase | Description | Tasks | Done | Remaining |
|-------|-------------|-------|------|-----------|
| 1 | Foundation & Core Infrastructure | 40 | 40 | 0 |
| 2 | Authentication | 8 | **8** | **0** |
| 3 | Dashboard | 9 | **9** | **0** |
| 4 | Crew Management | 14 | **14** | **0** |
| 5 | Assignment Engine | 12 | **12** | **0** |
| 6 | Wallet & Payments | 9 | **9** | **0** |
| 7 | SACCO Management | 11 | **11** | **0** |
| 8 | Vehicle & Route | 11 | **11** | **0** |
| 9 | Earnings Engine | 6 | 1 | 5 |
| 10 | Payroll & Compliance | 10 | 5 | 5 |
| 11 | Credit & Loans | 12 | 2 | 10 |
| 12 | Insurance | 4 | 1 | 3 |
| 13 | Notifications | 5 | 2 | 3 |
| 14 | System Administration | 10 | 2 | 8 |
| 15 | Documents | 6 | 0 | 6 |
| 16 | UX Polish | 12 | 0 | 12 |
| **TOTAL** | | **179** | **127** | **52** |

> **Current progress: 71% complete** (127 / 179 tasks)
>
> **All backend API endpoints are covered in `ApiService`** (72 methods). The remaining work is building UI components that consume them.
