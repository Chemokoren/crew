#!/usr/bin/env bash
# =============================================================================
# AMY MIS USSD Gateway — Local Test Startup Script
# =============================================================================
# Starts Redis (if not running), then launches the USSD gateway for local
# testing via Postman or curl.
#
# Usage:
#   chmod +x test_ussd.sh
#   ./test_ussd.sh             # Start USSD gateway
#   ./test_ussd.sh --stop      # Stop USSD Redis container
#   ./test_ussd.sh --simulate  # Start + run a quick smoke test
#
# USSD gateway will be available at: http://localhost:8090
#
# ── Endpoints ──
#   POST /ussd/webhook          → Production telco webhook (adapter auto-selected)
#   POST /ussd/simulator        → JSON simulator (for Postman)
#   GET  /health                → Health check
#   GET  /metrics               → Prometheus metrics
#
# ── Postman Examples ──
#
# 1) New session (Simulator/JSON):
#    POST http://localhost:8090/ussd/simulator
#    Body (JSON):
#    {
#      "session_id": "test-001",
#      "phone_number": "+254712345678",
#      "service_code": "*384*123#",
#      "input": ""
#    }
#
# 2) Select menu option (e.g., "1" for Check Balance):
#    POST http://localhost:8090/ussd/simulator
#    Body (JSON):
#    {
#      "session_id": "test-001",
#      "phone_number": "+254712345678",
#      "service_code": "*384*123#",
#      "input": "1"
#    }
#
# 3) Production webhook (Africa's Talking format):
#    POST http://localhost:8090/ussd/webhook
#    Body (x-www-form-urlencoded):
#      sessionId=ATSid_123
#      phoneNumber=+254712345678
#      serviceCode=*384*123#
#      text=
#
# ── IMPORTANT ──
# The USSD gateway calls the backend API at BACKEND_BASE_URL (default:
# http://localhost:8080). Start the backend first with:
#   cd ../backend && ./test_backend.sh
# =============================================================================

set -euo pipefail

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

