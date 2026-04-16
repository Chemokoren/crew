.PHONY: help dev build run test lint migrate-up migrate-down migrate-create clean

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Development ---

dev: ## Run server with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

run: ## Run server directly
	go run cmd/server/main.go

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/amy-mis cmd/server/main.go

# --- Testing ---

test: ## Run all tests
	go test ./... -v -race -cover

test-coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# --- Linting ---

lint: ## Run golangci-lint
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

# --- Database Migrations ---

MIGRATE_CMD = migrate -path migrations -database "$(DATABASE_URL)"

migrate-up: ## Run all pending migrations
	$(MIGRATE_CMD) up

migrate-down: ## Roll back the last migration
	$(MIGRATE_CMD) down 1

migrate-create: ## Create new migration (usage: make migrate-create name=create_users)
	migrate create -ext sql -dir migrations -seq $(name)

migrate-status: ## Show migration status
	$(MIGRATE_CMD) version

# --- Dependencies ---

deps: ## Download and tidy Go dependencies
	go mod download
	go mod tidy

# --- Docker ---

docker-build: ## Build Docker image
	docker build -t amy-mis:latest .

docker-up: ## Start all services (PostgreSQL, Redis, MinIO, app)
	docker compose up -d

docker-down: ## Stop all services
	docker compose down

docker-logs: ## Tail all service logs
	docker compose logs -f

# --- Utilities ---

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

seed: ## Run database seed script
	go run scripts/seed.go

swagger: ## Generate Swagger documentation
	@if command -v swag > /dev/null; then \
		swag init -g cmd/server/main.go -o docs/; \
	else \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		swag init -g cmd/server/main.go -o docs/; \
	fi
