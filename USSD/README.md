# AMY MIS — USSD Gateway

A telecom-grade USSD gateway service for AMY MIS, designed for high-throughput, low-latency processing of millions of concurrent sessions across multiple industries.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Telco Gateway   │────▶│  USSD Gateway    │────▶│  AMY MIS     │
│  (Africa's      │◀────│  (this svc)      │◀────│  Backend API  │
│   Talking)      │     │                  │     │              │
└─────────────────┘     └────────┬─────────┘     └──────────────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
              ┌─────▼─────┐ ┌───▼───┐ ┌──────▼───────┐
              │   Redis    │ │ Role  │ │  Hardcoded   │
              │ (sessions) │ │ Cache │ │  Fallback    │
              └───────────┘ └───────┘ └──────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Separate Go module | Independent scaling & deployment from backend |
| Redis sessions | Sub-millisecond lookup, TTL-based expiry. Language preferences persist beyond session TTL. |
| FSM engine | Deterministic menu flows, no hardcoded logic |
| Circuit breaker | Prevents cascading failures to backend |
| Per-MSISDN rate limiting | Protects against USSD bombing |
| Idempotency via request hashing | Handles telco retries safely |
| Gateway abstraction | Multi-provider support + simulator |
| 3-layer role cache | Registration menus are never affected by backend latency |
| Service code routing | Industry-agnostic onboarding via shortcode mapping |

---

## Service Code Routing

Each USSD service code maps to an industry, determining which registration roles a user sees. A construction worker dialing `*384*200#` always sees Mason / Carpenter / Plumber — never Driver / Conductor / Rider.

### Two-Level Priority

| Level | Description | Override Behavior |
|-------|-------------|-------------------|
| **Primary** | Industry-specific roles tied to the service code | Always applies unless overridden |
| **Secondary** | Tenant-specific roles for a pinned organization | Overrides the industry defaults when configured |

### How It Works

```
SERVICE_CODE_ROUTES=*384*123#:TRANSPORT,*384*200#:CONSTRUCTION,*384*201#:CONSTRUCTION@<org-uuid>
                   ▲                     ▲                      ▲
                   │                     │                      │
           Transport industry     Construction industry   Tenant override
           (Driver, Conductor,    (Mason, Carpenter,      (fetches custom roles
            Rider, Booking Agent)  Plumber, Electrician,   from this org's config
                                   General Laborer)        via backend API)
```

### Configured Industries

| Service Code | Industry | Self-Registration Roles |
|---|---|---|
| `*384*123#` | TRANSPORT | Driver, Conductor, Boda Rider, Booking Agent, Dispatcher |
| `*384*200#` | CONSTRUCTION | Mason, Carpenter, Plumber, Electrician, General Laborer |
| `*384*300#` | HEALTH | Community Health Volunteer, Community Health Promoter, Nurse |
| `*384*400#` | LOGISTICS | Delivery Rider, Driver, Loader, Dispatcher |
| `*384*500#` | AGRICULTURE | Picker/Harvester, Field Worker, Sorter/Grader |
| `*384*600#` | HOSPITALITY | Waiter/Waitress, Cook, Housekeeper |

> **Note:** SUPERVISOR and SUPPORT roles (Foreman, Site Manager, Coordinator, Data Clerk, etc.) are excluded from self-registration. Those roles are assigned by administrators through the web dashboard only.

---

## 3-Layer Role Cache

Registration role data follows a cache-first architecture. The user's USSD experience is **never affected by backend API latency**.

```
User dials *384*200#                          Background Cron (midnight-aligned)
      │                                              │
      ▼                                              ▼
┌───────────────────┐                    ┌──────────────────────────┐
│ Layer 1: Redis    │◄───────────────────│ Fetch from Backend API   │
│ cache (sub-ms)    │    populates       │  ├─ @org_id? → tenant    │
│                   │                    │  └─ else → industry tmpl │
└────────┬──────────┘                    └──────────────────────────┘
         │ MISS                                      ▲
         ▼                                           │
┌───────────────────┐              ┌─────────────────┴────────────┐
│ Layer 2: Hardcoded│              │ Redis Pub/Sub (event-driven) │
│ (0ms, compiled    │              │ Channel: ussd:role_cache:    │
│  into binary)     │              │          invalidate          │
└───────────────────┘              └──────────────────────────────┘
```

