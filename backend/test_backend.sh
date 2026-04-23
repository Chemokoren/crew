#!/usr/bin/env bash
# =============================================================================
# AMY MIS Backend — Local Test Startup Script
# =============================================================================
# Detects locally-running infrastructure (PostgreSQL, Redis) and starts only
# what's missing via Docker Compose, then launches the backend Go server.
#
# Usage:
#   chmod +x test_backend.sh
#   ./test_backend.sh          # Start everything
#   ./test_backend.sh --stop   # Stop Docker containers started by this script
#
# Backend will be available at: http://localhost:8080
# Swagger docs at:              http://localhost:8080/swagger/index.html
# Health check:                  http://localhost:8080/health
# Metrics:                       http://localhost:8080/metrics
# =============================================================================

set -euo pipefail

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

log()   { echo -e "${GREEN}[✓]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
err()   { echo -e "${RED}[✗]${NC} $1"; }
info()  { echo -e "${CYAN}[i]${NC} $1"; }

# --- Stop command ---
if [[ "${1:-}" == "--stop" ]]; then
    info "Stopping backend Docker containers..."
    docker compose down 2>/dev/null || true
    log "Done."
    exit 0
fi

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║      AMY MIS Backend — Local Test Setup      ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════╝${NC}"
echo ""

# --- 1. Check prerequisites ---
info "Checking prerequisites..."

if ! command -v go &>/dev/null; then
    err "Go is not installed. Install Go 1.25+ from https://go.dev"
    exit 1
fi
log "Go $(go version | awk '{print $3}') found"

# --- 2. Ensure .env exists ---
if [[ ! -f .env ]]; then
    if [[ -f .env.example ]]; then
        cp .env.example .env
        warn "Created .env from .env.example — review and update values if needed"
    else
        err "No .env or .env.example found"
        exit 1
    fi
fi
log ".env file present"

# --- 3. Source .env to get configured values ---
set -a
source .env
set +a

# --- 4. Detect locally-running services ---
DOCKER_SERVICES_NEEDED=()

# -- PostgreSQL --
info "Checking PostgreSQL..."
if ss -tlnp 2>/dev/null | grep -q ':5432 ' || netstat -tlnp 2>/dev/null | grep -q ':5432 '; then
    log "PostgreSQL already running on port 5432 — using local instance"
else
    warn "PostgreSQL not detected on port 5432 — will start via Docker"
    DOCKER_SERVICES_NEEDED+=("postgres")
fi

# -- Redis --
info "Checking Redis..."
if redis-cli ping 2>/dev/null | grep -q PONG; then
    log "Redis already running on port 6379 — using local instance"
else
    warn "Redis not detected — will start via Docker"
    DOCKER_SERVICES_NEEDED+=("redis")
fi

# -- MinIO (optional — backend continues without it) --
info "Checking MinIO..."
if curl -sf http://localhost:9000/minio/health/live &>/dev/null; then
    log "MinIO already running on port 9000"
else
    info "MinIO not available — document upload/download will be disabled"
    info "To enable, start MinIO separately or run: docker compose up -d minio"
fi

# --- 5. Start only missing services via Docker ---
if [[ ${#DOCKER_SERVICES_NEEDED[@]} -gt 0 ]]; then
    if ! command -v docker &>/dev/null; then
        err "Docker is not installed but required for: ${DOCKER_SERVICES_NEEDED[*]}"
        err "Either install Docker or start these services manually."
        exit 1
    fi

    info "Starting missing services via Docker: ${DOCKER_SERVICES_NEEDED[*]}"
    docker compose up -d "${DOCKER_SERVICES_NEEDED[@]}" 2>&1 | while IFS= read -r line; do
        echo "  $line"
    done

    # Wait for Docker services to be healthy
    MAX_RETRIES=30

    for svc in "${DOCKER_SERVICES_NEEDED[@]}"; do
        RETRY=0
        case "$svc" in
            postgres)
                info "Waiting for Docker PostgreSQL..."
                until docker compose exec -T postgres pg_isready -U jp -d amymis &>/dev/null; do
                    RETRY=$((RETRY + 1))
                    if [[ $RETRY -ge $MAX_RETRIES ]]; then
                        err "PostgreSQL failed to start after ${MAX_RETRIES}s"
                        docker compose logs postgres --tail=10
                        exit 1
                    fi
                    sleep 1
                done
                log "Docker PostgreSQL is ready"
                ;;
            redis)
                info "Waiting for Docker Redis..."
                until docker compose exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; do
                    RETRY=$((RETRY + 1))
                    if [[ $RETRY -ge $MAX_RETRIES ]]; then
                        err "Redis failed to start after ${MAX_RETRIES}s"
                        exit 1
                    fi
                    sleep 1
                done
                log "Docker Redis is ready"
                ;;
            minio)
                info "Waiting for Docker MinIO..."
                until curl -sf http://localhost:9000/minio/health/live &>/dev/null; do
                    RETRY=$((RETRY + 1))
                    if [[ $RETRY -ge $MAX_RETRIES ]]; then
                        err "MinIO failed to start after ${MAX_RETRIES}s"
                        docker compose logs minio --tail=10
                        exit 1
                    fi
                    sleep 1
                done
                log "Docker MinIO is ready"
                ;;
        esac
    done
else
    log "All services already running locally — no Docker needed"
fi

# --- 6. Download Go dependencies ---
info "Downloading Go dependencies..."
go mod download 2>&1 | tail -5
log "Dependencies ready"

# --- 7. Start the backend ---
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║          Starting AMY MIS Backend             ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
info "Backend API:    http://localhost:${PORT:-8080}"
info "Swagger docs:   http://localhost:${PORT:-8080}/swagger/index.html"
info "Health check:   http://localhost:${PORT:-8080}/health"
info "Metrics:        http://localhost:${PORT:-8080}/metrics"
echo ""
info "Press Ctrl+C to stop the backend server"
echo ""
echo "─────────────────────────────────────────────────"

exec go run cmd/server/main.go
