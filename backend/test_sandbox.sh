#!/usr/bin/env bash
set -euo pipefail

# ╔══════════════════════════════════════════════╗
# ║   AMY MIS Sandbox Financial Service          ║
# ║   Mirrors JamboPay API for dev/test          ║
# ╚══════════════════════════════════════════════╝

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║   AMY MIS Sandbox — Financial Test Service   ║"
echo "╚══════════════════════════════════════════════╝"
echo ""

# Check Go
if ! command -v go &>/dev/null; then
    echo "[✗] Go is not installed"
    exit 1
fi
echo "[✓] Go $(go version | awk '{print $3}') found"

# Load .env if present
if [ -f .env ]; then
    echo "[✓] .env file present"
fi

# Download dependencies
echo "[i] Downloading Go dependencies..."
go mod download 2>/dev/null
echo "[✓] Dependencies ready"

PORT="${SANDBOX_PORT:-8091}"

echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║     Starting Sandbox Service (port $PORT)     ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
echo "[i] Sandbox API:    http://localhost:$PORT"
echo "[i] Health check:   http://localhost:$PORT/health"
echo "[i] Admin stats:    http://localhost:$PORT/sandbox/admin/stats"
echo ""
echo "[i] ── JamboPay-Compatible Endpoints ──"
echo ""
echo "  POST http://localhost:$PORT/auth/token"
echo "  POST http://localhost:$PORT/payout"
echo "  POST http://localhost:$PORT/payout/authorize"
echo "  GET  http://localhost:$PORT/wallet/account?accountNo=..."
echo ""
echo "[i] ── Test Setup ──"
echo ""
echo "  Set in backend .env:"
echo "    JAMBOPAY_BASE_URL=http://localhost:$PORT"
echo "    JAMBOPAY_CLIENT_ID=sandbox-client"
echo "    JAMBOPAY_CLIENT_SECRET=sandbox-secret"
echo "    PAYMENT_JAMBOPAY_ENABLED=true"
echo ""
echo "[i] ── Seed Data ──"
echo ""
echo "  Account:  ed76cd33-b15e-49c5-a0b1-c4432286092d"
echo "  Phone:    +254713058775"
echo "  Balance:  KES 10,000"
echo "  PIN:      1234"
echo "  Earnings: 3 sample records"
echo "  Loan:     KES 5,000 (pending)"
echo "  Insurance: 1 active policy"
echo ""
echo "[i] Press Ctrl+C to stop the sandbox"
echo ""
echo "─────────────────────────────────────────────────"

# Trap Ctrl+C
trap 'echo ""; echo "[i] Sandbox stopped."; exit 0' INT TERM

# Run the sandbox
go run cmd/sandbox/main.go