| Layer | Source | Latency | Can Fail? | Used When |
|-------|--------|---------|-----------|-----------|
| **1** | Redis cache | <1ms | Yes → Layer 2 | Every registration request (hot path) |
| **2** | Hardcoded in `routing` package | 0ms | Never | Cache miss, Redis down, cold start |
| **Cron** | Backend API (midnight-aligned) | 50–200ms | Yes → skipped | Asynchronous refresh, never on hot path |
| **Pub/Sub** | Redis event → targeted refresh | <50ms | Yes → ignored | When backend publishes invalidation events |

### Cache Behavior

- **No TTL on cache entries** — they persist until the cron replaces them. Even if the backend is unreachable for days, stale-but-correct roles remain in Redis.
- **Initial population on startup** — the cron runs `RefreshAll()` immediately at boot (best-effort; failures fall through to Layer 2).
- **Midnight-aligned cron** — the first scheduled tick aligns to the next midnight boundary (local timezone), then repeats at the configured interval. This avoids unnecessary API calls during peak hours.
- **Periodic refresh** — configurable via `ROLE_CACHE_REFRESH_HOURS` (default: 24h).

### Event-Driven Invalidation (Redis Pub/Sub)

Instead of waiting for the next cron cycle, the backend can publish an invalidation event to trigger an immediate cache update:

```bash
# Refresh ALL service codes
redis-cli PUBLISH ussd:role_cache:invalidate "*"

# Refresh a SPECIFIC service code
redis-cli PUBLISH ussd:role_cache:invalidate "*384*200#"
```

| Payload | Effect |
|---------|--------|
| `*` | Full `RefreshAll()` — all service codes |
| `<service_code>` | Targeted refresh — only the specified code |

The Pub/Sub listener is optional (graceful degradation). If Pub/Sub is unavailable, the cron and admin endpoint remain operational.

### Admin Cache Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/admin/cache/refresh` | Trigger immediate full cache refresh via HTTP (returns 202) |

Also accessible from the frontend admin dashboard under the **USSD** tab.

---

## USSD Menu Flow

```
*384*<code>#
├── 1. Check Balance → [END: Show balance]
├── 2. Withdraw → Enter Amount → Confirm → PIN → [END: Success/Fail]
├── 3. Earnings
│   ├── 1. Today → [END: Daily summary]
│   ├── 2. This Week → [END: Weekly summary]
│   └── 3. This Month → [END: Monthly summary]
├── 4. Last Payment → [END: Transaction details]
├── 5. Loans
│   ├── 1. Check Status → [END: Loan details]
│   └── 2. Apply → Amount → Tenure → Confirm → [END: Applied]
├── 6. Register → Name → National ID → [Dynamic Role Menu] → PIN → Confirm → [END: Success]
├── 7. Language → [END: English/Kiswahili]
└── 0. Exit → [END: Goodbye]
```

> **[Dynamic Role Menu]** — The role selection shown at registration is determined by the dialed service code and the 3-layer cache. For `*384*200#`, the user sees Mason / Carpenter / Plumber etc.

## Quick Start

```bash
# 1. Copy environment config
cp .env.example .env

# 2. Start Redis (if not running)
docker compose up -d redis

# 3. Run the gateway
make run

# 4. Test with simulator
make simulate
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/ussd/africastalking` | Africa's Talking webhook |
| `POST` | `/ussd/simulator` | JSON-based simulator |
| `GET`  | `/health` | Health check |
| `GET`  | `/metrics` | Prometheus metrics |

### Africa's Talking Format

```bash
curl -X POST http://localhost:8090/ussd/africastalking \
  -d "sessionId=ATSid_123" \
  -d "phoneNumber=+254712345678" \
  -d "serviceCode=*384*123#" \
  -d "text="
```

### Simulator Format

Supports both JSON body and Query Parameters:

**JSON Body (Postman style):**
```bash
curl -X POST http://localhost:8090/ussd/simulator \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "test-001",
    "phone_number": "+254712345678",
    "service_code": "*384*200#",
    "input": ""
  }'
```

**Query Parameters (Browser/Curl style):**
```bash
curl -X POST "http://localhost:8090/ussd/simulator?session_id=test-001&phone_number=%2B254712345678&service_code=*384*200%23&input="
```

## Configuration

All configuration via environment variables. See [`.env.example`](.env.example) for full list.

