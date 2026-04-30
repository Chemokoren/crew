# AMY MIS — Workforce Financial Operating System

AMY MIS digitizes workforce operations, earnings, wallets, payroll, and SACCO management for Africa's informal transport sector.

## Project Structure

```
crew/
├── backend/        Go API server (Gin + GORM + PostgreSQL)
├── frontend/       Angular 20 PWA dashboard
├── USSD/           USSD gateway service
└── docs/           📚 All project documentation
```

## 📚 Documentation

All project documentation is centralized in the [`docs/`](docs/) directory:

- **[Documentation Hub](docs/README.md)** — Start here
- [Product Requirements (PRD)](docs/PRODUCT_REQUIREMENTS.txt)
- [System Documentation](docs/SYSTEM_DOCUMENTATION.md)
- [Gap Analysis](docs/GAP_ANALYSIS.md)
- [Implementation Plan](docs/IMPLEMENTATION_PLAN.md)
- [Test Accounts](docs/TEST_ACCOUNTS.md)
- [Phase 5 Report](docs/PHASE5_COMPLETION_REPORT.md)

## Quick Start

```bash
# Backend
cd backend
./test_backend.sh          # Start PostgreSQL, Redis, MinIO + API server
go run ./cmd/seed/         # Seed test data

# Frontend
cd frontend
npm install
npm run start              # http://localhost:4200
```

**Test login:** `+254700000000` / `password123` (System Admin) — See [Test Accounts](docs/TEST_ACCOUNTS.md) for all roles.