log()   { echo -e "${GREEN}[✓]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
err()   { echo -e "${RED}[✗]${NC} $1"; }
info()  { echo -e "${CYAN}[i]${NC} $1"; }

# --- Stop command ---
if [[ "${1:-}" == "--stop" ]]; then
    info "Stopping USSD infrastructure..."
    docker compose down 2>/dev/null || true
    log "USSD infrastructure stopped."
    exit 0
fi

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     AMY MIS USSD Gateway — Local Test Setup  ║${NC}"
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
        warn "Created .env from .env.example"
    else
        err "No .env or .env.example found"
        exit 1
    fi
fi
log ".env file present"

# --- 3. Ensure Redis is available ---
info "Checking Redis availability..."

REDIS_AVAILABLE=false

# Check if Redis is already running locally (e.g., started by backend)
if redis-cli -p 6379 ping 2>/dev/null | grep -q PONG; then
    log "Redis already running on port 6379 — reusing"
    REDIS_AVAILABLE=true
    export REDIS_URL="redis://localhost:6379/1"
fi

# If not, try starting via Docker Compose
if [[ "$REDIS_AVAILABLE" == "false" ]]; then
    if command -v docker &>/dev/null; then
        info "Starting Redis via Docker Compose..."
        docker compose up -d redis 2>&1 | while IFS= read -r line; do
            echo "  $line"
        done

        MAX_RETRIES=20
        RETRY=0
        until redis-cli -p 6380 ping 2>/dev/null | grep -q PONG; do
            RETRY=$((RETRY + 1))
            if [[ $RETRY -ge $MAX_RETRIES ]]; then
                err "Redis failed to start after ${MAX_RETRIES}s"
                exit 1
            fi
            sleep 1
        done
        log "Redis started on port 6380"
        export REDIS_URL="redis://localhost:6380/1"
    else
        err "Redis is not running and Docker is not available."
        err "Please start Redis manually: redis-server"
        exit 1
    fi
fi

# --- 4. Check if backend is reachable ---
BACKEND_URL="${BACKEND_BASE_URL:-http://localhost:8080}"

info "Checking backend at ${BACKEND_URL}..."
if curl -sf "${BACKEND_URL}/health" &>/dev/null; then
    log "Backend is reachable at ${BACKEND_URL}"
else
    warn "Backend is NOT reachable at ${BACKEND_URL}"
    warn "USSD will start but backend-dependent features (balance, withdraw, etc.) will fail."
    warn "Start the backend first:  cd ../backend && ./test_backend.sh"
    echo ""
fi

# --- 5. Download Go dependencies ---
info "Downloading Go dependencies..."
go mod download 2>&1 | tail -5
log "Dependencies ready"

# --- 6. Run smoke test if --simulate ---
run_smoke_test() {
    echo ""
    echo -e "${CYAN}── Smoke Test ──${NC}"
    echo ""

    # Wait for server to be ready
    MAX_RETRIES=10
    RETRY=0
    until curl -sf http://localhost:8090/health &>/dev/null; do
        RETRY=$((RETRY + 1))
        if [[ $RETRY -ge $MAX_RETRIES ]]; then
            err "USSD gateway failed to start"
            exit 1
        fi
        sleep 1
    done

    echo -e "${BOLD}1) New session (initial dial):${NC}"
    curl -s -X POST http://localhost:8090/ussd/simulator \
        -H "Content-Type: application/json" \
        -d '{
            "session_id": "smoke-test-001",
            "phone_number": "+254712345678",
            "service_code": "*384*123#",
            "input": ""
        }' | python3 -m json.tool 2>/dev/null || true
    echo ""

    echo -e "${BOLD}2) Select option 6 (Register):${NC}"
    curl -s -X POST http://localhost:8090/ussd/simulator \
        -H "Content-Type: application/json" \
        -d '{
            "session_id": "smoke-test-001",
            "phone_number": "+254712345678",
            "service_code": "*384*123#",
            "input": "6"
        }' | python3 -m json.tool 2>/dev/null || true
    echo ""

    echo -e "${BOLD}3) Exit (option 0):${NC}"
    curl -s -X POST http://localhost:8090/ussd/simulator \
        -H "Content-Type: application/json" \
        -d '{
            "session_id": "smoke-test-002",
            "phone_number": "+254712345678",
            "service_code": "*384*123#",
            "input": ""
        }' | python3 -m json.tool 2>/dev/null || true
    echo ""

    curl -s -X POST http://localhost:8090/ussd/simulator \
        -H "Content-Type: application/json" \
        -d '{
            "session_id": "smoke-test-002",
            "phone_number": "+254712345678",
            "service_code": "*384*123#",
            "input": "0"
        }' | python3 -m json.tool 2>/dev/null || true
    echo ""

    echo -e "${BOLD}4) Production webhook (Africa's Talking format):${NC}"
    curl -s -X POST http://localhost:8090/ussd/webhook \
        -d "sessionId=ATSid_smoke_001" \
        -d "phoneNumber=+254712345678" \
        -d "serviceCode=*384*123#" \
        -d "text="
    echo ""
    echo ""

    log "Smoke test complete!"
}

# --- 7. Start the USSD gateway ---
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       Starting USSD Gateway (port 8090)      ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
info "USSD Gateway:   http://localhost:8090"
info "Health check:   http://localhost:8090/health"
info "Metrics:        http://localhost:8090/metrics"
echo ""
info "── Postman Quick Start ──"
echo ""
echo -e "  ${BOLD}POST${NC} http://localhost:8090/ussd/simulator"
echo -e "  ${BOLD}Content-Type:${NC} application/json"
echo -e "  ${BOLD}Body:${NC}"
echo '  {'
echo '    "session_id": "test-001",'
echo '    "phone_number": "+254712345678",'
echo '    "service_code": "*384*123#",'
echo '    "input": ""'
echo '  }'
echo ""
info "Press Ctrl+C to stop the USSD gateway"
echo ""
echo "─────────────────────────────────────────────────"

if [[ "${1:-}" == "--simulate" ]]; then
    # Start server in background, run smoke test, then bring to foreground
    go run cmd/server/main.go &
    SERVER_PID=$!
    trap "kill $SERVER_PID 2>/dev/null; exit 0" INT TERM

    run_smoke_test

    echo ""
    info "Server still running (PID: $SERVER_PID). Press Ctrl+C to stop."
    wait $SERVER_PID
else
    exec go run cmd/server/main.go
fi
