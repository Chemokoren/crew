# AMY MIS — Test Accounts Reference

> **Last Updated:** 2026-04-30 | **Default Password:** `password123`

All test accounts are created by the database seed script at `backend/cmd/seed/main.go`.

---

## User Accounts

| Role | Phone | Email | Linked Entity |
|------|-------|-------|---------------|
| **SYSTEM_ADMIN** | `+254700000000` | admin@amy.com | Full platform access |
| **SACCO_ADMIN** | `+254711111111` | sacco_admin@amy.com | AMY SACCO LTD |
| **CREW** (Driver) | `+254722000000` | john.doe@amy.com | John Doe — CRW-0001 |
| **CREW** (Conductor) | `+254722111111` | jane.muthoni@amy.com | Jane Muthoni — CRW-0002 |
| **LENDER** | `+254733333333` | lender@amyfinance.com | — |
| **INSURER** | `+254744444444` | insurer@amycover.com | — |

---

## Role Access Matrix

| Module | SYSTEM_ADMIN | SACCO_ADMIN | CREW | LENDER | INSURER |
|--------|:----:|:----:|:----:|:----:|:----:|
| Dashboard | ✅ | ✅ | ✅ | ✅ | ✅ |
| Crew Management | ✅ | ✅ | ❌ | ❌ | ❌ |
| Assignments | ✅ | ✅ | ❌ | ❌ | ❌ |
| Earnings | ✅ | ✅ | ✅ | ✅ | ✅ |
| Wallets | ✅ | ✅ | ✅ | ✅ | ✅ |
| SACCOs | ✅ | ✅ | ❌ | ❌ | ❌ |
| Vehicles | ✅ | ✅ | ❌ | ❌ | ❌ |
| Routes | ✅ | ✅ | ❌ | ❌ | ❌ |
| Payroll | ✅ | ✅ | ❌ | ❌ | ❌ |
| Loans | ✅ | ✅ | ✅ | ✅ | ✅ |
| Insurance | ✅ | ✅ | ✅ | ✅ | ✅ |
| Notifications | ✅ | ✅ | ✅ | ✅ | ✅ |
| Admin Panel | ✅ | ❌ | ❌ | ❌ | ❌ |

---

## Supporting Test Data

### SACCOs
| Name | Registration | County |
|------|-------------|--------|
| AMY SACCO LTD | REG-AMY-1234 | Nairobi |
| CITY SHUTTLE SACCO | REG-CSH-5678 | Mombasa |

### Crew Members
| Crew ID | Name | Role | KYC Status | SACCO |
|---------|------|------|-----------|-------|
| CRW-0001 | John Doe | Driver | Verified | AMY SACCO LTD |
| CRW-0002 | Jane Muthoni | Conductor | Verified | AMY SACCO LTD |
| CRW-0003 | Peter Kamau | Rider | Pending | CITY SHUTTLE SACCO |

### Vehicles
| Registration | Type | Capacity | SACCO |
|-------------|------|----------|-------|
| KCX 123A | Matatu | 14 | AMY SACCO LTD |
| KDG 456B | Matatu | 33 | AMY SACCO LTD |
| KBZ 789C | Matatu | 14 | CITY SHUTTLE SACCO |

### Routes
| Name | From → To | Distance | Base Fare |
|------|-----------|----------|-----------|
| CBD - KILIMANI | CBD → Kilimani | 15.5 km | KES 100 |
| WESTLANDS - KAREN | Westlands → Karen | 22.0 km | KES 150 |

### Wallets (Initial Balances)
| Crew Member | Balance |
|-------------|---------|
| John Doe | KES 500.00 |
| Jane Muthoni | KES 325.00 |
| Peter Kamau | KES 120.00 |

---

## Running the Seed

```bash
cd backend
go run ./cmd/seed/
```

The seed is fully idempotent — it can be re-run at any time without creating duplicates.
