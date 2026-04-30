# AMY MIS — Documentation Hub

> Central documentation for the AMY MIS Workforce Financial Operating System.

---

## 📋 Documents

| Document | Description | Updated |
|----------|-------------|---------|
| [Product Requirements (PRD)](PRODUCT_REQUIREMENTS.txt) | Original product requirements document covering all modules, user personas, and MVP scope | — |
| [System Documentation](SYSTEM_DOCUMENTATION.md) | Complete backend architecture reference — tech stack, layered architecture, data model, auth, middleware, integrations, testing, and infrastructure | 2026-04-22 |
| [Gap Analysis](GAP_ANALYSIS.md) | Full entity-by-entity implementation matrix across all layers (Migration → Model → Repo → Service → Handler → Test) | 2026-04-23 |
| [Implementation Plan](IMPLEMENTATION_PLAN.md) | Angular PWA implementation tracker — 179 tasks across 16 phases with completion status | 2026-04-30 |
| [Test Accounts](TEST_ACCOUNTS.md) | All test user credentials, role access matrix, and supporting test data (SACCOs, vehicles, routes, crew) | 2026-04-30 |
| [Phase 5 Report](PHASE5_COMPLETION_REPORT.md) | Assignment Engine completion — detail page, filters, dropdown selectors, name resolution | 2026-04-30 |

---

## 🏗️ Architecture Quick Links

- **Backend**: `backend/` — Go 1.25, Gin, GORM, PostgreSQL
- **Frontend**: `frontend/` — Angular 20 PWA
- **USSD Gateway**: `USSD/` — Provider-agnostic strategy pattern
- **Database Seed**: `backend/cmd/seed/main.go` — Idempotent test data
- **API Docs**: `http://localhost:8080/swagger/index.html` (when backend is running)

---

## 🔑 Quick Start Testing

All accounts use password **`password123`**:

```
SYSTEM_ADMIN   → +254700000000
SACCO_ADMIN    → +254711111111
CREW (Driver)  → +254722000000
CREW (Cond.)   → +254722111111
LENDER         → +254733333333
INSURER        → +254744444444
```

See [Test Accounts](TEST_ACCOUNTS.md) for complete details and supporting data.

---

## 📊 Progress

| Phase | Status |
|-------|--------|
| 1. Foundation & Core | ✅ Complete |
| 2. Authentication | ✅ Complete |
| 3. Dashboard | ✅ Complete |
| 4. Crew Management | ✅ Complete |
| 5. Assignment Engine | ✅ Complete |
| 6–16. Remaining Phases | 🔄 In Progress (70 tasks remaining) |

**Overall: 61% complete** (109 / 179 tasks) — See [Implementation Plan](IMPLEMENTATION_PLAN.md) for full breakdown.
