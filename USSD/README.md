# AMY MIS — USSD Gateway

A telecom-grade USSD gateway service for AMY MIS (CrewPay), designed for high-throughput, low-latency processing of millions of concurrent sessions.

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│  Telco Gateway   │────▶│  USSD Gateway │────▶│  AMY MIS     │
│  (Africa's      │◀────│  (this svc)  │◀────│  Backend API  │
│   Talking)      │     │              │     │              │
└─────────────────┘     └──────┬───────┘     └──────────────┘
                               │
                        ┌──────▼───────┐
                        │    Redis     │
                        │  (sessions)  │
                        └──────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Separate Go module | Independent scaling & deployment from backend |
| Redis sessions | Sub-millisecond lookup, TTL-based expiry |
| FSM engine | Deterministic menu flows, no hardcoded logic |
| Circuit breaker | Prevents cascading failures to backend |
| Per-MSISDN rate limiting | Protects against USSD bombing |
| Idempotency via request hashing | Handles telco retries safely |
| Gateway abstraction | Multi-provider support + simulator |

## USSD Menu Flow

```
*384*123#
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
├── 6. Register → Name → National ID → Role → Confirm → [END: Success]
├── 7. Language → [END: English/Kiswahili]
└── 0. Exit → [END: Goodbye]
```

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

### Simulator (JSON) Format

```bash
curl -X POST http://localhost:8090/ussd/simulator \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "test-001",
    "phone_number": "+254712345678",
    "service_code": "*384*123#",
    "input": ""
  }'
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
- [ ] Configure monitoring/alerting on Prometheus metrics
- [ ] Set up Redis Sentinel or Cluster for HA
- [ ] Enable TLS termination at load balancer
- [ ] Deploy behind Nginx/Envoy for connection pooling

## Project Structure

```
USSD/
├── cmd/server/main.go          # Entry point, DI wiring
├── internal/
│   ├── config/                 # Environment-based configuration
│   ├── session/                # Redis-backed session management
│   ├── engine/                 # FSM menu engine
│   ├── backend/                # Backend API client + circuit breaker
│   ├── gateway/                # Telco gateway adapters (AT, generic)
│   ├── handler/                # HTTP request handler
│   ├── middleware/             # Rate limiting, idempotency, sanitization
│   ├── metrics/                # Prometheus instrumentation
│   └── i18n/                   # Internationalization (EN, SW)
├── Dockerfile                  # Multi-stage production build
├── docker-compose.yml          # Local development stack
├── Makefile                    # Build, test, deploy targets
├── .env.example                # Configuration template
└── go.mod                      # Go module definition
```