### Critical Tunables

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND_TIMEOUT_MS` | 1500 | Backend call timeout (must be < 2000ms) |
| `SESSION_TTL_SECONDS` | 180 | Session lifetime in Redis |
| `CB_MAX_FAILURES` | 5 | Circuit breaker failure threshold |
| `RATE_LIMIT_PER_MSISDN` | 30 | Max requests per phone/minute |
| `REDIS_POOL_SIZE` | 100 | Redis connection pool size |
| `SERVICE_CODE_ROUTES` | `*384*123#:TRANSPORT` | Service code → industry mapping |
| `ROLE_CACHE_REFRESH_HOURS` | 24 | How often the cron refreshes role cache from API |

## Observability

### Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `ussd_requests_total` | Counter | Total requests by gateway/status |
| `ussd_request_duration_seconds` | Histogram | Processing latency |
| `ussd_sessions_active` | Gauge | Current active sessions |
| `ussd_sessions_created_total` | Counter | Total sessions created |
| `ussd_sessions_completed_total` | Counter | Successfully completed sessions |
| `ussd_menu_step_total` | Counter | Navigation per menu state |
| `ussd_menu_dropoff_total` | Counter | Drop-offs per state |
| `ussd_backend_calls_total` | Counter | Backend API calls |
| `ussd_backend_latency_seconds` | Histogram | Backend call latency |
| `ussd_circuit_breaker_state` | Gauge | Circuit breaker state |
| `ussd_rate_limited_total` | Counter | Rate-limited requests |
| `ussd_errors_total` | Counter | Errors by type |
| `ussd_role_cache_hits_total{service_code}` | Counter | Per-code cache hits (served from Redis) |
| `ussd_role_cache_misses_total{service_code}` | Counter | Per-code cache misses (used hardcoded fallback) |
| `ussd_role_cache_refresh_total{service_code,result}` | Counter | Per-code refresh success/error |
| `ussd_role_cache_refresh_duration_seconds` | Histogram | Full refresh cycle latency |
| `ussd_role_cache_last_refresh_timestamp` | Gauge | Unix time of last successful refresh |

### Admin Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/admin/cache/refresh` | Trigger immediate role cache refresh (returns 202, runs async) |

### Admin Dashboard (Frontend)

The **USSD** tab in System Administration provides a full management interface with three sub-sections:

#### Service Code Management

Ops can add, edit, disable, or delete service code → industry mappings without redeploying.

| Column | Description |
|--------|-------------|
| Service Code | The USSD shortcode (e.g. `*384*200#`) |
| Industry | Which industry template to apply |
| Tenant | Optional org-specific override |
| Roles | Self-registration roles for this code |
| Status | Active or Disabled toggle |

> **Note:** Currently uses in-memory seed data. When the backend REST APIs below are implemented, the UI will persist changes to the database.

#### MNO Provisioning

Submit and track shortcode provisioning requests to Kenyan MNOs:

| MNO | Status Lifecycle |
|-----|-----------------|
| Safaricom | PENDING → PROVISIONED → ACTIVE |
| Airtel | PENDING → PROVISIONED → ACTIVE |
| Telkom | PENDING → PROVISIONED → ACTIVE |

Includes summary cards for Active / Pending / Rejected counts.

#### A/B Testing

Create registration flow experiments to optimize conversion rates:

| Field | Description |
|-------|-------------|
| Variants A/B | Different role orderings or subsets |
| Traffic Split | Configurable 10-90% slider |
| Metrics | Live impressions and conversion rates per variant |
| Lifecycle | DRAFT → RUNNING → PAUSED → COMPLETED |

#### Planned Backend API Contracts

```
# Service Code Routes
GET    /api/v1/admin/ussd/routes          → list all routes
POST   /api/v1/admin/ussd/routes          → create route
PUT    /api/v1/admin/ussd/routes/:id      → update route
DELETE /api/v1/admin/ussd/routes/:id      → delete route

# MNO Provisioning
GET    /api/v1/admin/ussd/mno-requests    → list requests
POST   /api/v1/admin/ussd/mno-requests    → submit new request

# A/B Testing
GET    /api/v1/admin/ussd/ab-tests        → list experiments
POST   /api/v1/admin/ussd/ab-tests        → create experiment
PUT    /api/v1/admin/ussd/ab-tests/:id/status → start/pause/end
```

## Testing

