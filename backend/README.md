# AMY MIS — Backend

A Workforce Financial Operating System for Informal Economies.

## Quick Start

### Prerequisites
- Go 1.24+
- PostgreSQL 16
- Redis 7
- MinIO (or any S3-compatible storage)

### Using Docker Compose (recommended)

```bash
# Start all services
docker compose up -d

# Check logs
docker compose logs -f app
```

### Manual Setup

```bash
# 1. Copy environment file
cp .env.example .env
# Edit .env with your actual values

# 2. Install dependencies
make deps

# 3. Run database migrations
make migrate-up

# 4. Start the server
make dev    # Hot reload (requires air)
# or
make run    # Direct run
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Liveness check |
| `GET /ready` | Readiness check (DB + Redis) |
| `GET /api/v1/...` | API v1 (requires auth) |

## Development Commands

```bash
make help           # Show all commands
make dev            # Run with hot reload
make test           # Run tests
make lint           # Run linter
make build          # Build binary
make migrate-up     # Run migrations
make migrate-create name=create_foo  # Create migration
make docker-up      # Start Docker services
```

## Project Structure

```
cmd/server/         — Entry point
internal/config/    — Configuration
internal/models/    — GORM data models
internal/repository/ — Data access layer
internal/service/   — Business logic
internal/handler/   — HTTP handlers + DTOs
internal/middleware/ — Auth, RBAC, logging
internal/worker/    — Background jobs (Asynq)
internal/external/  — External API clients
pkg/                — Shared utilities
migrations/         — SQL migration files
```

## Environment Variables

See [.env.example](.env.example) for all configuration options.
