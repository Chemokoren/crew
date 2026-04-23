# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy dependency files first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary — static, stripped
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/ussd-gateway cmd/server/main.go

# --- Runtime stage (minimal) ---
FROM alpine:3.20

WORKDIR /app

# Runtime dependencies only
RUN apk add --no-cache ca-certificates tzdata

# Copy binary
COPY --from=builder /bin/ussd-gateway /app/ussd-gateway

# Non-root user (security)
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup
USER appuser

EXPOSE 8090

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8090/health || exit 1

ENTRYPOINT ["/app/ussd-gateway"]