```bash
# Unit tests
make test

# With coverage
make test-cover

# Benchmarks
make bench

# Load test (100 concurrent sessions)
make stress

# Drift check (compares USSD hardcoded roles with backend templates)
go test ./internal/routing/ -run TestHardcodedRoles_MatchBackendTemplates -v

# Registration integration tests
go test ./internal/engine/ -run TestRegistrationFlow -v
```

## Deployment

```bash
# Build Docker image
make docker-build

# Run with Docker Compose
make docker-run

# View logs
make docker-logs
```

### Production Checklist

- [ ] Set `ENVIRONMENT=production`
- [ ] Configure `BACKEND_API_KEY`
- [ ] Configure `AT_API_KEY` and `AT_SHORTCODE`
- [ ] Set `REDIS_POOL_SIZE` based on expected load
- [ ] Configure `SERVICE_CODE_ROUTES` with actual MNO-assigned shortcodes
- [ ] Set `ROLE_CACHE_REFRESH_HOURS` (24h default; lower for staging)
- [ ] Configure monitoring/alerting on Prometheus metrics
- [ ] Set up Redis Sentinel or Cluster for HA
- [ ] Enable TLS termination at load balancer
- [ ] Deploy behind Nginx/Envoy for connection pooling

---

## Trade-offs & Design Rationale

### 1. Hardcoded Roles vs. API-Only Roles

**Decision:** Roles are hardcoded in `routing.go` as the fallback layer, while the Redis cache (populated by the cron from the API) is the primary source.

| Approach | Pros | Cons |
|----------|------|------|
| **Hardcoded fallback** (current) | Zero-failure guarantee; no dependency on API or Redis for basic functionality | Requires a code deployment to update fallback roles |
| API-only | Always up-to-date | Single point of failure; API outage = broken registration |

**Trade-off accepted:** A code deployment is required to update the hardcoded fallback, but in practice the cron keeps the cache warm with fresh API data. The hardcoded fallback is only used during cold start before the first cron cycle completes, or when both Redis and the backend are simultaneously down.

### 2. Cache TTL Strategy — No Expiry

**Decision:** Cache entries have **no TTL**. They are overwritten only by the cron.

**Why:** If the cron fails (backend is unreachable), stale-but-correct data is better than no data. A Transport service code will always show transport roles, even if the last refresh was 3 days ago. This is preferable to expiring the cache and falling back to hardcoded (which is correct but may be outdated compared to what the backend has).

**Risk:** If a tenant's roles change in the backend (e.g., a new job type is added), it takes up to `ROLE_CACHE_REFRESH_HOURS` for the USSD to reflect the change.

### 3. Single Cron vs. Per-Request Refresh

**Decision:** A single background cron refreshes all service codes. Individual user requests never call the API for role data.

| Approach | Pros | Cons |
|----------|------|------|
| **Background cron** (current) | Predictable latency (sub-ms always); backend load is constant regardless of user traffic | Up to N hours staleness |
| Per-request (cache-aside) | Always fresh data | Latency spike on cache miss; thundering herd problem during cold start; backend load scales linearly with traffic |

**Trade-off accepted:** Up to 24h staleness for the guarantee of zero API latency on the hot path. For USSD, where the total session budget is ~2 seconds and includes telco round-trips, this is the correct choice.

### 4. USSD Hardcoded Roles ↔ Backend Template Drift

**Decision:** The USSD `routing.go` and the backend `industry_templates.go` each independently define job types.

