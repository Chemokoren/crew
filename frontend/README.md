# AMY MIS — Frontend

Angular-based admin dashboard and crew management UI for the AMY MIS platform.

## Development

```bash
npm install
ng serve          # http://localhost:4200
```

## Key Features

### System Administration

The admin dashboard (`/admin`) provides platform-wide management:

| Tab | Features |
|-----|----------|
| **Overview** | User/crew/wallet stats |
| **Users** | Search, enable/disable, password reset |
| **Audit Logs** | Filterable activity history |
| **Templates** | Notification template CRUD |
| **Statutory Rates** | Tax/levy rate display |
| **USSD** | Gateway management (see below) |

### USSD Gateway Management

The USSD tab contains three sub-sections:

#### Service Code Management
- Add/edit/delete service code → industry mappings
- Per-code role configuration with tenant override support
- Activation toggles for operational control

#### MNO Provisioning
- Submit shortcode requests to Safaricom, Airtel, Telkom
- Track provisioning status (PENDING → PROVISIONED → ACTIVE)
- Summary cards for at-a-glance status

#### A/B Testing
- Create registration flow experiments with variant configuration
- Adjustable traffic split (10-90%) via slider
- Live conversion rate tracking with visual progress bars
- Experiment lifecycle: DRAFT → RUNNING → PAUSED → COMPLETED

#### Cache Management
- One-click role cache refresh via `POST /admin/cache/refresh`
- Success/error feedback with auto-dismissal

> **Note:** Service codes, MNO provisioning, and A/B testing currently use in-memory seed data. They will connect to backend REST APIs when those endpoints are implemented. See the [USSD README](../USSD/README.md) for planned API contracts.

## Proxy Configuration

Frontend requests are proxied to the appropriate backends during development:

| Path | Target | Service |
|------|--------|---------|
| `/api/` | `http://localhost:8080` | AMY MIS Backend |
| `/ussd-admin/` | `http://localhost:8090/admin/` | USSD Gateway |

Configuration: [`proxy.conf.json`](./proxy.conf.json)

## Project Structure

```
src/app/
├── core/
│   ├── models/index.ts           # All DTO interfaces (incl. ServiceCodeRoute, ABTest)
│   └── services/api.service.ts   # HTTP client (incl. refreshUSSDRoleCache)
├── features/
│   ├── admin/
│   │   ├── admin-dashboard/      # Main admin tabs
│   │   └── ussd-management/      # USSD sub-component (routes, MNO, A/B)
│   ├── auth/                     # Login, password flows
│   ├── crew/                     # Crew member management
│   └── ...
└── shared/                       # Pipes, directives, components
```

## Build

```bash
ng build                          # Production bundle → dist/amy-mis
ng test                           # Unit tests
```