**Risk:** These two files can drift apart. If a developer adds a new role in the backend template but forgets to update `routing.go`, the cron will fetch the updated roles (correct), but the hardcoded fallback will be stale (outdated but not wrong — it just won't show the new role).

**Mitigation:** ✅ `drift_test.go` is a CI check that parses both sources and fails the build if they diverge. This was already triggered once, catching a missing DISPATCHER role in the TRANSPORT industry.

### 5. Admin Dashboard — Frontend-First Approach

**Decision:** The admin dashboard (service code management, MNO provisioning, A/B testing) was built frontend-first with in-memory seed data. The UI is production-ready and defines the API contract; the backend REST endpoints are built when needed.

| Approach | Pros | Cons |
|----------|------|------|
| **Frontend-first** (current) | Fast iteration on UX; clear API contract for backend; no blocked dependency | Data is ephemeral until backend APIs exist |
| Backend-first | Data persistence from day 1 | Slower iteration; backend may build APIs the UI never uses |

**Trade-off accepted:** For operational tooling that admins use infrequently, getting the UX right first ensures the backend builds exactly what's needed. Seed data demonstrates all states (ACTIVE, PENDING, REJECTED, RUNNING, etc.) for review.

### 6. Event-Driven vs. Polling Invalidation

**Decision:** Redis Pub/Sub for event-driven cache invalidation runs alongside the midnight cron, not as a replacement.

| Trigger | When | Scope |
|---------|------|-------|
| Midnight cron | Automatic, every 24h | All service codes |
| Pub/Sub event | On-demand (backend publishes) | Single code or all |
| Admin endpoint | Manual HTTP POST | All service codes |

**Trade-off accepted:** Three invalidation paths increase complexity but provide defense in depth. If Pub/Sub is unavailable, the cron catches up. If the cron misses, the admin endpoint is a manual override.

---

## Areas of Improvement

### ✅ Completed (Short-Term)

1. ~~**Cache refresh Prometheus metrics**~~ — Per-service-code hit/miss counters, refresh duration histogram, last-refresh timestamp gauge.
2. ~~**On-demand cache invalidation endpoint**~~ — `POST /admin/cache/refresh` + frontend admin USSD tab.
3. ~~**Automated drift detection**~~ — CI test (`drift_test.go`) that compares USSD hardcoded roles with backend `industry_templates.go`.
4. ~~**Integration tests for registration flow**~~ — 8 tests covering all 6 industries, tenant overrides, cache miss fallback, and menu rendering.

### ✅ Completed (Medium-Term)

5. ~~**Event-driven cache invalidation (Redis Pub/Sub)**~~ — The USSD gateway subscribes to `ussd:role_cache:invalidate`. The backend can publish `*` (refresh all) or a specific service code for targeted updates.
6. ~~**Per-service-code cache metrics**~~ — Hit/miss/refresh counters now include `service_code` label for per-code observability.
7. ~~**Midnight-aligned cron**~~ — The first scheduled refresh aligns to the next midnight boundary. Subsequent refreshes follow the configured interval.

### Remaining

8. **Multi-region Redis replication** — For deployments spanning multiple data centers, ensure the role cache is replicated or each region runs its own cron instance.

### ✅ Completed (Frontend UI — Pending Backend API)

9. ~~**Admin dashboard for service code management**~~ — Full CRUD UI for service code → industry mappings with role editing, activation toggles, and tenant override support. Uses seed data until backend REST endpoints are built.
10. ~~**Telco-level shortcode provisioning**~~ — MNO provisioning request UI supporting Safaricom, Airtel, and Telkom. Status tracking (PENDING → PROVISIONED → ACTIVE) and summary cards. Backend MNO API integration pending.
11. ~~**A/B testing for registration flows**~~ — Experiment management UI with variant configuration, traffic split slider, live conversion rate tracking with progress bars, and start/pause/end controls. Backend experimentation engine pending.

---

## Project Structure

```
USSD/
├── cmd/server/main.go          # Entry point, DI wiring, cron + pub/sub startup
├── internal/
│   ├── config/                 # Environment-based configuration
│   ├── session/                # Redis-backed session management
│   ├── engine/                 # FSM menu engine
│   ├── backend/                # Backend API client + circuit breaker
│   ├── gateway/                # Telco gateway adapters (AT, generic)
│   ├── handler/                # HTTP request handler
│   ├── middleware/             # Rate limiting, idempotency, sanitization
│   ├── metrics/                # Prometheus instrumentation (incl. per-code cache metrics)
│   ├── i18n/                   # Internationalization (EN, SW)
│   ├── routing/                # Service code → industry mapping + hardcoded fallback roles
│   └── rolecache/              # 3-layer cache + midnight cron + Pub/Sub + Redis adapter
├── Dockerfile                  # Multi-stage production build
├── docker-compose.yml          # Local development stack
├── Makefile                    # Build, test, deploy targets
├── .env.example                # Configuration template
└── go.mod                      # Go module definition

frontend/src/app/features/admin/
├── admin-dashboard/            # Main admin component (tabs: Overview, Users, Audit, etc.)
└── ussd-management/            # USSD admin sub-component
    ├── ussd-management.component.ts    # Service codes, MNO, A/B test logic
    ├── ussd-management.component.html  # Sub-tab template with tables, modals, cards
    └── ussd-management.component.css   # Code badges, role chips, conversion bars
```
